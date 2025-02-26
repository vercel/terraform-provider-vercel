package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testCheckSharedEnvironmentVariableProjectUnlinked(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		resp, err := testClient().GetSharedEnvironmentVariable(context.TODO(), teamID, rs.Primary.Attributes["shared_environment_variable_id"])
		if err != nil {
			return err
		}
		for _, p := range resp.ProjectIDs {
			if p == rs.Primary.Attributes["project_id"] {
				return fmt.Errorf("expected project to be unlinked from shared environment variable")
			}
		}

		return nil
	}
}

func testCheckSharedEnvironmentVariableProjectLinked(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		resp, err := testClient().GetSharedEnvironmentVariable(context.TODO(), teamID, rs.Primary.Attributes["shared_environment_variable_id"])
		if err != nil {
			return err
		}
		for _, p := range resp.ProjectIDs {
			if p == rs.Primary.Attributes["project_id"] {
				return nil
			}
		}

		return fmt.Errorf("expected project to be linked to shared environment variable")
	}
}

func TestAcc_SharedEnvironmentVariableProjectLink(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckSharedEnvironmentVariableProjectUnlinked("vercel_shared_environment_variable_project_link.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccSharedEnvironmentVariableProjectLink(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test", testTeam()),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable_project_link.test", "team_id", testTeam()),
				),
			},
		},
	})
}

func testAccSharedEnvironmentVariableProjectLink(name, team string) string {
	return fmt.Sprintf(`
data "vercel_endpoint_verification" "test" {
    %[2]s
}

data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
		target = ["production", "preview", "development"]
    %[2]s
}

resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
		enable_affected_projects_deployments = true
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test.id
    %[2]s
}
`, name, team)
}
