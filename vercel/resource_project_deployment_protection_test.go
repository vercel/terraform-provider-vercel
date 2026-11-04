package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_ProjectDeploymentProtection(t *testing.T) {
	name := "test-acc-standalone-protection-" + acctest.RandString(12)
	config := func(scope string) string {
		protection := ""
		if scope != "" {
			protection = fmt.Sprintf(`resource "vercel_project_deployment_protection" "settings" {
 project_id = vercel_project.existing.id
 vercel_authentication = { deployment_type = %q }
}`, scope)
		}
		return cfg(fmt.Sprintf(`resource "vercel_project" "existing" {
 name = %q
 manage_deployment_protection = false
}
%s`, name, protection))
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.existing", testTeam(t)),
		Steps: []resource.TestStep{
			{Config: config("preview"), Check: resource.TestCheckResourceAttr("vercel_project_deployment_protection.settings", "vercel_authentication.deployment_type", "preview")},
			{ResourceName: "vercel_project_deployment_protection.settings", ImportState: true, ImportStateVerify: true, ImportStateIdFunc: func(s *terraform.State) (string, error) {
				return testTeam(t) + "/" + s.RootModule().Resources["vercel_project.existing"].Primary.ID, nil
			}},
			{Config: config("all"), Check: resource.TestCheckResourceAttr("vercel_project_deployment_protection.settings", "vercel_authentication.deployment_type", "all")},
			{Config: config(""), Check: func(s *terraform.State) error {
				project, err := testClient(t).GetProject(context.Background(), s.RootModule().Resources["vercel_project.existing"].Primary.ID, testTeam(t))
				if err != nil {
					return err
				}
				if project.VercelAuthentication != nil {
					return fmt.Errorf("authentication remained enabled after removing the standalone resource")
				}
				return nil
			}},
		},
	})
}
