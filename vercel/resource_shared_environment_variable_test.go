package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func testAccSharedEnvironmentVariableExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetSharedEnvironmentVariable(context.TODO(), teamID, rs.Primary.ID)
		return err
	}
}

func TestAcc_SharedEnvironmentVariables(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccSharedEnvironmentVariablesConfig(nameSuffix, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "key", fmt.Sprintf("test_acc_foo_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "value", "bar"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "comment", "Test comment for example"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "production"),

					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.sensitive_example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "key", fmt.Sprintf("test_acc_foo_sensitive_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "value", "bar"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "comment", "Test comment for sensitive example"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.sensitive_example", "target.*", "production"),

					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.no_comment_example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.no_comment_example", "key", fmt.Sprintf("test_acc_foo_no_comment_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.no_comment_example", "value", "baz"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.no_comment_example", "comment", ""),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.no_comment_example", "target.*", "production"),
				),
			},
			{
				Config: testAccSharedEnvironmentVariablesConfigUpdated(nameSuffix, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "key", fmt.Sprintf("test_acc_foo_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "value", "updated-bar"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "development"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "preview"),

					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "key", fmt.Sprintf("test_acc_foo_sensitive_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "value", "bar-updated"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.sensitive_example", "target.*", "production"),
				),
			},
			{
				ResourceName:      "vercel_shared_environment_variable.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getSharedEnvironmentVariableImportID("vercel_shared_environment_variable.example"),
			},
			{
				Config: testAccSharedEnvironmentVariablesConfigDeleted(nameSuffix, teamIDConfig(t)),
			},
		},
	})
}

func getSharedEnvironmentVariableImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func testAccSharedEnvironmentVariablesConfig(projectName string, teamIDConfig string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[2]s
}

resource "vercel_shared_environment_variable" "example" {
	%[2]s
	key         = "test_acc_foo_%[1]s"
	value       = "bar"
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
    comment     = "Test comment for example"
}

resource "vercel_shared_environment_variable" "sensitive_example" {
	%[2]s
	key         = "test_acc_foo_sensitive_%[1]s"
	value       = "bar"
    sensitive   = true
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
    comment     = "Test comment for sensitive example"
}

resource "vercel_shared_environment_variable" "no_comment_example" {
	%[2]s
	key         = "test_acc_foo_no_comment_%[1]s"
	value       = "baz"
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
}
`, projectName, teamIDConfig)
}

func testAccSharedEnvironmentVariablesConfigUpdated(projectName string, teamIDConfig string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[2]s
}

resource "vercel_project" "example2" {
	name = "test-acc-example-project-2-%[1]s"
	%[2]s
}

resource "vercel_shared_environment_variable" "example" {
	%[2]s
	key         = "test_acc_foo_%[1]s"
	value       = "updated-bar"
	target      = ["preview", "development"]
    project_ids = [
        vercel_project.example.id,
        vercel_project.example2.id
    ]
}

resource "vercel_shared_environment_variable" "sensitive_example" {
	%[2]s
	key         = "test_acc_foo_sensitive_%[1]s"
	value       = "bar-updated"
    sensitive   = true
	target      = ["production"]
    project_ids = [
        vercel_project.example.id,
        vercel_project.example2.id
    ]
}
`, projectName, teamIDConfig)
}

func testAccSharedEnvironmentVariablesConfigDeleted(projectName string, teamIDConfig string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[2]s
}

resource "vercel_project" "example2" {
	name = "test-acc-example-project-2-%[1]s"
	%[2]s
}
    `, projectName, teamIDConfig)
}
