package vercel_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_FeatureFlagConfigResource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))
	key := fmt.Sprintf("checkout-%s", projectSuffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckFeatureFlagDeleted(testClient(t), "vercel_feature_flag_definition.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagConfigResourceConfig(projectSuffix, key, "treatment", false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagExists(testClient(t), "vercel_feature_flag_definition.test"),
					resource.TestCheckResourceAttrSet("vercel_feature_flag_config.test", "id"),
					resource.TestCheckResourceAttr("vercel_feature_flag_config.test", "preview.default_variant_id", "treatment"),
					resource.TestCheckResourceAttr("vercel_feature_flag_config.test", "development.enabled", "false"),
				),
			},
			{
				Config: cfg(testAccFeatureFlagConfigResourceConfig(projectSuffix, key, "control", true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagExists(testClient(t), "vercel_feature_flag_definition.test"),
					resource.TestCheckResourceAttr("vercel_feature_flag_config.test", "preview.default_variant_id", "control"),
					resource.TestCheckResourceAttr("vercel_feature_flag_config.test", "development.enabled", "true"),
				),
			},
			{
				ResourceName:      "vercel_feature_flag_config.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getFeatureFlagImportID("vercel_feature_flag_config.test"),
			},
		},
	})
}

func testAccFeatureFlagConfigResourceConfig(projectSuffix, key, previewDefault string, developmentEnabled bool) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-config-%[1]s"
}

resource "vercel_feature_flag_definition" "test" {
  project_id = vercel_project.test.id
  key        = "%[2]s"
  kind       = "string"
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

resource "vercel_feature_flag_config" "test" {
  project_id = vercel_project.test.id
  flag_id    = vercel_feature_flag_definition.test.id

  production = {
    enabled             = true
    default_variant_id  = "control"
    disabled_variant_id = "control"
  }

  preview = {
    enabled             = true
    default_variant_id  = "%[3]s"
    disabled_variant_id = "control"
  }

  development = {
    enabled             = %[4]t
    default_variant_id  = "treatment"
    disabled_variant_id = "control"
  }
}
`, projectSuffix, key, previewDefault, developmentEnabled)
}
