package vercel_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func testAccDeploymentExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no DeploymentID is set")
		}

		_, err := testClient().GetDeployment(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func contains(items []string, i string) bool {
	for _, j := range items {
		if j == i {
			return true
		}
	}
	return false
}

func testAccEnvironmentSet(n, teamID string, envs ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no DeploymentID is set")
		}

		dpl, err := testClient().GetDeployment(context.TODO(), rs.Primary.ID, teamID)
		if err != nil {
			return err
		}

		for _, e := range envs {
			if !contains(dpl.Build.Environment, e) {
				things := strings.Join(dpl.Build.Environment, ",")
				return fmt.Errorf("Deployment should include environment variable %s, but only had '%s'", e, things)
			}
		}

		return nil
	}
}

func noopDestroyCheck(*terraform.State) error {
	return nil
}

func TestAcc_Deployment(t *testing.T) {
	testAccDeployment(t, "")
}

func TestAcc_DeploymentWithTeamID(t *testing.T) {
	testAccDeployment(t, testTeam())
}

func TestAcc_DeploymentWithEnvironment(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(projectSuffix, "", `environment = {
                    FOO = "baz",
                    BAR = "qux"
                }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists("vercel_deployment.test", ""),
					testAccEnvironmentSet("vercel_deployment.test", "", "FOO", "BAR"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "environment.FOO", "baz"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "environment.BAR", "qux"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithProjectSettings(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(projectSuffix, "", `project_settings = {
                    output_directory = ".",
                    # build command is commented out until a later point, as it is causing issues
                    # build_command = "echo 'wat'"
                }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists("vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "project_settings.output_directory", "."),
					// resource.TestCheckResourceAttr("vercel_deployment.test", "project_settings.build_command", "npm run build"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithRootDirectoryOverride(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRootDirectoryOverride(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists("vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithPathPrefix(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRootDirectoryWithPathPrefix(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists("vercel_deployment.test", ""),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithDeleteOnDestroy(t *testing.T) {
	t.Parallel()
	projectSuffix := acctest.RandString(16)
	extraConfig := "delete_on_destroy = true"
	deploymentID := ""
	storeDeploymentId := func(n string, did *string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources[n]
			if !ok {
				return fmt.Errorf("not found: %s", n)
			}
			*did = rs.Primary.ID
			return nil
		}
	}
	testDeploymentGone := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			_, err := testClient().GetDeployment(context.TODO(), deploymentID, "")
			if err == nil {
				return fmt.Errorf("expected not_found error, but got no error")
			}

			var apiErr client.APIError
			if err == nil {
				return fmt.Errorf("Found deployment but expected it to have been deleted")
			}
			if err != nil && errors.As(err, &apiErr) {
				if apiErr.StatusCode == 404 {
					return nil
				}
				return fmt.Errorf("Unexpected error checking for deleted deployment: %s", apiErr)
			}
			return err
		}
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(projectSuffix, "", extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists("vercel_deployment.test", ""),
					storeDeploymentId("vercel_deployment.test", &deploymentID),
				),
			},
			{
				Config: testAccDeploymentConfigWithNoDeployment(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testDeploymentGone(),
				),
			},
		},
	})

}

func testAccDeployment(t *testing.T, tid string) {
	projectSuffix := acctest.RandString(16)
	extraConfig := ""
	testTeamID := resource.TestCheckNoResourceAttr("vercel_deployment.test", "team_id")
	if tid != "" {
		extraConfig = fmt.Sprintf(`team_id = "%s"`, tid)
		testTeamID = resource.TestCheckResourceAttr("vercel_deployment.test", "team_id", tid)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(projectSuffix, extraConfig, extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testTeamID,
					testAccDeploymentExists("vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithGitSource(t *testing.T) {
	tests := map[string]string{
		"personal scope": "",
		"team scope":     testTeam(),
	}

	for name, teamID := range tests {
		t.Run(name, func(t *testing.T) {
			extraConfig := ""
			projectSuffix := acctest.RandString(16)
			if teamID != "" {
				extraConfig = fmt.Sprintf(`team_id = "%s"`, teamID)
			}
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				CheckDestroy:             noopDestroyCheck,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccDeployFromGitSource(projectSuffix, extraConfig),
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccDeploymentExists("vercel_deployment.bitbucket", teamID),
							testAccDeploymentExists("vercel_deployment.gitlab", teamID),
							testAccDeploymentExists("vercel_deployment.github", teamID),
						),
					},
				},
			})
		})
	}
}

func testAccDeploymentConfigWithNoDeployment(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%s"
  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}
`, projectSuffix)
}

func testAccDeploymentConfig(projectSuffix, projectExtras, deploymentExtras string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%s"
  %s
  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}

data "vercel_file" "index" {
    path = "example/index.html"
}

resource "vercel_deployment" "test" {
  %s
  project_id = vercel_project.test.id

  files      = data.vercel_file.index.file
  production = true
}
`, projectSuffix, projectExtras, deploymentExtras)
}

func testAccRootDirectoryOverride(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%s"
}

data "vercel_file" "index" {
    path = "../vercel/example/index.html"
}

resource "vercel_deployment" "test" {
  project_id = vercel_project.test.id
  files      = data.vercel_file.index.file
  production = true
  project_settings = {
      root_directory = "vercel/example"
  }
}`, projectSuffix)
}

func testAccRootDirectoryWithPathPrefix(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%s"
}

data "vercel_file" "index" {
    path = "../vercel/example/index.html"
}

resource "vercel_deployment" "test" {
  project_id    = vercel_project.test.id
  files         = data.vercel_file.index.file
  path_prefix   = "../vercel/example"
}`, projectSuffix)
}

func testAccDeployFromGitSource(projectSuffix, extras string) string {
	return fmt.Sprintf(`
resource "vercel_project" "github" {
  name = "test-acc-deployment-%[1]s-github"
  git_repository = {
      type = "github"
      repo = "%[2]s"
  }
  %[5]s
}
resource "vercel_project" "gitlab" {
  name = "test-acc-deployment-%[1]s-gitlab"
  git_repository = {
      type = "gitlab"
      repo = "%[3]s"
  }
  %[5]s
}
resource "vercel_project" "bitbucket" {
  name = "test-acc-deployment-%[1]s-bitbucket"
  git_repository = {
      type = "bitbucket"
      repo = "%[4]s"
  }
  %[5]s
}
resource "vercel_deployment" "github" {
  project_id = vercel_project.github.id
  ref        = "main"
  %[5]s
}
resource "vercel_deployment" "gitlab" {
  project_id = vercel_project.gitlab.id
  ref        = "main"
  %[5]s
}
resource "vercel_deployment" "bitbucket" {
  project_id = vercel_project.bitbucket.id
  ref        = "main"
  %[5]s
}
`, projectSuffix, testGithubRepo(), testGitlabRepo(), testBitbucketRepo(), extras)
}
