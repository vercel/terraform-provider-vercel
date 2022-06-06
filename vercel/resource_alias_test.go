package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func testCheckAliasExists(teamID, alias string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		_, err := testClient().GetAlias(context.TODO(), alias, teamID)
		return err
	}
}

func testCheckAliasDestroyed(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no alias is set")
		}

		_, err := testClient().GetAlias(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted project: %s", err)
		}

		return nil
	}
}

func TestAcc_AliasResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAliasDestroyed("vercel_alias.test", ""),
		Steps: []resource.TestStep{
			{
				Config: testAccAliasResourceConfig(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAliasExists("", fmt.Sprintf("test-acc-%s.vercel.app", name)),
					resource.TestCheckResourceAttr("vercel_alias.test", "alias", fmt.Sprintf("test-acc-%s.vercel.app", name)),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "deployment_id"),
				),
			},
		},
	})
}

func TestAcc_AliasResourceTeam(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAliasDestroyed("vercel_alias.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccAliasResourceConfig(name, fmt.Sprintf("team_id = \"%s\"", testTeam())),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAliasExists(testTeam(), fmt.Sprintf("test-acc-%s.vercel.app", name)),
					resource.TestCheckResourceAttr("vercel_alias.test", "alias", fmt.Sprintf("test-acc-%s.vercel.app", name)),
					resource.TestCheckResourceAttr("vercel_alias.test", "team_id", testTeam()),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_alias.test", "deployment_id"),
				),
			},
		},
	})
}

func testAccAliasResourceConfig(name, team string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
    %[2]s
    git_repository = {
        type = "github"
        repo = "%[3]s"
    }
}

resource "vercel_deployment" "test" {
    project_id = vercel_project.test.id
    ref        = "main"
    %[2]s
}

resource "vercel_alias" "test" {
    alias         = "test-acc-%[1]s.vercel.app"
    deployment_id = vercel_deployment.test.id
    %[2]s
}
`, name, team, testGithubRepo())
}
