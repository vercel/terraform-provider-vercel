package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"

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
			return fmt.Errorf("no projectID is set")
		}

		c := client.New(os.Getenv("VERCEL_API_TOKEN"))
		_, err := c.GetDeployment(context.TODO(), rs.Primary.ID, teamID)
		return err
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

func TestAcc_DeploymentWithProjectSettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig("", `project_settings = {
                    output_directory = "."
                    dev_command = "npm run dev"
                }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists("vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "project_settings.output_directory", "."),
					resource.TestCheckResourceAttr("vercel_deployment.test", "project_settings.dev_command", "npm run dev"),
				),
			},
		},
	})
}

func testAccDeployment(t *testing.T, tid string) {
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
				Config: testAccDeploymentConfig(extraConfig, extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testTeamID,
					testAccDeploymentExists("vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
				),
			},
		},
	})
}

func testAccDeploymentConfig(projectExtras, deploymentExtras string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-one"
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

  files         = data.vercel_file.index.file
  production = true
}
`, projectExtras, deploymentExtras)
}
