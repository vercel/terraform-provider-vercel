package vercel_test

import (
	"context"
	"fmt"
	"os"
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

		c := client.New(os.Getenv("VERCEL_API_TOKEN"))
		_, err := c.GetDeployment(context.TODO(), rs.Primary.ID, teamID)
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

		c := client.New(os.Getenv("VERCEL_API_TOKEN"))
		dpl, err := c.GetDeployment(context.TODO(), rs.Primary.ID, teamID)
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
	testAccDeployment(t, os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM"))
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
  files         = data.vercel_file.index.file
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
