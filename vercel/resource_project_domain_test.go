package vercel_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func TestAcc_ProjectDomain(t *testing.T) {
	t.Parallel()
	testAccProjectDomain(t, "")
}

func TestAcc_ProjectDomainWithTeamID(t *testing.T) {
	t.Parallel()
	testAccProjectDomain(t, testTeam())
}

func testAccProjectDomainExists(n, teamID, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient().GetProjectDomain(context.TODO(), rs.Primary.ID, domain, teamID)
		return err
	}
}

func testAccProjectDomainDestroy(n, teamID, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient().GetProjectDomain(context.TODO(), rs.Primary.ID, domain, teamID)
		var apiErr client.APIError
		if err == nil {
			return fmt.Errorf("Found project domain but expected it to have been deleted")
		}
		if err != nil && errors.As(err, &apiErr) {
			if apiErr.StatusCode == 404 {
				return nil
			}
			return fmt.Errorf("Unexpected error checking for deleted project domain: %s", apiErr)
		}

		return err
	}
}

func testAccProjectDomain(t *testing.T, tid string) {
	extraConfig := ""
	testTeamID := resource.TestCheckNoResourceAttr("vercel_project.test", "team_id")
	if tid != "" {
		extraConfig = fmt.Sprintf(`team_id = "%s"`, tid)
		testTeamID = resource.TestCheckResourceAttr("vercel_project.test", "team_id", tid)
	}

	projectSuffix := acctest.RandString(16)
	domain := acctest.RandString(30) + ".vercel.app"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectDomainConfig(projectSuffix, domain, extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDomainExists("vercel_project.test", tid, domain),
					testTeamID,
					resource.TestCheckResourceAttr("vercel_project_domain.test", "domain", domain),
				),
			},
			// Update testing
			{
				Config: testAccProjectDomainConfigUpdated(projectSuffix, domain, extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect", "test-acc-domain.vercel.app"),
				),
			},
			// Redirect Update testing
			{
				Config: testAccProjectDomainConfigUpdated2(projectSuffix, domain, extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect", "test-acc-domain.vercel.app"),
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect_status_code", "307"),
				),
			},
			// Delete testing
			{
				Config: testAccProjectDomainConfigDeleted(projectSuffix, extraConfig),
				Check:  testAccProjectDomainDestroy("vercel_project.test", tid, domain),
			},
		},
	})
}

func testAccProjectDomainConfig(projectSuffix, domain, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
  %s
}

resource "vercel_project_domain" "test" {
  domain = "%s"
  %s
  project_id = vercel_project.test.id
}
`, projectSuffix, extra, domain, extra)
}

func testAccProjectDomainConfigUpdated(projectSuffix, domain, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
  %s
}

resource "vercel_project_domain" "test" {
  domain = "%s"
  project_id = vercel_project.test.id
  %s

  redirect = "test-acc-domain.vercel.app"
}
`, projectSuffix, extra, domain, extra)
}

func testAccProjectDomainConfigUpdated2(projectSuffix, domain, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
  %s
}

resource "vercel_project_domain" "test" {
  domain = "%s"
  project_id = vercel_project.test.id
  %s

  redirect = "test-acc-domain.vercel.app"
  redirect_status_code = 307
}
`, projectSuffix, extra, domain, extra)
}

func testAccProjectDomainConfigDeleted(projectSuffix, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
  %s
}
`, projectSuffix, extra)
}
