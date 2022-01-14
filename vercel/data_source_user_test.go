package vercel_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_user.me", "username"),
					resource.TestCheckResourceAttrSet("data.vercel_user.me", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_user.me", "plan"),
				),
			},
		},
	})
}

func testAccUserConfig() string {
	return `
data "vercel_user" "me" {}
`
}
