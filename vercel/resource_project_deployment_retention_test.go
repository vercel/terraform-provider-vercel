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

func testAccProjectDeploymentRetentionExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetDeploymentRetention(context.TODO(), rs.Primary.Attributes["project_id"], teamID)
		return err
	}
}

func TestAcc_ProjectDeploymentRetention(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDeploymentRetentionsConfigWithMissingFields(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "6m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "1y"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "1m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "3m"),
				),
			},
			{
				Config: cfg(testAccProjectDeploymentRetentionsConfig(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),

					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "1w"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "1d"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1y"),
				),
			},
			{
				Config: cfg(testAccProjectDeploymentRetentionsConfigUpdated(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "6m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1m"),
				),
			},
		},
	})
}

func testAccProjectDeploymentRetentionsConfig(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_deployment_retention" "example" {
	project_id = vercel_project.example.id
	expiration_preview    = "1w"
	expiration_production = "2m"
	expiration_canceled   = "1d"
	expiration_errored    = "1y"
}
`, projectName, githubRepo)
}

func testAccProjectDeploymentRetentionsConfigUpdated(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_deployment_retention" "example" {
	project_id = vercel_project.example.id
	expiration_preview    = "2m"
	expiration_production = "3m"
	expiration_canceled   = "6m"
	expiration_errored    = "1m"
}
`, projectName, githubRepo)
}

func testAccProjectDeploymentRetentionsConfigWithMissingFields(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_deployment_retention" "example" {
	project_id = vercel_project.example.id
}
`, projectName, githubRepo)
}
