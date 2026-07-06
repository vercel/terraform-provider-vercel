package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_VCRRepositoryDataSource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccVCRRepositoryDataSource(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_vcr_repository.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_vcr_repository.test", "project_id"),
					resource.TestCheckResourceAttr("data.vercel_vcr_repository.test", "name", fmt.Sprintf("test-acc-%s", projectSuffix)),
					resource.TestCheckResourceAttrSet("data.vercel_vcr_repository.test", "url"),
				),
			},
		},
	})
}

func testAccVCRRepositoryDataSource(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-vcr-repo-%[1]s"
}

resource "vercel_vcr_repository" "test" {
  project_id = vercel_project.test.id
  name       = "test-acc-%[1]s"
}

data "vercel_vcr_repository" "test" {
  project_id = vercel_vcr_repository.test.project_id
  name       = vercel_vcr_repository.test.name
}
`, projectSuffix)
}
