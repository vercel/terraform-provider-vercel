package vercel_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

type projectEnvironmentVariableImportSecretPlanCheck struct{}

func (projectEnvironmentVariableImportSecretPlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}
	if strings.Contains(string(encoded), "test-acc-import-secret-") {
		resp.Error = fmt.Errorf("saved plan contains an environment variable secret")
	}
}

func TestAcc_ProjectEnvironmentVariable_WriteOnlyImport(t *testing.T) {
	// Setup creates resources through the API before Terraform imports them.
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests require TF_ACC")
	}

	for _, explicitTeam := range []bool{true, false} {
		name := "default_team"
		if explicitTeam {
			name = "explicit_team"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			c := testClient(t)
			teamID := testTeam(t)
			project, err := c.CreateProject(ctx, teamID, client.CreateProjectRequest{
				Name: "test-acc-wo-import-" + acctest.RandString(16),
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := c.DeleteProject(ctx, project.ID, teamID); err != nil && !client.NotFound(err) {
					t.Errorf("delete import test project: %s", err)
					return
				}
				if _, err := c.GetProject(ctx, project.ID, teamID); !client.NotFound(err) {
					t.Errorf("import test project still exists or deletion could not be verified: %v", err)
				}
			})

			const initial = "test-acc-import-secret-initial"
			const ignored = "test-acc-import-secret-ignored"
			const rotated = "test-acc-import-secret-rotated"
			env, err := c.CreateEnvironmentVariable(ctx, client.CreateEnvironmentVariableRequest{
				ProjectID: project.ID,
				TeamID:    teamID,
				EnvironmentVariable: client.EnvironmentVariableRequest{
					Key: "IMPORT_SECRET", Value: initial, Type: "encrypted",
					Target: []string{"development", "preview"},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			importID := project.ID + "/" + env.ID
			if explicitTeam {
				importID = teamID + "/" + importID
			}
			importID = "write-only:" + importID
			t.Setenv("TF_VAR_import_secret", initial)

			config := func(version int, comment string, includeImport bool) string {
				result := fmt.Sprintf(`
variable "import_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}
resource "vercel_project_environment_variable" "imported" {
  project_id       = %q
  key              = "IMPORT_SECRET"
  target           = ["development", "preview"]
  sensitive        = false
  value_wo         = var.import_secret
  value_wo_version = %d
  comment          = %q
}
`, project.ID, version, comment)
				if includeImport {
					result += fmt.Sprintf(`
import {
  to = vercel_project_environment_variable.imported
  id = %q
}
`, importID)
				}
				return cfg(result)
			}
			const address = "vercel_project_environment_variable.imported"
			check := func(want, comment string) resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "id", env.ID),
					resource.TestCheckNoResourceAttr(address, "value"),
					resource.TestCheckNoResourceAttr(address, "value_wo"),
					func(s *terraform.State) error {
						encoded, err := json.Marshal(s)
						if err != nil {
							return err
						}
						if strings.Contains(string(encoded), "test-acc-import-secret-") {
							return fmt.Errorf("saved state contains an environment variable secret")
						}
						remote, err := c.GetEnvironmentVariable(ctx, project.ID, teamID, env.ID)
						if err != nil {
							return err
						}
						if remote.Value != want {
							return fmt.Errorf("remote environment variable has an unexpected value")
						}
						if remote.Comment != comment {
							return fmt.Errorf("remote comment: got %q, want %q", remote.Comment, comment)
						}
						return nil
					},
				)
			}
			checks := resource.ConfigPlanChecks{
				PreApply:             []plancheck.PlanCheck{projectEnvironmentVariableImportSecretPlanCheck{}},
				PostApplyPreRefresh:  []plancheck.PlanCheck{projectEnvironmentVariableImportSecretPlanCheck{}},
				PostApplyPostRefresh: []plancheck.PlanCheck{projectEnvironmentVariableImportSecretPlanCheck{}},
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy: func(_ *terraform.State) error {
					_, err := c.GetEnvironmentVariable(ctx, project.ID, teamID, env.ID)
					if client.NotFound(err) {
						return nil
					}
					return fmt.Errorf("imported environment variable still exists or deletion could not be verified: %v", err)
				},
				Steps: []resource.TestStep{
					{Config: config(1, "", true), ConfigPlanChecks: checks, Check: check(initial, "")},
					{
						ResourceName: address, ImportState: true,
						ImportStateId: strings.TrimPrefix(importID, "write-only:"),
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 || states[0].ID != env.ID || states[0].Attributes["value"] != initial {
								return fmt.Errorf("legacy import did not retain the readable value and resource identity")
							}
							return nil
						},
					},
					{
						ResourceName: address, ImportState: true, ImportStateId: importID,
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 || states[0].ID != env.ID || states[0].Attributes["value"] != "" || states[0].Attributes["value_wo"] != "" {
								return fmt.Errorf("write-only CLI import did not omit the value or preserve resource identity")
							}
							return nil
						},
					},
					{Config: config(1, "", false), ConfigPlanChecks: checks, Check: check(initial, "")},
					{
						PreConfig: func() { t.Setenv("TF_VAR_import_secret", ignored) },
						// Force an update without changing the version to verify that PATCH preserves the secret.
						Config: config(1, "metadata update", false), ConfigPlanChecks: checks, Check: check(initial, "metadata update"),
					},
					{
						PreConfig: func() { t.Setenv("TF_VAR_import_secret", rotated) },
						Config:    config(2, "metadata update", false), ConfigPlanChecks: checks, Check: check(rotated, "metadata update"),
					},
					{RefreshState: true, Check: check(rotated, "metadata update")},
				},
			})
		})
	}
}
