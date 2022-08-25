package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// func testAccDNSRecordDestroy1(n, teamID string) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		rs, ok := s.RootModule().Resources[n]
// 		if !ok {
// 			return fmt.Errorf("not found: %s", n)
// 		}

// 		if rs.Primary.ID == "" {
// 			return fmt.Errorf("no ID is set")
// 		}

// 		_, err := testClient().GetDNSRecord(context.TODO(), rs.Primary.ID, teamID)

// 		var apiErr client.APIError
// 		if err == nil {
// 			return fmt.Errorf("Found project but expected it to have been deleted")
// 		}
// 		if err != nil && errors.As(err, &apiErr) {
// 			if apiErr.StatusCode == 404 {
// 				return nil
// 			}
// 			return fmt.Errorf("Unexpected error checking for deleted project: %s", apiErr)
// 		}

// 		return err
// 	}
// }

func testAccProjectEnvironmentVariableExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEnvironmentVariable(context.TODO(), rs.Primary.Attributes["project_id"], teamID, rs.Primary.ID)
		return err
	}
}

func testAccProjectEnvironmentVariablesDoNotExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		project, err := testClient().GetProject(context.TODO(), rs.Primary.Attributes["project_id"], teamID, true)

		if err != nil {
			return fmt.Errorf("could not fetch the project: %w", err)
		}

		if len(project.EnvironmentVariables) != 0 {
			return fmt.Errorf("project environment variables not deleted, they still exist")
		}

		return nil
	}
}

func TestAcc_ProjectEnvironmentVariables(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy("vercel_project.example", testTeam()),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectEnvironmentVariablesConfig(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariableExists("vercel_project_environment_variable.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "value", "bar"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example", "target.*", "production"),

					testAccProjectEnvironmentVariableExists("vercel_project_environment_variable.example_git_branch", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "value", "bar-staging"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_git_branch", "target.*", "preview"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "git_branch", "bla"),
				),
			},
			{
				Config: testAccProjectEnvironmentVariablesConfigUpdated(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariableExists("vercel_project_environment_variable.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "value", "bar-new"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example", "target.*", "production"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example", "target.*", "preview"),

					testAccProjectEnvironmentVariableExists("vercel_project_environment_variable.example_git_branch", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "value", "bar-staging"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_git_branch", "target.*", "preview"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "git_branch", "test-pr"),
				),
			},
			{
				ResourceName:      "vercel_project_environment_variable.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectEnvironmentVariableImportID("vercel_project_environment_variable.example"),
			},
			{
				ResourceName:      "vercel_project_environment_variable.example_git_branch",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectEnvironmentVariableImportID("vercel_project_environment_variable.example_git_branch"),
			},
			{
				Config: testAccProjectEnvironmentVariablesConfigDeleted(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariablesDoNotExists("vercel_project.example", testTeam()),
				),
			},
		},
	})
}

func getProjectEnvironmentVariableImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return fmt.Sprintf("%s/%s", rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
		}
		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
	}
}

func testAccProjectEnvironmentVariablesConfig(projectName string) string {
	return fmt.Sprintf(`
	resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[3]s

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
	}

	resource "vercel_project_environment_variable" "example" {
	project_id = vercel_project.example.id
	%[3]s
	key        = "foo"
	value      = "bar"
	target     = ["production"]
	}

	resource "vercel_project_environment_variable" "example_git_branch" {
	project_id = vercel_project.example.id
	%[3]s
	key        = "foo"
	value      = "bar-staging"
	target     = ["preview"]
	git_branch = "bla"
	}
`, projectName, testGithubRepo(), teamIDConfig())
}

func testAccProjectEnvironmentVariablesConfigUpdated(projectName string) string {
	return fmt.Sprintf(`
	resource "vercel_project" "example" {
		name = "test-acc-example-project-%[1]s"
		%[3]s

		git_repository = {
			type = "github"
			repo = "%[2]s"
		}
		}

		resource "vercel_project_environment_variable" "example" {
		project_id = vercel_project.example.id
		%[3]s
		key        = "foo"
		value      = "bar-new"
		target     = ["production", "preview"]
		}

		resource "vercel_project_environment_variable" "example_git_branch" {
		project_id = vercel_project.example.id
		%[3]s
		key        = "foo"
		value      = "bar-staging"
		target     = ["preview"]
		git_branch = "test-pr"
		}
`, projectName, testGithubRepo(), teamIDConfig())
}

func testAccProjectEnvironmentVariablesConfigDeleted(projectName string) string {
	return fmt.Sprintf(`
	resource "vercel_project" "example" {
		name = "test-acc-example-project-%[1]s"
		%[3]s

		git_repository = {
			type = "github"
			repo = "%[2]s"
		}
		}
`, projectName, testGithubRepo(), teamIDConfig())
}


