package vercel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_TeamDefaultDeploymentProtection(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test requires TF_ACC")
	}
	teamID, token := testTeam(t), apiToken(t)
	teamPath := "/v2/teams/" + teamID
	original, err := teamProtectionAPI(token, http.MethodGet, teamPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defaults, _ := original["defaultDeploymentProtection"].(map[string]any)
	originalSSO, exists := defaults["ssoProtection"]
	// PATCH cannot restore absence; restore its effective Standard Protection default.
	if !exists {
		originalSSO = map[string]any{"deploymentType": "all_except_custom_domains"}
	}
	t.Cleanup(func() {
		if _, err := teamProtectionAPI(token, http.MethodPatch, teamPath, map[string]any{
			"defaultDeploymentProtection": map[string]any{"ssoProtection": originalSSO},
		}); err != nil {
			t.Errorf("restore testing team deployment protection: %v", err)
			return
		}
		restored, err := teamProtectionAPI(token, http.MethodGet, teamPath, nil)
		if err != nil {
			t.Errorf("read restored testing team: %v", err)
			return
		}
		restoredDefaults, _ := restored["defaultDeploymentProtection"].(map[string]any)
		want, _ := json.Marshal(originalSSO)
		got, _ := json.Marshal(restoredDefaults["ssoProtection"])
		if !bytes.Equal(want, got) {
			t.Errorf("testing team SSO default was not restored")
		}
	})
	suffix := acctest.RandString(12)
	config := func(protection string, stage int) string {
		result := fmt.Sprintf(`
resource "vercel_team_config" "test" {
 id = %q
 default_deployment_protection = {
  vercel_authentication = { deployment_type = %q }
 }
}
data "vercel_team_config" "test" {
 id = vercel_team_config.test.id
 depends_on = [vercel_team_config.test]
}
resource "vercel_project" "inherited" {
 name = "test-acc-team-default-inherited-%s"
 git_comments = { on_commit = false, on_pull_request = false }
 depends_on = [vercel_team_config.test]
}
resource "vercel_project" "override" {
 name = "test-acc-team-default-override-%s"
 vercel_authentication = { deployment_type = "none" }
 depends_on = [vercel_team_config.test]
}
`, teamID, protection, suffix, suffix)
		if stage >= 2 {
			result += fmt.Sprintf(`
resource "vercel_project" "disabled" {
 name = "test-acc-team-default-disabled-%s"
 git_comments = { on_commit = false, on_pull_request = false }
 depends_on = [vercel_team_config.test]
}
`, suffix)
		}
		if stage >= 3 {
			result += fmt.Sprintf(`
resource "vercel_project" "standard" {
 name = "test-acc-team-default-standard-%s"
 git_comments = { on_commit = false, on_pull_request = false }
 depends_on = [vercel_team_config.test]
}
`, suffix)
		}
		return cfg(result)
	}
	attr := "default_deployment_protection.vercel_authentication.deployment_type"
	check := func(protection string, stage int) resource.TestCheckFunc {
		checks := []resource.TestCheckFunc{
			resource.TestCheckResourceAttr("vercel_team_config.test", attr, protection),
			resource.TestCheckResourceAttr("data.vercel_team_config.test", attr, protection),
			checkTeamProtectionAPI(token, teamPath, protection),
			checkProjectProtectionAPI(token, teamID, "vercel_project.inherited", "all"),
			checkProjectProtectionAPI(token, teamID, "vercel_project.override", "none"),
		}
		if stage >= 2 {
			checks = append(checks, checkProjectProtectionAPI(token, teamID, "vercel_project.disabled", "none"))
		}
		if stage >= 3 {
			checks = append(checks, checkProjectProtectionAPI(token, teamID, "vercel_project.standard", "all_except_custom_domains"))
		}
		return resource.ComposeAggregateTestCheckFunc(checks...)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			for _, name := range []string{"inherited", "override", "disabled", "standard"} {
				if err := testAccProjectDestroy(testClient(t), "vercel_project."+name, teamID)(s); err != nil {
					return err
				}
			}
			return checkTeamProtectionAPI(token, teamPath, "standard_protection_new")(s)
		},
		Steps: []resource.TestStep{
			{Config: config("all_deployments", 1), Check: check("all_deployments", 1)},
			{ResourceName: "vercel_team_config.test", ImportState: true, ImportStateId: teamID,
				ImportStateVerify: true, ImportStateVerifyIgnore: []string{"avatar"}},
			{Config: config("none", 2), Check: check("none", 2)},
			{Config: config("standard_protection_new", 3), Check: check("standard_protection_new", 3)},
			{PreConfig: func() {
				_, err := teamProtectionAPI(token, http.MethodPatch, teamPath, map[string]any{
					"defaultDeploymentProtection": map[string]any{"ssoProtection": map[string]any{"deploymentType": "all"}},
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := checkTeamProtectionAPI(token, teamPath, "all_deployments")(nil); err != nil {
					t.Fatal(err)
				}
			}, Config: config("standard_protection_new", 3), Check: check("standard_protection_new", 3)},
		},
	})
}

func teamProtectionAPI(token, method, path string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, "https://api.vercel.com"+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s returned HTTP %d", method, path, response.StatusCode)
	}
	var result map[string]any
	err = json.NewDecoder(response.Body).Decode(&result)
	return result, err
}

func checkTeamProtectionAPI(token, path, expected string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		team, err := teamProtectionAPI(token, http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		defaults, _ := team["defaultDeploymentProtection"].(map[string]any)
		raw := map[string]string{"all_deployments": "all", "standard_protection_new": "all_except_custom_domains", "none": "none"}[expected]
		return checkRawProtection(defaults["ssoProtection"], raw)
	}
}

func checkProjectProtectionAPI(token, teamID, name, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		project, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("project missing from state: %s", name)
		}
		result, err := teamProtectionAPI(token, http.MethodGet, "/v10/projects/"+project.Primary.ID+"?teamId="+teamID, nil)
		if err != nil {
			return err
		}
		return checkRawProtection(result["ssoProtection"], expected)
	}
}

func checkRawProtection(value any, expected string) error {
	if expected == "none" {
		if value != nil {
			return fmt.Errorf("expected disabled SSO protection, got %v", value)
		}
		return nil
	}
	protection, ok := value.(map[string]any)
	if !ok || protection["deploymentType"] != expected {
		return fmt.Errorf("expected SSO deployment type %s, got %v", expected, value)
	}
	return nil
}
