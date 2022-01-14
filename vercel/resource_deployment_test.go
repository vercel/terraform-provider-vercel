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

func TestAccDeployment(t *testing.T) {
	testAccDeployment(t, "")
}

func TestAccDeploymentWithTeamID(t *testing.T) {
	testAccDeployment(t, os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM"))
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
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testTeamID,
					testAccDeploymentExists("vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
				),
			},
		},
	})
}

func testAccDeploymentConfig(extras string) string {
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
`, extras, extras)
}
