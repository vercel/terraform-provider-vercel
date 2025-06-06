package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_AccessGroupProjectDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAccessGroupProjectDataSource(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_access_group_project.test", "role", "ADMIN"),
				),
			},
		},
	})
}

func testAccAccessGroupProjectDataSource(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_access_group" "test" {
	name = "test-acc-%[1]s"
}

resource "vercel_access_group_project" "test" {
	access_group_id = vercel_access_group.test.id
	project_id = vercel_project.test.id
	role = "ADMIN"
}

data "vercel_access_group_project" "test" {
  access_group_id = vercel_access_group.test.id
  project_id = vercel_project.test.id
  depends_on = [
      vercel_access_group_project.test
  ]
}
`, name)
}
