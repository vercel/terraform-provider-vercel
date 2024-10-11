package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func TestAcc_ProjectDomain(t *testing.T) {
	t.Skip()
	testTeamID := resource.TestCheckNoResourceAttr("vercel_project.test", "team_id")
	if testTeam() != "" {
		testTeamID = resource.TestCheckResourceAttr("vercel_project.test", "team_id", testTeam())
	}

	projectSuffix := acctest.RandString(16)
	domain := acctest.RandString(30) + ".vercel.app"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Check error adding production domain
			{
				Config: testAccProjectDomainWithProductionGitBranch(projectSuffix, domain, teamIDConfig()),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll("the git_branch specified is the production branch. If you want to use this domain as a production domain, please omit the git_branch field.", " ", `\s*`),
				),
			},
			// Create and Read testing
			{
				Config: testAccProjectDomainConfig(projectSuffix, domain, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDomainExists("vercel_project.test", testTeam(), domain),
					testTeamID,
					resource.TestCheckResourceAttr("vercel_project_domain.test", "domain", domain),
				),
			},
			// Update testing
			{
				Config: testAccProjectDomainConfigUpdated(projectSuffix, domain, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect", "test-acc-domain.vercel.app"),
				),
			},
			// Redirect Update testing
			{
				Config: testAccProjectDomainConfigUpdated2(projectSuffix, domain, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect", "test-acc-domain.vercel.app"),
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect_status_code", "307"),
				),
			},
			// Delete testing
			{
				Config: testAccProjectDomainConfigDeleted(projectSuffix, teamIDConfig()),
				Check:  testAccProjectDomainDestroy("vercel_project.test", testTeam(), domain),
			},
		},
	})

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
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted deployment: %s", err)
		}
		return nil
	}
}

func testAccProjectDomainWithProductionGitBranch(projectSuffix, domain, teamIDConfig string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
  %[2]s
  git_repository = {
    type = "github"
    repo = "%[4]s"
  }
}

resource "vercel_project_domain" "test" {
  domain = "%[3]s"
  %[2]s
  git_branch = "main"
  project_id = vercel_project.test.id
}
`, projectSuffix, teamIDConfig, domain, testGithubRepo())
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
