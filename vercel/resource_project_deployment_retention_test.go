package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccProjectDeploymentRetentionExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetDeploymentRetention(context.TODO(), rs.Primary.Attributes["project_id"], teamID)
		return err
	}
}

func TestAcc_ProjectDeploymentRetention(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy("vercel_project.example", testTeam()),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDeploymentRetentionsConfig(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists("vercel_project_deployment_retention.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "1m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1y"),
				),
			},
			{
				Config: testAccProjectDeploymentRetentionsConfigUpdated(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists("vercel_project_deployment_retention.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "6m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1m"),
				),
			},
		},
	})
}

func testAccProjectDeploymentRetentionsConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[3]s

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_deployment_retention" "example" {
	project_id = vercel_project.example.id
	%[3]s
	expiration_preview    = "1m"
	expiration_production = "2m"
	expiration_canceled   = "3m"
	expiration_errored    = "1y"
}
`, projectName, testGithubRepo(), teamIDConfig())
}

func testAccProjectDeploymentRetentionsConfigUpdated(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
	%[3]s

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_deployment_retention" "example" {
	project_id = vercel_project.example.id
	%[3]s
	expiration_preview    = "2m"
	expiration_production = "3m"
	expiration_canceled   = "6m"
	expiration_errored    = "1m"
}
`, projectName, testGithubRepo(), teamIDConfig())
}
