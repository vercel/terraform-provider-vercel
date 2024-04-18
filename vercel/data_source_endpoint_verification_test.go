package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_EndpointVerificationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEndpointVerificationDataSourceConfig(teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_endpoint_verification.test", "verification_code"),
				),
			},
		},
	})
}

func testAccEndpointVerificationDataSourceConfig(teamID string) string {
	return fmt.Sprintf(`
data "vercel_endpoint_verification" "test" {
    %[1]s
}
`, teamID)
}
