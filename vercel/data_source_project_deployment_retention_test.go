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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy("vercel_project.example", testTeam()),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDeploymentRetentionDataSourceConfig(nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDeploymentRetentionExists("vercel_project_deployment_retention.example", testTeam()),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_preview", "1m"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_production", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_canceled", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example", "expiration_errored", "unlimited"),

					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example_2", "expiration_preview", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example_2", "expiration_production", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example_2", "expiration_canceled", "unlimited"),
					resource.TestCheckResourceAttr("data.vercel_project_deployment_retention.example_2", "expiration_errored", "unlimited"),
				),
			},
		},
	})
}

func testAccProjectDeploymentRetentionDataSourceConfig(projectName string) string {
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
    expiration_preview = "1m"
}

data "vercel_project_deployment_retention" "example" {
	project_id = vercel_project_deployment_retention.example.project_id
	%[3]s
}

resource "vercel_project" "example_2" {
	name = "test-acc-example-project-2-%[1]s"
	%[3]s

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

data "vercel_project_deployment_retention" "example_2" {
	project_id = vercel_project.example_2.id
	%[3]s
}
`, projectName, testGithubRepo(), teamIDConfig())
}
