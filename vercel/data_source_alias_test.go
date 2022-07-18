package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_AliasDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAliasDataSourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_alias.test", "alias", fmt.Sprintf("test-acc-%s.vercel.app", name)),
					resource.TestCheckResourceAttr("data.vercel_alias.test", "team_id", testTeam()),
					resource.TestCheckResourceAttrSet("data.vercel_alias.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_alias.test", "deployment_id"),
				),
			},
		},
	})
}

func testAccAliasDataSourceConfig(name, team string) string {
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
    %[2]s
    ref        = "main"
}

resource "vercel_alias" "test" {
    alias         = "test-acc-%[1]s.vercel.app"
    %[2]s
    deployment_id = vercel_deployment.test.id
}

data "vercel_alias" "test" {
    alias = vercel_alias.test.alias
    %[2]s
}
`, name, team, testGithubRepo())
}
