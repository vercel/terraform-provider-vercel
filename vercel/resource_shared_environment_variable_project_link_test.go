package vercel_test

import (
	"context"
	"fmt"
	"slices"
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
		if slices.Contains(resp.ProjectIDs, rs.Primary.Attributes["project_id"]) {
			return fmt.Errorf("expected project to be unlinked from shared environment variable %s", n)
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
		if !slices.Contains(resp.ProjectIDs, rs.Primary.Attributes["project_id"]) {
			return fmt.Errorf("expected project to be linked to shared environment variable %s", n)
		}
		return nil
	}
}

func TestAcc_SharedEnvironmentVariableProjectLink(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             func (s *terraform.State) error {
			if err := testCheckSharedEnvironmentVariableProjectUnlinked("vercel_shared_environment_variable_project_link.test0", testTeam())(s); err != nil {
				return err
			}
			if err := testCheckSharedEnvironmentVariableProjectUnlinked("vercel_shared_environment_variable_project_link.test1", testTeam())(s); err != nil {
				return err
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccSharedEnvironmentVariableProjectLinkSetup(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test0", testTeam()),
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test1", testTeam()),
				),
			},
			{
				Config: testAccSharedEnvironmentVariableProjectLinkAdd1(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test0", testTeam()),
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test1", testTeam()),
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test2", testTeam()),
				),
			},
			{
				Config: testAccSharedEnvironmentVariableProjectLinkDrop1(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test0", testTeam()),
					testCheckSharedEnvironmentVariableProjectLinked("vercel_shared_environment_variable_project_link.test1", testTeam()),
					testCheckSharedEnvironmentVariableProjectUnlinked("vercel_shared_environment_variable_project_link.test2", testTeam()),
				),
			},
		},
	})
}

func testAccSharedEnvironmentVariableProjectLinkSetup(name, team string) string {
	return fmt.Sprintf(`
data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
		target = ["production", "preview", "development"]
    %[2]s
}

resource "vercel_project" "test0" {
    name = "test-acc-shared-env-0-%[1]s"
    %[2]s
}

resource "vercel_project" "test1" {
    name = "test-acc-shared-env-1-%[1]s"
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test0" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test0.id
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test1" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
    %[2]s
}
`, name, team)
}

func testAccSharedEnvironmentVariableProjectLinkAdd1(name, team string) string {
	return fmt.Sprintf(`
data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
		target = ["production", "preview", "development"]
    %[2]s
}

resource "vercel_project" "test0" {
    name = "test-acc-shared-env-0-%[1]s"
    %[2]s
}

resource "vercel_project" "test1" {
    name = "test-acc-shared-env-1-%[1]s"
    %[2]s
}

resource "vercel_project" "test2" {
    name = "test-acc-shared-env-1-%[1]s"
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test0" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test0.id
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test1" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test2" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
    %[2]s
}
`, name, team)
}

func testAccSharedEnvironmentVariableProjectLinkDrop1(name, team string) string {
	return fmt.Sprintf(`
data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
		target = ["production", "preview", "development"]
    %[2]s
}

resource "vercel_project" "test0" {
    name = "test-acc-shared-env-0-%[1]s"
    %[2]s
}

resource "vercel_project" "test1" {
    name = "test-acc-shared-env-1-%[1]s"
    %[2]s
}

resource "vercel_project" "test2" {
    name = "test-acc-shared-env-1-%[1]s"
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test0" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test0.id
    %[2]s
}

resource "vercel_shared_environment_variable_project_link" "test1" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
    %[2]s
}
`, name, team)
}
