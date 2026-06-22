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

func TestAcc_Domain(t *testing.T) {
	domain := acctest.RandString(20) + ".terraform-ci"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccDomainDestroy(testClient(t), testTeam(t), domain),
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cfg(testAccDomainConfig(domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDomainExists(testClient(t), testTeam(t), domain),
					resource.TestCheckResourceAttr("vercel_domain.test", "name", domain),
					resource.TestCheckResourceAttrSet("vercel_domain.test", "id"),
				),
			},
			// Import testing
			{
				ResourceName:      "vercel_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return testTeam(t) + "/" + domain, nil
				},
			},
		},
	})
}

func testAccDomainExists(testClient *client.Client, teamID, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testClient.GetDomain(context.TODO(), domain, teamID)
		return err
	}
}

func testAccDomainDestroy(testClient *client.Client, teamID, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testClient.GetDomain(context.TODO(), domain, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted domain: %s", err)
		}
		return nil
	}
}

func testAccDomainConfig(domain string) string {
	return fmt.Sprintf(`
resource "vercel_domain" "test" {
  name = "%[1]s"
}
`, domain)
}
