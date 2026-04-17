package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectDelegatedProtectionDataSource(t *testing.T) {
	nameSuffix := acctest.RandString(8)
	resourceName := "vercel_project_delegated_protection.example"
	dataSourceName := "data.vercel_project_delegated_protection.example"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDelegatedProtectionDataSourceConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDelegatedProtectionExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttrPair(dataSourceName, "id", "vercel_project.example", "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "project_id", resourceName, "project_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "team_id", resourceName, "team_id"),
					resource.TestCheckResourceAttr(dataSourceName, "client_id", "client-id-initial"),
					resource.TestCheckResourceAttr(dataSourceName, "issuer", "https://vercel.com"),
					resource.TestCheckResourceAttr(dataSourceName, "deployment_type", "standard_protection_new"),
					resource.TestCheckResourceAttr(dataSourceName, "cookie_name", "_vercel_delegated_custom"),
					resource.TestCheckNoResourceAttr(dataSourceName, "client_secret"),
				),
			},
			{
				Config: cfg(testAccProjectDelegatedProtectionDataSourceConfigWithoutCookie(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDelegatedProtectionExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttrPair(dataSourceName, "project_id", resourceName, "project_id"),
					resource.TestCheckResourceAttr(dataSourceName, "client_id", "client-id-updated"),
					resource.TestCheckResourceAttr(dataSourceName, "issuer", "https://vercel.com"),
					resource.TestCheckResourceAttr(dataSourceName, "deployment_type", "standard_protection"),
					resource.TestCheckNoResourceAttr(dataSourceName, "cookie_name"),
					resource.TestCheckNoResourceAttr(dataSourceName, "client_secret"),
				),
			},
		},
	})
}

func testAccProjectDelegatedProtectionDataSourceConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-delegated-protection-ds-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_delegated_protection" "example" {
	project_id      = vercel_project.example.id
	client_id       = "client-id-initial"
	client_secret   = "client-secret-initial"
	cookie_name     = "_vercel_delegated_custom"
	deployment_type = "standard_protection_new"
	issuer          = "https://vercel.com"
}

data "vercel_project_delegated_protection" "example" {
	project_id = vercel_project_delegated_protection.example.project_id
}
`, nameSuffix)
}

func testAccProjectDelegatedProtectionDataSourceConfigWithoutCookie(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-delegated-protection-ds-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_delegated_protection" "example" {
	project_id      = vercel_project.example.id
	client_id       = "client-id-updated"
	client_secret   = "client-secret-updated"
	deployment_type = "standard_protection"
	issuer          = "https://vercel.com"
}

data "vercel_project_delegated_protection" "example" {
	project_id = vercel_project_delegated_protection.example.project_id
}
`, nameSuffix)
}
