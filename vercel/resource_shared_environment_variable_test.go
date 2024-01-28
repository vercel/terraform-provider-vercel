package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccSharedEnvironmentVariableExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetSharedEnvironmentVariable(context.TODO(), teamID, rs.Primary.ID)
		return err
	}
}

func testAccSharedEnvironmentVariableDoesNotExist(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetSharedEnvironmentVariable(context.TODO(), teamID, rs.Primary.ID)

		if err != nil {
			return nil
		}
		return fmt.Errorf("expected an error, but got none")
	}
}

func TestAcc_SharedEnvironmentVariables(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy("vercel_project.example", testTeam()),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccSharedEnvironmentVariablesConfig(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists("vercel_shared_environment_variable.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "value", "bar"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "production"),
				),
			},
			{
				Config: testAccSharedEnvironmentVariablesConfigUpdated(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists("vercel_shared_environment_variable.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "value", "updated-bar"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "development"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "preview"),
				),
			},
			{
				Config: testAccSharedEnvironmentVariablesConfigUpdated(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariableExists("vercel_project_environment_variable.example_sensitive", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "value", "bar-production"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_sensitive", "target.*", "production"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "sensitive", "true"),
				),
			},
			{
				ResourceName:      "vercel_shared_environment_variable.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getSharedEnvironmentVariableImportID("vercel_shared_environment_variable.example"),
			},
			{
				ResourceName:      "vercel_project_environment_variable.example_sensitive",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectEnvironmentVariableImportID("vercel_project_environment_variable.example_sensitive"),
			},
			{
				Config: testAccSharedEnvironmentVariablesConfigDeleted(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableDoesNotExist("vercel_project.example", testTeam()),
				),
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

func testAccSharedEnvironmentVariablesConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[2]s
}

resource "vercel_shared_environment_variable" "example" {
	%[2]s
	key         = "foo"
	value       = "bar"
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
}
`, projectName, teamIDConfig())
}

func testAccSharedEnvironmentVariablesConfigUpdated(projectName string) string {
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
	key         = "foo"
	value       = "updated-bar"
	target      = ["preview", "development"]
    project_ids = [
        vercel_project.example.id,
        vercel_project.example2.id
    ]
}
`, projectName, teamIDConfig())
}

func testAccSharedEnvironmentVariablesConfigDeleted(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[2]s
}

resource "vercel_project" "example2" {
	name = "test-acc-example-project-2-%[1]s"
	%[2]s
}
    `, projectName, teamIDConfig())
}
