package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_SharedEnvironmentVariableDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSharedEnvironmentVariableDataSource(name, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.test", "key", "test_acc_"+name),
					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.test", "value", "foobar"),
					resource.TestCheckTypeSetElemAttr("data.vercel_shared_environment_variable.test", "target.*", "production"),
					resource.TestCheckTypeSetElemAttr("data.vercel_shared_environment_variable.test", "target.*", "preview"),
					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.test", "sensitive", "false"),

					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.by_key_and_target", "key", "test_acc_"+name),
					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.by_key_and_target", "value", "foobar"),
					resource.TestCheckTypeSetElemAttr("data.vercel_shared_environment_variable.by_key_and_target", "target.*", "production"),
					resource.TestCheckTypeSetElemAttr("data.vercel_shared_environment_variable.by_key_and_target", "target.*", "preview"),
					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.by_key_and_target", "sensitive", "false"),

					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.sensitive", "key", "test_acc_"+name+"_sensitive"),
					resource.TestCheckNoResourceAttr("data.vercel_shared_environment_variable.sensitive", "value"),
					resource.TestCheckTypeSetElemAttr("data.vercel_shared_environment_variable.sensitive", "target.*", "production"),
					resource.TestCheckResourceAttr("data.vercel_shared_environment_variable.sensitive", "sensitive", "true"),
				),
			},
		},
	})
}

func testAccSharedEnvironmentVariableDataSource(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_shared_environment_variable" "test" {
  key = "test_acc_%[1]s"
  value = "foobar"
  target = [ "production", "preview" ]
  project_ids = [ vercel_project.test.id ]
  %[2]s
}

data "vercel_shared_environment_variable" "test" {
    id = vercel_shared_environment_variable.test.id
    %[2]s
}

data "vercel_shared_environment_variable" "by_key_and_target" {
    key = vercel_shared_environment_variable.test.key
    target = vercel_shared_environment_variable.test.target
    %[2]s
}

resource "vercel_shared_environment_variable" "sensitive" {
    key = "test_acc_%[1]s_sensitive"
    %[2]s
    value = "foobar"
    target = [ "production" ]
    project_ids = [ vercel_project.test.id ]
    sensitive = true
}

data "vercel_shared_environment_variable" "sensitive" {
    id = vercel_shared_environment_variable.sensitive.id
    %[2]s
}
`, name, teamID)
}
