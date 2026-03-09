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

func testAccBulkRedirectsLiveVersion(testClient *client.Client, projectID, teamID string) (*client.BulkRedirectVersion, error) {
	versions, err := testClient.GetBulkRedirectVersions(context.TODO(), projectID, teamID)
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		if version.IsLive {
			return &version, nil
		}
	}

	return nil, nil
}

func testAccBulkRedirectsExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		version, err := testAccBulkRedirectsLiveVersion(testClient, rs.Primary.Attributes["project_id"], teamID)
		if err != nil {
			return err
		}
		if version == nil {
			return fmt.Errorf("no live bulk redirects version found")
		}

		_, err = testClient.GetBulkRedirects(context.TODO(), client.GetBulkRedirectsRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    teamID,
			VersionID: version.ID,
		})
		return err
	}
}

func testAccBulkRedirectsEmpty(testClient *client.Client, projectResourceName, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[projectResourceName]
		if !ok {
			return fmt.Errorf("not found: %s", projectResourceName)
		}

		version, err := testAccBulkRedirectsLiveVersion(testClient, rs.Primary.ID, teamID)
		if err != nil {
			return err
		}
		if version == nil {
			return nil
		}

		redirects, err := testClient.GetBulkRedirects(context.TODO(), client.GetBulkRedirectsRequest{
			ProjectID: rs.Primary.ID,
			TeamID:    teamID,
			VersionID: version.ID,
		})
		if err != nil {
			return err
		}
		if len(redirects.Redirects) != 0 {
			return fmt.Errorf("expected no live redirects, got %d", len(redirects.Redirects))
		}

		return nil
	}
}

func TestAcc_BulkRedirects(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBulkRedirectsConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccBulkRedirectsExists(testClient(t), "vercel_bulk_redirects.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_bulk_redirects.example", "redirects.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_bulk_redirects.example", "redirects.*", map[string]string{
						"source":         "/old-path",
						"destination":    "/new-path",
						"status_code":    "307",
						"case_sensitive": "false",
						"query":          "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_bulk_redirects.example", "redirects.*", map[string]string{
						"source":         "/blog",
						"destination":    "https://example.com/blog",
						"status_code":    "308",
						"case_sensitive": "true",
						"query":          "true",
					}),
					resource.TestCheckResourceAttrSet("vercel_bulk_redirects.example", "version_id"),
				),
			},
			{
				Config: cfg(testAccBulkRedirectsConfigUpdated(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccBulkRedirectsExists(testClient(t), "vercel_bulk_redirects.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_bulk_redirects.example", "redirects.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_bulk_redirects.example", "redirects.*", map[string]string{
						"source":         "/docs",
						"destination":    "https://example.com/docs",
						"status_code":    "308",
						"case_sensitive": "false",
						"query":          "false",
					}),
					resource.TestCheckResourceAttrSet("vercel_bulk_redirects.example", "version_id"),
				),
			},
			{
				Config: cfg(testAccBulkRedirectsProjectOnlyConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccBulkRedirectsEmpty(testClient(t), "vercel_project.example", testTeam(t)),
				),
			},
		},
	})
}

func testAccBulkRedirectsConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_bulk_redirects" "example" {
	project_id = vercel_project.example.id
	redirects = [
		{
			source                = "/old-path"
			destination           = "/new-path"
			status_code           = 307
			case_sensitive        = false
			query                 = false
		},
		{
			source                = "/blog"
			destination           = "https://example.com/blog"
			status_code           = 308
			case_sensitive        = true
			query                 = true
		},
	]
}
`, projectName)
}

func testAccBulkRedirectsConfigUpdated(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_bulk_redirects" "example" {
	project_id = vercel_project.example.id
	redirects = [
		{
			source                = "/docs"
			destination           = "https://example.com/docs"
			status_code           = 308
			case_sensitive        = false
			query                 = false
		},
	]
}
`, projectName)
}

func testAccBulkRedirectsProjectOnlyConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}
`, projectName)
}
