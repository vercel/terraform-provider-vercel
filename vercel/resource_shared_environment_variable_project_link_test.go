package vercel_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
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
				Config: cfg(testAccSharedEnvironmentVariableProjectLinkSetup(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
				),
			},
			{
				Config: cfg(testAccSharedEnvironmentVariableProjectLinkAdd1(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test2", testTeam(t)),
				),
			},
			{
				Config: cfg(testAccSharedEnvironmentVariableProjectLinkDrop1(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test0", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectLinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test1", testTeam(t)),
					testCheckSharedEnvironmentVariableProjectUnlinked(testClient(t), "data.vercel_shared_environment_variable.test", "vercel_project.test2", testTeam(t)),
				),
			},
		},
	})
}

func testAccSharedEnvironmentVariableProjectLinkSetup(name string) string {
	return fmt.Sprintf(`
data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
	target = ["production", "preview", "development"]
}

resource "vercel_project" "test0" {
    name = "test-acc-shared-env-0-%[1]s"
}

resource "vercel_project" "test1" {
    name = "test-acc-shared-env-1-%[1]s"
}

resource "vercel_shared_environment_variable_project_link" "test0" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test0.id
}

resource "vercel_shared_environment_variable_project_link" "test1" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
}
`, name)
}

func testAccSharedEnvironmentVariableProjectLinkAdd1(name string) string {
	return fmt.Sprintf(`
data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
	target = ["production", "preview", "development"]
}

resource "vercel_project" "test0" {
    name = "test-acc-shared-env-0-%[1]s"
}

resource "vercel_project" "test1" {
    name = "test-acc-shared-env-1-%[1]s"
}

resource "vercel_project" "test2" {
    name = "test-acc-shared-env-2-%[1]s"
}

resource "vercel_shared_environment_variable_project_link" "test0" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test0.id
}

resource "vercel_shared_environment_variable_project_link" "test1" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
}

resource "vercel_shared_environment_variable_project_link" "test2" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test2.id
}
`, name)
}

func testAccSharedEnvironmentVariableProjectLinkDrop1(name string) string {
	return fmt.Sprintf(`
data "vercel_shared_environment_variable" "test" {
    key = "TEST_SHARED_ENV_VAR"
	target = ["production", "preview", "development"]
}

resource "vercel_project" "test0" {
    name = "test-acc-shared-env-0-%[1]s"
}

resource "vercel_project" "test1" {
    name = "test-acc-shared-env-1-%[1]s"
}

resource "vercel_project" "test2" {
    name = "test-acc-shared-env-2-%[1]s"
}

resource "vercel_shared_environment_variable_project_link" "test0" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test0.id
}

resource "vercel_shared_environment_variable_project_link" "test1" {
    shared_environment_variable_id = data.vercel_shared_environment_variable.test.id
    project_id                     = vercel_project.test1.id
}
`, name)
}
