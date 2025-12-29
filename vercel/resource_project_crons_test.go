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

func testAccProjectCronsExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetProjectCrons(context.TODO(), rs.Primary.Attributes["project_id"], teamID)
		return err
	}
}

func TestAcc_ProjectCrons(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectCronsConfig(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectCronsExists(testClient(t), "vercel_project_crons.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_crons.example", "enabled", "false"),
				),
			},
			{
				Config: cfg(testAccProjectCronsConfigUpdated(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectCronsExists(testClient(t), "vercel_project_crons.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_crons.example", "enabled", "true"),
				),
			},
		},
	})
}

func testAccProjectCronsConfig(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_crons" "example" {
	project_id = vercel_project.example.id
	enabled    = false
}
`, projectName, githubRepo)
}

func testAccProjectCronsConfigUpdated(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_crons" "example" {
	project_id = vercel_project.example.id
	enabled    = true
}
`, projectName, githubRepo)
}
