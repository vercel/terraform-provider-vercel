package vercel_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

func testCheckSharedEnvironmentVariableProjectUnlinked(testClient *client.Client, envVarName, projectName, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		envVar, ok := s.RootModule().Resources[envVarName]
		if !ok {
			return fmt.Errorf("env var not found: %s", envVarName)
		}
		project, ok := s.RootModule().Resources[projectName]
		if !ok {
			return fmt.Errorf("project not found: %s", projectName)
		}

		resp, err := testClient.GetSharedEnvironmentVariable(context.TODO(), teamID, envVar.Primary.Attributes["id"])
		if err != nil {
			return err
		}
		if slices.Contains(resp.ProjectIDs, project.Primary.Attributes["id"]) {
			return fmt.Errorf("expected project to be unlinked to shared environment variable %s", projectName)
		}
		return nil
	}
}

func testCheckSharedEnvironmentVariableProjectLinked(testClient *client.Client, envVarName, projectName, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		envVar, ok := s.RootModule().Resources[envVarName]
		if !ok {
			return fmt.Errorf("env var not found: %s", envVarName)
		}
		project, ok := s.RootModule().Resources[projectName]
		if !ok {
			return fmt.Errorf("project not found: %s", projectName)
		}

		resp, err := testClient.GetSharedEnvironmentVariable(context.TODO(), teamID, envVar.Primary.Attributes["id"])
		if err != nil {
			return err
		}
		if !slices.Contains(resp.ProjectIDs, project.Primary.Attributes["id"]) {
			return fmt.Errorf("expected project to be linked to shared environment variable %s", projectName)
		}
		return nil
	}
}

func TestAcc_SharedEnvironmentVariableProjectLink(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckSharedEnvironmentVariableProjectUnlinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
			testCheckSharedEnvironmentVariableProjectUnlinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
			testCheckSharedEnvironmentVariableProjectUnlinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test2", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccSharedEnvironmentVariableProjectLinkSetup(name, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
				),
			},
			{
				Config: testAccSharedEnvironmentVariableProjectLinkAdd1(name, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test2", testTeam(t)),
				),
			},
			{
				Config: testAccSharedEnvironmentVariableProjectLinkDrop1(name, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectUnlinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test2", testTeam(t)),
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
    name = "test-acc-shared-env-2-%[1]s"
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
    project_id                     = vercel_project.test2.id
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
    name = "test-acc-shared-env-2-%[1]s"
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
