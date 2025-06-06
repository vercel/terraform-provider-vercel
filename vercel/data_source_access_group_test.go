package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_AccessGroupDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAccessGroupDataSource(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_access_group.test", "name", "test-acc-"+name),
				),
			},
		},
	})
}

func testAccAccessGroupDataSource(name string) string {
	return fmt.Sprintf(`
resource "vercel_access_group" "test" {
  name = "test-acc-%[1]s"
}

data "vercel_access_group" "test" {
	id = vercel_access_group.test.id
}
`, name)
}
