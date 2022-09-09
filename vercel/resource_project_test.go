package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func TestAcc_Project(t *testing.T) {
	testTeamID := resource.TestCheckNoResourceAttr("vercel_project.test", "team_id")
	if testTeam() != "" {
		testTeamID = resource.TestCheckResourceAttr("vercel_project.test", "team_id", testTeam())
	}
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", testTeam()),
		Steps: []resource.TestStep{
			// Ensure we get nice framework / serverless_function_region errors
			{
				Config: `
                    resource "vercel_project" "test" {
                        name = "foo"
                        serverless_function_region = "notexist"
                    }
                `,
				ExpectError: regexp.MustCompile("Invalid Serverless Function Region"),
			},
			{
				Config: `
                    resource "vercel_project" "test" {
                        name = "foo"
                        framework = "notexist"
                    }
                `,
				ExpectError: regexp.MustCompile("Invalid Framework"),
			},
			// Create and Read testing
			{
				Config: testAccProjectConfig(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", testTeam()),
					testTeamID,
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-project-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("vercel_project.test", "framework", "nextjs"),
					resource.TestCheckResourceAttr("vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "root_directory", "ui/src"),
					resource.TestCheckResourceAttr("vercel_project.test", "ignore_command", "echo 'wat'"),
					resource.TestCheckResourceAttr("vercel_project.test", "serverless_function_region", "syd1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "foo",
						"value": "bar",
					}),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "environment.0.target.*", "production"),
				),
			},
			// Update testing
			{
				Config: testAccProjectConfigUpdated(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-two-%s", projectSuffix)),
					resource.TestCheckNoResourceAttr("vercel_project.test", "build_command"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "bar",
						"value": "baz",
					}),
				),
			},
		},
	})
}

func TestAcc_ProjectAddingEnvAfterInitialCreation(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigWithoutEnv(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", testTeam()),
				),
			},
			{
				Config: testAccProjectConfigUpdated(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", testTeam()),
				),
			},
		},
	})
}

func TestAcc_ProjectWithGitRepository(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test_git", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigWithGitRepo(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test_git", testTeam()),
					resource.TestCheckResourceAttr("vercel_project.test_git", "git_repository.type", "github"),
					resource.TestCheckResourceAttr("vercel_project.test_git", "git_repository.repo", testGithubRepo()),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test_git", "environment.*", map[string]string{
						"key":        "foo",
						"value":      "bar",
						"git_branch": "staging",
					}),
				),
			},
			{
				Config: testAccProjectConfigWithGitRepoUpdated(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test_git", testTeam()),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test_git", "environment.*", map[string]string{
						"key":        "foo",
						"value":      "bar2",
						"git_branch": "staging",
					}),
				),
			},
		},
	})
}

func getProjectImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return rs.Primary.ID, nil
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func TestAcc_ProjectImport(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: projectConfigWithoutEnv(projectSuffix, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", testTeam()),
				),
			},
			{
				ResourceName:      "vercel_project.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectImportID("vercel_project.test"),
			},
		},
	})
}

func testAccProjectExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient().GetProject(context.TODO(), rs.Primary.ID, teamID, false)
		return err
	}
}

func testAccProjectDestroy(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient().GetProject(context.TODO(), rs.Primary.ID, teamID, false)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted project: %s", err)
		}

		return nil
	}
}

func testAccProjectConfigWithoutEnv(projectSuffix, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  %s
}
`, projectSuffix, teamID)
}

func testAccProjectConfigUpdated(projectSuffix, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  %s
  environment = [
    {
      key    = "two"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "baz"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "three"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "oh_no"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "bar"
      value  = "baz"
      target = ["production"]
    }
  ]
}
`, projectSuffix, teamID)
}

func testAccProjectConfigWithGitRepo(projectSuffix, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
  %s
  git_repository = {
    type = "github"
    repo = "%s"
  }
  environment = [
    {
      key        = "foo"
      value      = "bar"
      target     = ["preview"]
      git_branch = "staging"
    }
  ]
}
    `, projectSuffix, teamID, testGithubRepo())
}

func testAccProjectConfigWithGitRepoUpdated(projectSuffix, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
  %s
  public_source = false
  git_repository = {
    type = "github"
    repo = "%s"
  }
  environment = [
    {
      key        = "foo"
      value      = "bar2"
      target     = ["preview"]
      git_branch = "staging"
    }
  ]
}
    `, projectSuffix, teamID, testGithubRepo())
}

func projectConfigWithoutEnv(projectSuffix, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  %s
  build_command = "npm run build"
  dev_command = "npm run serve"
  ignore_command = "echo 'wat'"
  serverless_function_region = "syd1"
  framework = "nextjs"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
}
`, projectSuffix, teamID)
}

func testAccProjectConfig(projectSuffix, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  %s
  build_command = "npm run build"
  dev_command = "npm run serve"
  ignore_command = "echo 'wat'"
  serverless_function_region = "syd1"
  framework = "nextjs"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "two"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "three"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "baz"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "bar"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "oh_no"
      value  = "bar"
      target = ["production"]
    }
  ]
}
`, projectSuffix, teamID)
}
