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
			// First create the project
			{
				Config: cfg(testAccProjectConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
				),
			},
			// Then enable with initial configuration
			{
				Config: cfg(testAccProjectRollingReleasesConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.stages.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "20",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "50",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "100",
					}),
				),
			},
			// Then update to new configuration
			{
				Config: cfg(testAccProjectRollingReleasesConfigUpdate(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.stages.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "20",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "50",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "80",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "rolling_release.stages.*", map[string]string{
						"require_approval":  "true",
						"target_percentage": "100",
					}),
				),
			},
			// Finally disable
			{
				Config: cfg(testAccProjectRollingReleasesConfigOff(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.advancement_type", ""),
					resource.TestCheckResourceAttr(resourceName, "rolling_release.stages.#", "0"),
				),
			},
		},
	})
}

func testAccProjectRollingReleasesConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	depends_on = [vercel_project.example]
	rolling_release = {
		enabled          = true
		advancement_type = "manual-approval"
		stages = [
			{
				require_approval  = true
				target_percentage = 20
			},
			{
				require_approval  = true
				target_percentage = 50
			},
			{
				require_approval  = true
				target_percentage = 100
			}
		]
	}
	lifecycle {
		depends_on = [vercel_project.example]
	}
}
`, nameSuffix)
}

func testAccProjectRollingReleasesConfigUpdate(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	depends_on = [vercel_project.example]
	rolling_release = {
		enabled          = true
		advancement_type = "manual-approval"
		stages = [
			{
				require_approval  = true
				target_percentage = 20
			},
			{
				require_approval  = true
				target_percentage = 50
			},
			{
				require_approval  = true
				target_percentage = 80
			},
			{
				require_approval  = true
				target_percentage = 100
			}
		]
	}
	lifecycle {
		depends_on = [vercel_project.example]
	}
}
`, nameSuffix)
}

func testAccProjectRollingReleasesConfigOff(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	depends_on = [vercel_project.example]
	rolling_release = {
		enabled          = false
		advancement_type = ""
		stages          = []
	}
	lifecycle {
		depends_on = [vercel_project.example]
	}
}
`, nameSuffix)
}
