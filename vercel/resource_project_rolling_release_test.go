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

func testAccProjectRollingReleaseExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetRollingRelease(context.TODO(), rs.Primary.Attributes["project_id"], teamID)
		return err
	}
}

func TestAcc_ProjectRollingRelease(t *testing.T) {
	resourceName := "vercel_project_rolling_release.example"
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRollingReleasesConfig(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.advancement_type", "automatic"),

					resource.TestCheckResourceAttr(resourceName, "rolling_release.stages.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "1",
						"require_approval":  "false",
						"target_percentage": "15",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "1",
						"require_approval":  "false",
						"target_percentage": "50",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "1",
						"require_approval":  "false",
						"target_percentage": "100",
					}),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.1.id"),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.2.id"),
				),
			},
			{
				Config: cfg(testAccProjectRollingReleasesConfigUpdate(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.advancement_type", "automatic"),

					resource.TestCheckResourceAttr(resourceName, "rolling_release.stages.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "10",
						"require_approval":  "false",
						"target_percentage": "15",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "1",
						"require_approval":  "false",
						"target_percentage": "55",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "100",
						"require_approval":  "false",
						"target_percentage": "80",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"duration":          "1",
						"require_approval":  "false",
						"target_percentage": "100",
					}),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.1.id"),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.2.id"),
					resource.TestCheckResourceAttrSet(resourceName, "rolling_release.stages.3.id"),
				),
			},
			{
				Config: cfg(testAccProjectRollingReleasesConfigOff(nameSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.enabled", "false"),
				),
			},
		},
	})
}

func testAccProjectRollingReleasesConfig(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
  team_id    = vercel_project.example.team_id
  rolling_release = {
    enabled          = true
    advancement_type = "automatic"
    stages = [
      {
        duration          = 1
        require_approval  = false
        target_percentage = 15
      },
      {
        duration          = 1
        require_approval  = false
        target_percentage = 50
      },
      {
        duration          = 1
        require_approval  = false
        target_percentage = 100
      }
    ]
  }
}
`, projectName, githubRepo)
}

func testAccProjectRollingReleasesConfigUpdate(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
  team_id    = vercel_project.example.team_id
  rolling_release = {
    enabled          = true
    advancement_type = "automatic"
    stages = [
      {
        duration          = 10
        require_approval  = false
        target_percentage = 15
      },
      {
        duration          = 1
        require_approval  = false
        target_percentage = 55
      },
      {
        duration          = 100
        require_approval  = false
        target_percentage = 85
      },
      {
        duration          = 1
        require_approval  = false
        target_percentage = 100
      }
    ]
  }
}
`, projectName, githubRepo)
}

func testAccProjectRollingReleasesConfigOff(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"

	git_repository = {
		type = "github"
		repo = "%[2]s"
	}
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
  team_id    = vercel_project.example.team_id
  rolling_release = {
    enabled          = false
  }
}
`, projectName, githubRepo)
}
