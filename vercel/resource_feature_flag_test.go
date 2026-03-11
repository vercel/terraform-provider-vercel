package vercel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestAcc_FeatureFlagSegmentResource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))
	slug := fmt.Sprintf("internal-users-%s", projectSuffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckFeatureFlagSegmentDeleted(testClient(t), "vercel_feature_flag_segment.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagSegmentResourceConfig(projectSuffix, slug)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagSegmentExists(testClient(t), "vercel_feature_flag_segment.test"),
					resource.TestCheckResourceAttrSet("vercel_feature_flag_segment.test", "id"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "slug", slug),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "name", "Internal Users"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "hint", "user-email"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "include.#", "1"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "exclude.#", "0"),
				),
			},
			{
				Config: cfg(testAccFeatureFlagSegmentResourceConfigUpdated(projectSuffix, slug)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagSegmentExists(testClient(t), "vercel_feature_flag_segment.test"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "description", "Employee allowlist with contractor exclusions"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "hint", "user-id"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "include.#", "1"),
					resource.TestCheckResourceAttr("vercel_feature_flag_segment.test", "exclude.#", "1"),
				),
			},
			{
				ResourceName:      "vercel_feature_flag_segment.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getFeatureFlagSegmentImportID("vercel_feature_flag_segment.test"),
			},
		},
	})
}

func TestAcc_FeatureFlagSDKKeyResource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckFeatureFlagSDKKeyDeleted(testClient(t), "vercel_feature_flag_sdk_key.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagSDKKeyResourceConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagSDKKeyExists(testClient(t), "vercel_feature_flag_sdk_key.test"),
					resource.TestCheckResourceAttrSet("vercel_feature_flag_sdk_key.test", "id"),
					resource.TestCheckResourceAttr("vercel_feature_flag_sdk_key.test", "environment", "production"),
					resource.TestCheckResourceAttr("vercel_feature_flag_sdk_key.test", "type", "server"),
					resource.TestCheckResourceAttr("vercel_feature_flag_sdk_key.test", "label", "backend-sdk"),
					resource.TestCheckResourceAttrSet("vercel_feature_flag_sdk_key.test", "sdk_key"),
					resource.TestCheckResourceAttrSet("vercel_feature_flag_sdk_key.test", "connection_string"),
				),
			},
			{
				ResourceName:            "vercel_feature_flag_sdk_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"sdk_key", "connection_string"},
				ImportStateIdFunc:       getFeatureFlagSDKKeyImportID("vercel_feature_flag_sdk_key.test"),
			},
		},
	})
}

func getFeatureFlagImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf(
			"%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["project_id"],
			rs.Primary.ID,
		), nil
	}
}

func getFeatureFlagSegmentImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf(
			"%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["project_id"],
			rs.Primary.ID,
		), nil
	}
}

func getFeatureFlagSDKKeyImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf(
			"%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["project_id"],
			rs.Primary.ID,
		), nil
	}
}

func testCheckFeatureFlagExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetFeatureFlag(context.TODO(), client.GetFeatureFlagRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
			FlagID:    rs.Primary.ID,
		})
		return err
	}
}

func testCheckFeatureFlagDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetFeatureFlag(context.TODO(), client.GetFeatureFlagRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
			FlagID:    rs.Primary.ID,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted feature flag: %s", err)
		}
		return nil
	}
}

func testCheckFeatureFlagSegmentExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetFeatureFlagSegment(context.TODO(), client.GetFeatureFlagSegmentRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
			SegmentID: rs.Primary.ID,
		})
		return err
	}
}

func testCheckFeatureFlagSegmentDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetFeatureFlagSegment(context.TODO(), client.GetFeatureFlagSegmentRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
			SegmentID: rs.Primary.ID,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted feature flag segment: %s", err)
		}
		return nil
	}
}

func testCheckFeatureFlagSDKKeyExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		keys, err := testClient.ListFeatureFlagSDKKeys(context.TODO(), client.ListFeatureFlagSDKKeysRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
		})
		if err != nil {
			if client.NotFound(err) || strings.Contains(err.Error(), "project_not_found") {
				return nil
			}
			return err
		}

		for _, key := range keys {
			if key.HashKey == rs.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("feature flag sdk key %s not found", rs.Primary.ID)
	}
}

func testCheckFeatureFlagSDKKeyDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		keys, err := testClient.ListFeatureFlagSDKKeys(context.TODO(), client.ListFeatureFlagSDKKeysRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
		})
		if err != nil {
			if client.NotFound(err) || strings.Contains(err.Error(), "project_not_found") {
				return nil
			}
			return err
		}

		for _, key := range keys {
			if key.HashKey == rs.Primary.ID {
				return fmt.Errorf("feature flag sdk key %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}
}

func testAccFeatureFlagSegmentResourceConfig(projectSuffix, slug string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-segment-%[1]s"
}

resource "vercel_feature_flag_segment" "test" {
  project_id   = vercel_project.test.id
  slug         = "%[2]s"
  name         = "Internal Users"
  description  = "Employee allowlist"
  hint         = "user-email"
  include = [
    {
      entity    = "user"
      attribute = "email"
      values    = ["alice@example.com", "bob@example.com"]
    },
  ]
}
`, projectSuffix, slug)
}

func testAccFeatureFlagSegmentResourceConfigUpdated(projectSuffix, slug string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-segment-%[1]s"
}

resource "vercel_feature_flag_segment" "test" {
  project_id   = vercel_project.test.id
  slug         = "%[2]s"
  name         = "Internal Users"
  description  = "Employee allowlist with contractor exclusions"
  hint         = "user-id"
  include = [
    {
      entity    = "user"
      attribute = "email"
      values    = ["alice@example.com", "bob@example.com", "charlie@example.com"]
    },
  ]

  exclude = [
    {
      entity    = "user"
      attribute = "id"
      values    = ["contractor-123"]
    },
  ]
}
`, projectSuffix, slug)
}

func testAccFeatureFlagSDKKeyResourceConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-feature-flag-sdk-key-%[1]s"
}

resource "vercel_feature_flag_definition" "bootstrap" {
  project_id = vercel_project.test.id
  key        = "bootstrap-%[1]s"
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
  environment = "production"
  type        = "server"
  label       = "backend-sdk"

  depends_on = [vercel_feature_flag_definition.bootstrap]
}
`, projectSuffix)
}
