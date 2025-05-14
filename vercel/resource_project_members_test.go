package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectMembers(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectMembersConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
				),
			},
			{
				Config: cfg(testAccProjectMembersConfigUpdated(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "2"),
				),
			},
			{
				Config: cfg(testAccProjectMembersConfigUpdatedAgain(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
				),
			},
		},
	})
}

func testAccProjectMembersConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [{
    email = "doug+test2@vercel.com"
    role  = "PROJECT_VIEWER"
  }]
}
`, projectSuffix)
}

func testAccProjectMembersConfigUpdated(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [{
      email = "doug+test2@vercel.com"
      role  = "PROJECT_DEVELOPER"
    },
    {
      email = "doug+test3@vercel.com"
      role  = "PROJECT_VIEWER"
    }
  ]
}
`, projectSuffix)
}

func testAccProjectMembersConfigUpdatedAgain(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [
    {
      email = "doug+test3@vercel.com"
      role  = "PROJECT_VIEWER"
    }
  ]
}
`, projectSuffix)
}
