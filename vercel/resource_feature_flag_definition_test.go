package vercel_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_FeatureFlagDefinitionResource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))
	key := fmt.Sprintf("checkout-%s", projectSuffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckFeatureFlagDeleted(testClient(t), "vercel_feature_flag_definition.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagDefinitionResourceConfig(projectSuffix, key, "Controls the checkout experience")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagExists(testClient(t), "vercel_feature_flag_definition.test"),
					resource.TestCheckResourceAttrSet("vercel_feature_flag_definition.test", "id"),
					resource.TestCheckResourceAttr("vercel_feature_flag_definition.test", "key", key),
					resource.TestCheckResourceAttr("vercel_feature_flag_definition.test", "kind", "string"),
					resource.TestCheckResourceAttr("vercel_feature_flag_definition.test", "description", "Controls the checkout experience"),
					resource.TestCheckResourceAttr("vercel_feature_flag_definition.test", "variant.#", "2"),
				),
			},
			{
				Config: cfg(testAccFeatureFlagDefinitionResourceConfig(projectSuffix, key, "Controls the checkout experience (updated)")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagExists(testClient(t), "vercel_feature_flag_definition.test"),
					resource.TestCheckResourceAttr("vercel_feature_flag_definition.test", "description", "Controls the checkout experience (updated)"),
					resource.TestCheckResourceAttr("vercel_feature_flag_definition.test", "variant.#", "2"),
				),
			},
			{
				ResourceName:      "vercel_feature_flag_definition.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getFeatureFlagImportID("vercel_feature_flag_definition.test"),
			},
		},
	})
}

func testAccFeatureFlagDefinitionResourceConfig(projectSuffix, key, description string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-definition-%[1]s"
}

resource "vercel_feature_flag_definition" "test" {
  project_id   = vercel_project.test.id
  key          = "%[2]s"
  description  = "%[3]s"
  kind         = "string"
  variant = [
    {
      id           = "control"
      label        = "Control"
      value_string = "control"
    },
    {
      id           = "treatment"
      label        = "Treatment"
      value_string = "treatment"
    },
  ]
}
`, projectSuffix, key, description)
}
