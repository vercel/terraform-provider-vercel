package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func testCheckAliasExists(testClient *client.Client, teamID, alias string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		_, err := testClient.GetAlias(context.TODO(), alias, teamID)
		return err
	}
}

func testCheckAliasDestroyed(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no alias is set")
		}

		_, err := testClient.GetAlias(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted alias: %s", err)
		}

		return nil
	}
}

func TestAcc_AliasResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAliasDestroyed(testClient(t), "vercel_alias.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAliasResourceConfig(name, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAliasExists(testClient(t), testTeam(t), fmt.Sprintf("test-acc-%s-alias.vercel.app", name)),
					resource.TestCheckResourceAttr("vercel_alias.test", "alias", fmt.Sprintf("test-acc-%s-alias.vercel.app", name)),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "deployment_id"),
				),
			},
			{
				Config: cfg(testAccAliasResourceConfigUpdated(name, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAliasExists(testClient(t), testTeam(t), fmt.Sprintf("test-acc-%s-alias.vercel.app", name)),
					resource.TestCheckResourceAttr("vercel_alias.test", "alias", fmt.Sprintf("test-acc-%s-alias.vercel.app", name)),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "deployment_id"),
				),
			},
		},
	})
}

func testAccAliasResourceConfig(name, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
    git_repository = {
        type = "github"
        repo = "%[2]s"
    }
}

resource "vercel_deployment" "test" {
    project_id = vercel_project.test.id
    ref        = "main"
}

resource "vercel_alias" "test" {
    alias         = "test-acc-%[1]s-alias.vercel.app"
    deployment_id = vercel_deployment.test.id
}
`, name, githubRepo)
}

func testAccAliasResourceConfigUpdated(name, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
    git_repository = {
        type = "github"
        repo = "%[2]s"
    }
}

resource "vercel_deployment" "test_two" {
    project_id = vercel_project.test.id
    ref        = "main"
}

resource "vercel_alias" "test" {
    alias         = "test-acc-%[1]s-alias.vercel.app"
    deployment_id = vercel_deployment.test_two.id
}
`, name, githubRepo)
}
