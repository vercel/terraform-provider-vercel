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

func TestAcc_ProjectEnvironmentVariables(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
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
		},
	})
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
