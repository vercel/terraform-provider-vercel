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
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func TestAcc_ProjectDomain(t *testing.T) {
	testTeamID := resource.TestCheckNoResourceAttr("vercel_project.test", "team_id")
	if testTeam(t) != "" {
		testTeamID = resource.TestCheckResourceAttr("vercel_project.test", "team_id", testTeam(t))
	}

	projectSuffix := acctest.RandString(16)
	domain := acctest.RandString(30) + ".vercel.app"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Check error adding production domain
			{
				Config: cfg(testAccProjectDomainWithProductionGitBranch(projectSuffix, "1"+domain, testGithubRepo(t))),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll("the git_branch specified is the production branch. If you want to use this domain as a production domain, please omit the git_branch field.", " ", `\s*`),
				),
			},
			// Create and Read testing
			{
				Config: cfg(testAccProjectDomainConfig(projectSuffix, "2"+domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDomainExists(testClient(t), "vercel_project.test", testTeam(t), "2"+domain),
					testTeamID,
					resource.TestCheckResourceAttr("vercel_project_domain.test", "domain", "2"+domain),
				),
			},
			// Update testing
			{
				Config: cfg(testAccProjectDomainConfigUpdated(projectSuffix, "2"+domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_domain.test", "redirect"),
				),
			},
			// Redirect Update testing
			{
				Config: cfg(testAccProjectDomainConfigUpdated2(projectSuffix, "2"+domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_domain.test", "redirect"),
					resource.TestCheckResourceAttr("vercel_project_domain.test", "redirect_status_code", "307"),
				),
			},
			// Delete testing
			{
				Config: cfg(testAccProjectDomainConfigDeleted(projectSuffix)),
				Check:  testAccProjectDomainDestroy(testClient(t), "vercel_project.test", testTeam(t), domain),
			},
		},
	})
}

func TestAcc_ProjectDomainCustomEnvironment(t *testing.T) {
	randomSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Ensure we can't have both git_branch and custom_environment_id
			{
				Config: cfg(testAccProjectDomainConfigWithCustomEnvironmentAndGitBranch(randomSuffix, testGithubRepo(t))),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll("Attribute \"git_branch\" cannot be specified when \"custom_environment_id\" is specified", " ", `\s*`),
				),
			},
			{
				Config: cfg(testAccProjectDomainConfigWithCustomEnvironment(randomSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_domain.test", "custom_environment_id"),
				),
			},
		},
	})
}

func testAccProjectDomainExists(testClient *client.Client, n, teamID, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient.GetProjectDomain(context.TODO(), rs.Primary.ID, domain, teamID)
		return err
	}
}

func testAccProjectDomainDestroy(testClient *client.Client, n, teamID, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient.GetProjectDomain(context.TODO(), rs.Primary.ID, domain, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted deployment: %s", err)
		}
		return nil
	}
}

func testAccProjectDomainWithProductionGitBranch(projectSuffix, domain, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
  git_repository = {
    type = "github"
    repo = "%[3]s"
  }
}

resource "vercel_project_domain" "test" {
  domain = "%[2]s"
  git_branch = "main"
  project_id = vercel_project.test.id
}
`, projectSuffix, domain, githubRepo)
}

func testAccProjectDomainConfig(projectSuffix, domain string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
}

resource "vercel_project_domain" "test" {
  domain = "%s"
  project_id = vercel_project.test.id
}
`, projectSuffix, domain)
}

func testAccProjectDomainConfigUpdated(projectSuffix, domain string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
}

resource "vercel_project_domain" "test" {
  domain = "%s"
  project_id = vercel_project.test.id

  redirect = vercel_project_domain.redirect_target.domain
}

resource "vercel_project_domain" "redirect_target" {
    domain = "redirect-target-1-%[2]s"
    project_id = vercel_project.test.id
}
`, projectSuffix, domain)
}

func testAccProjectDomainConfigUpdated2(projectSuffix, domain string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
}

resource "vercel_project_domain" "redirect_target" {
    domain = "redirect-target-1-%[2]s"
    project_id = vercel_project.test.id
}

resource "vercel_project_domain" "redirect_target_2" {
    domain = "redirect-target-2-%[2]s"
    project_id = vercel_project.test.id
}

resource "vercel_project_domain" "test" {
  domain = "%[2]s"
  project_id = vercel_project.test.id

  redirect = vercel_project_domain.redirect_target_2.domain
  redirect_status_code = 307
}
`, projectSuffix, domain)
}

func testAccProjectDomainConfigDeleted(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%s"
}
`, projectSuffix)
}

func testAccProjectDomainConfigWithCustomEnvironment(randomSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
}

resource "vercel_custom_environment" "test" {
    name = "test-acc-custom-environment"
    project_id = vercel_project.test.id
}

resource "vercel_project_domain" "test" {
    domain = "test-acc-domain-%[1]s-foobar.vercel.app"
    project_id = vercel_project.test.id
    custom_environment_id = vercel_custom_environment.test.id
}
`, randomSuffix)
}

func testAccProjectDomainConfigWithCustomEnvironmentAndGitBranch(randomSuffix, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-%[1]s"
  git_repository = {
    type = "github"
    repo = "%[2]s"
  }
}

resource "vercel_custom_environment" "test" {
    name = "test-acc-custom-environment"
    project_id = vercel_project.test.id
}

resource "vercel_project_domain" "test" {
    domain = "test-acc-domain-%[1]s.vercel.app"
    project_id = vercel_project.test.id
    custom_environment_id = vercel_custom_environment.test.id
    git_branch = "staging"
}
`, randomSuffix, githubRepo)
}
