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

func TestAcc_ProjectDeploymentRetentions(t *testing.T) {
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
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "1m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "1m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "1m"),

					testAccProjectDeploymentRetentionExists("vercel_project_deployment_retention.example_diff", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_preview", "1m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_production", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_canceled", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_errored", "6m"),
				),
			},
			{
				Config: testAccProjectDeploymentRetentionsConfigUpdated(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists("vercel_project_deployment_retention.example", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_preview", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_production", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_canceled", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example", "expiration_errored", "2m"),

					testAccProjectDeploymentRetentionExists("vercel_project_deployment_retention.example_diff", testTeam()),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_preview", "2m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_production", "3m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_canceled", "6m"),
					resource.TestCheckResourceAttr("vercel_project_deployment_retention.example_diff", "expiration_errored", "1y"),
				),
			},
			{
				ResourceName:      "vercel_project_deployment_retention.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectDeploymentRetentionImportID("vercel_project_deployment_retention.example"),
			},
			{
				ResourceName:      "vercel_project_deployment_retention.example_diff",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectDeploymentRetentionImportID("vercel_project_deployment_retention.example_diff"),
			},
		},
	})
}

func getProjectDeploymentRetentionImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return rs.Primary.Attributes["project_id"], nil
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"]), nil
	}
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

resource "vercel_project" "example_diff" {
	name = "test-acc-example-project-%[1]s-diff"
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
	expiration_production = "1m"
	expiration_canceled   = "1m"
	expiration_errored    = "1m"
}

resource "vercel_project_deployment_retention" "example_diff" {
	project_id = vercel_project.example_diff.id
	%[3]s
	expiration_preview    = "1m"
	expiration_production = "2m"
	expiration_canceled   = "3m"
	expiration_errored    = "6m"
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

resource "vercel_project" "example_diff" {
	name = "test-acc-example-project-%[1]s-diff"
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
	expiration_production = "2m"
	expiration_canceled   = "2m"
	expiration_errored    = "2m"
}

resource "vercel_project_deployment_retention" "example_diff" {
	project_id = vercel_project.example_diff.id
	%[3]s
	expiration_preview    = "2m"
	expiration_production = "3m"
	expiration_canceled   = "6m"
	expiration_errored    = "1y"
}
`, projectName, testGithubRepo(), teamIDConfig())
}
