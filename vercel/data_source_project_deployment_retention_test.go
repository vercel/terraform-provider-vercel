package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectDeploymentRetentionDataSource(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDeploymentRetentionDataSourceConfig(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists(testClient(t), "vercel_project_deployment_retention.example", testTeam(t)),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_preview", "1m"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_production", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_canceled", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_errored", "unlimited"),

					resource.TestCheckResourceAttrSet("data.vercel_project_deployment_retention.example_2", "expiration_preview"),
					resource.TestCheckResourceAttrSet("data.vercel_project_deployment_retention.example_2", "expiration_production"),
					resource.TestCheckResourceAttrSet("data.vercel_project_deployment_retention.example_2", "expiration_canceled"),
					resource.TestCheckResourceAttrSet("data.vercel_project_deployment_retention.example_2", "expiration_errored"),
				),
			},
		},
	})
}

func testAccProjectDeploymentRetentionDataSourceConfig(projectName string, githubRepo string) string {
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
    expiration_preview = "1m"
}

data "vercel_project_deployment_retention" "example" {
	project_id = vercel_project_deployment_retention.example.project_id
}

resource "vercel_project" "example_2" {
	name = "test-acc-example-project-2-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

data "vercel_project_deployment_retention" "example_2" {
	project_id = vercel_project.example_2.id
}
`, projectName, githubRepo)
}
