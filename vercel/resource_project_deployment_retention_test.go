package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
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
				Config: testAccProjectDeploymentRetentionsConfigWithMissingFields(nameSuffix, testGithubRepo(t), teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "unlimited"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "unlimited"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "unlimited"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "unlimited"),
				),
			},
			{
				Config: testAccProjectDeploymentRetentionsConfig(nameSuffix, testGithubRepo(t), teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "1m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1y"),
				),
			},
			{
				Config: testAccProjectDeploymentRetentionsConfigUpdated(nameSuffix, testGithubRepo(t), teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "6m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1m"),
				),
			},
			{
				Config: testAccProjectDeploymentRetentionsConfigAllUnlimited(nameSuffix, testGithubRepo(t), teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "unlimited"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "unlimited"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "unlimited"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "unlimited"),
				),
			},
		},
	})
}

func testAccProjectDeploymentRetentionsConfig(projectName string, githubRepo string, teamIDConfig string) string {
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
`, projectName, githubRepo, teamIDConfig)
}

func testAccProjectDeploymentRetentionsConfigUpdated(projectName string, githubRepo string, teamIDConfig string) string {
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
`, projectName, githubRepo, teamIDConfig)
}

func testAccProjectDeploymentRetentionsConfigAllUnlimited(projectName string, githubRepo string, teamIDConfig string) string {
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
	expiration_preview    = "unlimited"
	expiration_production = "unlimited"
	expiration_canceled   = "unlimited"
	expiration_errored    = "unlimited"
}
`, projectName, githubRepo, teamIDConfig)
}

func testAccProjectDeploymentRetentionsConfigWithMissingFields(projectName string, githubRepo string, teamIDConfig string) string {
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
}
`, projectName, githubRepo, teamIDConfig)
}
