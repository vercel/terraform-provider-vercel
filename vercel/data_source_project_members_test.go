package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectMembersDataSource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectMembersDataSourceConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("data.vercel_project_members.test", "members.#", "1"),
				),
			},
		},
	})
}

func testAccProjectMembersDataSourceConfig(projectSuffix string) string {
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

data "vercel_project_members" "test" {
  project_id = vercel_project_members.test.project_id
}
`, projectSuffix)
}
