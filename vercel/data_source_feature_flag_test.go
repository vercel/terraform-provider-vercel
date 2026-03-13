package vercel_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_FeatureFlagDataSource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))
	key := fmt.Sprintf("homepage-%s", projectSuffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagDataSourceConfig(projectSuffix, key)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.vercel_feature_flag.test", "id", "vercel_feature_flag_definition.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag.test", "key", key),
					resource.TestCheckResourceAttr("data.vercel_feature_flag.test", "kind", "boolean"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag.test", "archived", "false"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag.test", "production.default_variant_id", "on"),
				),
			},
		},
	})
}

func TestAcc_FeatureFlagSegmentDataSource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))
	slug := fmt.Sprintf("beta-users-%s", projectSuffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagSegmentDataSourceConfig(projectSuffix, slug)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.vercel_feature_flag_segment.test", "id", "vercel_feature_flag_segment.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag_segment.test", "slug", slug),
					resource.TestCheckResourceAttr("data.vercel_feature_flag_segment.test", "name", "Beta Users"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag_segment.test", "include.#", "1"),
				),
			},
		},
	})
}

func TestAcc_FeatureFlagSDKKeyDataSource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagSDKKeyDataSourceConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.vercel_feature_flag_sdk_key.test", "id", "vercel_feature_flag_sdk_key.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag_sdk_key.test", "environment", "preview"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag_sdk_key.test", "type", "client"),
					resource.TestCheckResourceAttr("data.vercel_feature_flag_sdk_key.test", "label", "web-sdk"),
				),
			},
		},
	})
}

func testAccFeatureFlagDataSourceConfig(projectSuffix, key string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-ds-%[1]s"
}

resource "vercel_feature_flag_definition" "test" {
  project_id = vercel_project.test.id
  key        = "%[2]s"
  kind       = "boolean"
  variant = [
    {
      id         = "off"
      value_bool = false
    },
    {
      id         = "on"
      value_bool = true
    },
  ]
}

resource "vercel_feature_flag_config" "test" {
  project_id = vercel_project.test.id
  flag_id    = vercel_feature_flag_definition.test.id

  production = {
    enabled             = true
    default_variant_id  = "on"
    disabled_variant_id = "off"
  }

  preview = {
    enabled             = true
    default_variant_id  = "off"
    disabled_variant_id = "off"
  }

  development = {
    enabled             = false
    default_variant_id  = "off"
    disabled_variant_id = "off"
  }
}

data "vercel_feature_flag" "test" {
  project_id = vercel_project.test.id
  key        = vercel_feature_flag_definition.test.key

  depends_on = [vercel_feature_flag_config.test]
}
`, projectSuffix, key)
}

func testAccFeatureFlagSegmentDataSourceConfig(projectSuffix, slug string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-segment-ds-%[1]s"
}

resource "vercel_feature_flag_segment" "test" {
  project_id = vercel_project.test.id
  slug       = "%[2]s"
  name       = "Beta Users"
  include = [
    {
      entity    = "user"
      attribute = "email"
      values    = ["beta@example.com"]
    },
  ]
}

data "vercel_feature_flag_segment" "test" {
  project_id = vercel_project.test.id
  slug       = vercel_feature_flag_segment.test.slug
}
`, projectSuffix, slug)
}

func testAccFeatureFlagSDKKeyDataSourceConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-sdk-key-ds-%[1]s"
}

resource "vercel_feature_flag_definition" "bootstrap" {
  project_id = vercel_project.test.id
  key        = "bootstrap-ds-%[1]s"
  kind       = "boolean"
  variant = [
    {
      id         = "off"
      value_bool = false
    },
    {
      id         = "on"
      value_bool = true
    },
  ]
}

resource "vercel_feature_flag_sdk_key" "test" {
  project_id  = vercel_project.test.id
  environment = "preview"
  type        = "client"
  label       = "web-sdk"

  depends_on = [vercel_feature_flag_definition.bootstrap]
}

data "vercel_feature_flag_sdk_key" "test" {
  project_id = vercel_project.test.id
  id         = vercel_feature_flag_sdk_key.test.id
}
`, projectSuffix)
}
