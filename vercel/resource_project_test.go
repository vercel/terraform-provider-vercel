package vercel_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func TestAcc_Project(t *testing.T) {
	tests := map[string]string{
		"personal scope": "",
		"team scope":     testTeam(),
	}

	for name, teamID := range tests {
		t.Run(name, func(t *testing.T) {
			extraConfig := ""
			testTeamID := resource.TestCheckNoResourceAttr("vercel_project.test", "team_id")
			if teamID != "" {
				extraConfig = fmt.Sprintf(`team_id = "%s"`, teamID)
				testTeamID = resource.TestCheckResourceAttr("vercel_project.test", "team_id", teamID)
			}
			projectSuffix := acctest.RandString(16)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccProjectDestroy("vercel_project.test", teamID),
				Steps: []resource.TestStep{
					// Create and Read testing
					{
						Config: testAccProjectConfig(projectSuffix, extraConfig),
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccProjectExists("vercel_project.test", teamID),
							testTeamID,
							resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-project-%s", projectSuffix)),
							resource.TestCheckResourceAttr("vercel_project.test", "build_command", "npm run build"),
							resource.TestCheckResourceAttr("vercel_project.test", "dev_command", "npm run serve"),
							resource.TestCheckResourceAttr("vercel_project.test", "framework", "nextjs"),
							resource.TestCheckResourceAttr("vercel_project.test", "install_command", "npm install"),
							resource.TestCheckResourceAttr("vercel_project.test", "output_directory", ".output"),
							resource.TestCheckResourceAttr("vercel_project.test", "public_source", "true"),
							resource.TestCheckResourceAttr("vercel_project.test", "root_directory", "ui/src"),
							resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
								"key":   "foo",
								"value": "bar",
							}),
							resource.TestCheckTypeSetElemAttr("vercel_project.test", "environment.0.target.*", "production"),
						),
					},
					// Update testing
					{
						Config: testAccProjectConfigUpdated(projectSuffix, extraConfig),
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
		})
	}
}

func TestAcc_ProjectAddingEnvAfterInitialCreation(t *testing.T) {
	t.Parallel()
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", ""),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigWithoutEnv(projectSuffix, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", ""),
				),
			},
			{
				Config: testAccProjectConfigUpdated(projectSuffix, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", ""),
				),
			},
		},
	})
}

func TestAcc_ProjectWithGitRepository(t *testing.T) {
	t.Parallel()
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test_git", ""),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigWithGitRepo(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test_git", ""),
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
				Config: testAccProjectConfigWithGitRepoUpdated(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test_git", ""),
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

func TestAcc_ProjectImport(t *testing.T) {
	t.Parallel()
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", ""),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(projectSuffix, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", ""),
				),
			},
			{
				ResourceName:      "vercel_project.test",
				ImportState:       true,
				ImportStateVerify: true,
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

		_, err := testClient().GetProject(context.TODO(), rs.Primary.ID, teamID)
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

		_, err := testClient().GetProject(context.TODO(), rs.Primary.ID, teamID)

		var apiErr client.APIError
		if err == nil {
			return fmt.Errorf("Found project but expected it to have been deleted")
		}
		if err != nil && errors.As(err, &apiErr) {
			if apiErr.StatusCode == 404 {
				return nil
			}
			return fmt.Errorf("Unexpected error checking for deleted project: %s", apiErr)
		}

		return err
	}
}

func testAccProjectConfigWithoutEnv(projectSuffix, extras string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  %s
}
`, projectSuffix, extras)
}

func testAccProjectConfigUpdated(projectSuffix, extras string) string {
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
`, projectSuffix, extras)
}

func testAccProjectConfigWithGitRepo(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
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
    `, projectSuffix, testGithubRepo())
}

func testAccProjectConfigWithGitRepoUpdated(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
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
    `, projectSuffix, testGithubRepo())
}

func testAccProjectConfig(projectSuffix, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  %s
  build_command = "npm run build"
  dev_command = "npm run serve"
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
`, projectSuffix, extra)
}
