package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func testAccProjectEnvironmentVariableExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetEnvironmentVariable(context.TODO(), rs.Primary.Attributes["project_id"], teamID, rs.Primary.ID)
		return err
	}
}

func testAccProjectEnvironmentVariableDoNotExist(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		envs, err := testClient.GetEnvironmentVariables(context.TODO(), rs.Primary.ID, teamID)
		if err != nil {
			return fmt.Errorf("could not fetch the project: %w", err)
		}

		if len(envs) != 0 {
			return fmt.Errorf("project environment variables not deleted, they still exist")
		}

		return nil
	}
}

func TestAcc_ProjectEnvironmentVariable(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectEnvironmentVariableConfig(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "value", "bar"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example", "target.*", "production"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "comment", "this is with a comment"),

					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example_git_branch", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "value", "bar-staging"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_git_branch", "target.*", "preview"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "git_branch", "production"),

					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example_sensitive", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "key", "foo_sensitive"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "value", "bar-sensitive"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_sensitive", "target.*", "production"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "sensitive", "true"),

					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example_not_sensitive", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_not_sensitive", "key", "foo_not_sensitive"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_not_sensitive", "value", "bar-not-sensitive"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_not_sensitive", "target.*", "production"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_not_sensitive", "sensitive", "false"),
				),
			},
			{
				Config: cfg(testAccProjectEnvironmentVariableConfigUpdated(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "value", "bar-new"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example", "target.*", "production"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example", "target.*", "preview"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example", "comment", "this is an updated comment"),

					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example_git_branch", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "key", "foo"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "value", "bar-staging"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_git_branch", "target.*", "preview"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_git_branch", "git_branch", "test"),

					testAccProjectEnvironmentVariableExists(testClient(t), "vercel_project_environment_variable.example_sensitive", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "key", "foo_sensitive"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "value", "bar-sensitive-updated"),
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.example_sensitive", "target.*", "production"),
					resource.TestCheckResourceAttr("vercel_project_environment_variable.example_sensitive", "sensitive", "true"),
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
				Config: cfg(testAccProjectEnvironmentVariableConfigDeleted(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectEnvironmentVariableDoNotExist(testClient(t), "vercel_project.example", testTeam(t)),
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

func testAccProjectEnvironmentVariableConfig(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_environment_variable" "example" {
	project_id = vercel_project.example.id
	key        = "foo"
	value      = "bar"
	target     = ["production"]
    comment    = "this is with a comment"
}

resource "vercel_project_environment_variable" "example_git_branch" {
	project_id = vercel_project.example.id
	key        = "foo"
	value      = "bar-staging"
	target     = ["preview"]
    git_branch = "production"
}

resource "vercel_project_environment_variable" "example_sensitive" {
	project_id = vercel_project.example.id
	key        = "foo_sensitive"
	value      = "bar-sensitive"
	target     = ["production"]
	sensitive  = true
}

resource "vercel_project_environment_variable" "example_not_sensitive" {
	project_id = vercel_project.example.id
	key        = "foo_not_sensitive"
	value      = "bar-not-sensitive"
	target     = ["production"]
	sensitive  = false
}
`, projectName, githubRepo)
}

func testAccProjectEnvironmentVariableConfigUpdated(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
    name = "test-acc-example-project-%[1]s"

    git_repository = {
        type = "github"
        repo = "%[2]s"
    }
}

resource "vercel_project_environment_variable" "example" {
    project_id = vercel_project.example.id
    key        = "foo"
    value      = "bar-new"
    target     = ["production", "preview"]
    comment    = "this is an updated comment"
}

resource "vercel_project_environment_variable" "example_git_branch" {
    project_id = vercel_project.example.id
    key        = "foo"
    value      = "bar-staging"
    target     = ["preview"]
    git_branch = "test"
}

resource "vercel_project_environment_variable" "example_sensitive" {
	project_id = vercel_project.example.id
	key        = "foo_sensitive"
	value      = "bar-sensitive-updated"
	target     = ["production"]
	sensitive  = true
}
`, projectName, githubRepo)
}

func testAccProjectEnvironmentVariableConfigDeleted(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
    name = "test-acc-example-project-%[1]s"

    git_repository = {
        type = "github"
        repo = "%[2]s"
    }
}
`, projectName, githubRepo)
}
