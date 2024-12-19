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
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

func TestAcc_ProjectDomain(t *testing.T) {
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
				Config: testAccProjectDomainWithProductionGitBranch(projectSuffix, "1"+domain, teamIDConfig()),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll("the git_branch specified is the production branch. If you want to use this domain as a production domain, please omit the git_branch field.", " ", `\s*`),
				),
			},
			// Create and Read testing
			{
				Config: testAccProjectDomainConfig(projectSuffix, "2"+domain, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDomainExists("vercel_project.test", testTeam(), "2"+domain),
					testTeamID,
					resource.TestCheckResourceAttr("vercel_project_domain.test", "domain", "2"+domain),
				),
			},
			// Update testing
			{
				Config: testAccProjectDomainConfigUpdated(projectSuffix, "2"+domain, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_domain.test", "redirect"),
				),
			},
			// Redirect Update testing
			{
				Config: testAccProjectDomainConfigUpdated2(projectSuffix, "2"+domain, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_domain.test", "redirect"),
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

func TestAcc_ProjectDomainCustomEnvironment(t *testing.T) {
	randomSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Ensure we can't have both git_branch and custom_environment_id
			{
				Config: testAccProjectDomainConfigWithCustomEnvironmentAndGitBranch(randomSuffix),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll("Attribute \"git_branch\" cannot be specified when \"custom_environment_id\" is specified", " ", `\s*`),
				),
			},
			{
				Config: testAccProjectDomainConfigWithCustomEnvironment(randomSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_domain.test", "custom_environment_id"),
				),
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

  redirect = vercel_project_domain.redirect_target.domain
}

resource "vercel_project_domain" "redirect_target" {
    domain = "redirect-target-1-%[3]s"
    project_id = vercel_project.test.id
    %[2]s
}
`, projectSuffix, extra, domain, extra)
}

func testAccProjectDomainConfigUpdated2(projectSuffix, domain, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
  %[2]s
}

resource "vercel_project_domain" "redirect_target" {
    domain = "redirect-target-1-%[3]s"
    project_id = vercel_project.test.id
    %[2]s
}

resource "vercel_project_domain" "redirect_target_2" {
    domain = "redirect-target-2-%[3]s"
    project_id = vercel_project.test.id
    %[2]s
}

resource "vercel_project_domain" "test" {
  domain = "%[3]s"
  project_id = vercel_project.test.id
  %[2]s

  redirect = vercel_project_domain.redirect_target_2.domain
  redirect_status_code = 307
}
`, projectSuffix, extra, domain)
}

func testAccProjectDomainConfigDeleted(projectSuffix, extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
  %s
}
`, projectSuffix, extra)
}

func testAccProjectDomainConfigWithCustomEnvironment(randomSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
  %[2]s
}

resource "vercel_custom_environment" "test" {
    name = "test-acc-custom-environment"
    project_id = vercel_project.test.id
    %[2]s
}

resource "vercel_project_domain" "test" {
    domain = "test-acc-domain-%[1]s-foobar.vercel.app"
    project_id = vercel_project.test.id
    custom_environment_id = vercel_custom_environment.test.id
    %[2]s
}
`, randomSuffix, teamIDConfig())
}

func testAccProjectDomainConfigWithCustomEnvironmentAndGitBranch(randomSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
  %[2]s
  git_repository = {
    type = "github"
    repo = "%[3]s"
  }
}

resource "vercel_custom_environment" "test" {
    name = "test-acc-custom-environment"
    project_id = vercel_project.test.id
    %[2]s
}

resource "vercel_project_domain" "test" {
    domain = "test-acc-domain-%[1]s.vercel.app"
    project_id = vercel_project.test.id
    custom_environment_id = vercel_custom_environment.test.id
    git_branch = "staging"
    %[2]s
}
`, randomSuffix, teamIDConfig(), testGithubRepo())
}
