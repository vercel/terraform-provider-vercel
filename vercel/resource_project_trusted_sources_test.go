package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_ProjectTrustedSources(t *testing.T) {
	suffix := acctest.RandString(12)
	config := func(label string) string {
		trusted := ""
		if label != "" {
			trusted = fmt.Sprintf(`resource "vercel_project_trusted_sources" "settings" {
 project_id = vercel_project.existing.id
 projects = [{ project_id = vercel_project.caller.id, label = %q }]
}`, label)
		}
		return cfg(fmt.Sprintf(`resource "vercel_project" "existing" {
 name                   = "test-acc-trusted-sources-%[1]s"
 manage_trusted_sources = false
}
resource "vercel_project" "caller" {
 name = "test-acc-trusted-caller-%[1]s"
}
%[2]s`, suffix, trusted))
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.existing", testTeam(t)),
		Steps: []resource.TestStep{
			{Config: config("caller"), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("vercel_project_trusted_sources.settings", "projects.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("vercel_project_trusted_sources.settings", "projects.*", map[string]string{"label": "caller"}),
				resource.TestCheckNoResourceAttr("vercel_project.existing", "trusted_sources"),
			)},
			{ResourceName: "vercel_project_trusted_sources.settings", ImportState: true, ImportStateVerify: true, ImportStateIdFunc: func(s *terraform.State) (string, error) {
				return testTeam(t) + "/" + s.RootModule().Resources["vercel_project.existing"].Primary.ID, nil
			}},
			{Config: config("renamed"), Check: resource.TestCheckTypeSetElemNestedAttrs("vercel_project_trusted_sources.settings", "projects.*", map[string]string{"label": "renamed"})},
			{Config: config(""), Check: func(s *terraform.State) error {
				project, err := testClient(t).GetProject(context.Background(), s.RootModule().Resources["vercel_project.existing"].Primary.ID, testTeam(t))
				if err != nil {
					return err
				}
				if project.TrustedSources != nil && len(project.TrustedSources.Projects) > 0 {
					return fmt.Errorf("trusted projects remained after removing the standalone resource")
				}
				return nil
			}},
		},
	})
}
