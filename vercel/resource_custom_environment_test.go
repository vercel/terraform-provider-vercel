package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

func testCheckCustomEnvironmentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		projectID := rs.Primary.Attributes["project_id"]
		name := rs.Primary.Attributes["name"]

		_, err := testClient().GetCustomEnvironment(context.TODO(), client.GetCustomEnvironmentRequest{
			TeamID:    testTeam(),
			ProjectID: projectID,
			Slug:      name,
		})
		if client.NotFound(err) {
			return fmt.Errorf("test failed because the custom environment %s %s %s - %s could not be found", testTeam(), projectID, name, rs.Primary.ID)
		}
		return err
	}
}

func TestAcc_CustomEnvironmentResource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccCustomEnvironment(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckCustomEnvironmentExists("vercel_custom_environment.test"),
					resource.TestCheckResourceAttrSet("vercel_custom_environment.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_custom_environment.test", "project_id"),
					resource.TestCheckResourceAttrSet("vercel_custom_environment.test", "name"),
					resource.TestCheckNoResourceAttr("vercel_custom_environment.test", "branch_tracking"),
					resource.TestCheckResourceAttr("vercel_custom_environment.test", "description", "without branch tracking"),

					/*
						testCheckCustomEnvironmentExists("vercel_custom_environment.test_bt"),
						resource.TestCheckResourceAttrSet("vercel_custom_environment.test_bt", "id"),
						resource.TestCheckResourceAttrSet("vercel_custom_environment.test_bt", "project_id"),
						resource.TestCheckResourceAttr("vercel_custom_environment.test_bt", "name", "test-acc-bt"),
						resource.TestCheckResourceAttr("vercel_custom_environment.test_bt", "branch_tracking.type", "startsWith"),
						resource.TestCheckResourceAttr("vercel_custom_environment.test_bt", "branch_tracking.pattern", "staging-"),
						resource.TestCheckResourceAttr("vercel_custom_environment.test_bt", "description", "with branch tracking"),
					*/

					// check project env var
					resource.TestCheckTypeSetElemAttr("vercel_project_environment_variable.test", "target.*", "test-acc"),

					// check shared env var
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.test", "target.*", "test-acc"),
				),
			},
			{
				Config: testAccCustomEnvironmentUpdated(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckCustomEnvironmentExists("vercel_custom_environment.test"),
					resource.TestCheckResourceAttrSet("vercel_custom_environment.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_custom_environment.test", "project_id"),
					resource.TestCheckResourceAttrSet("vercel_custom_environment.test", "name"),
					resource.TestCheckResourceAttr("vercel_custom_environment.test", "branch_tracking.type", "endsWith"),
					resource.TestCheckResourceAttr("vercel_custom_environment.test", "branch_tracking.pattern", "staging-"),
					resource.TestCheckResourceAttr("vercel_custom_environment.test", "description", "without branch tracking updated"),
				),
			},
			{
				ResourceName:      "vercel_custom_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getCustomEnvImportID("vercel_shared_environment_variable.example"),
			},
		},
	})
}

func getCustomEnvImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"], rs.Primary.Attributes["name"]), nil
	}
}

func testAccCustomEnvironment(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-custom-env-%[1]s"
  %[2]s
}

resource "vercel_custom_environment" "test" {
  project_id = vercel_project.test.id
  %[2]s
  name = "test-acc-%[1]s"
  description = "without branch tracking"
}

// Ensure project_environment_variable works
resource "vercel_project_environment_variable" "test" {
    project_id = vercel_project.test.id
    %[2]s
    key = "foo"
    value = "test-acc-env-var"
    target = [vercel_custom_environment.test.name]
}

// Ensurer shared_environment_variable works
resource "vercel_shared_environment_variable" "test" {
    project_ids = [vercel_project.test.id]
    %[2]s
    key = "bar"
    value = "test-acc-shared-env-var"
    target = [vercel_custom_environment.test.name]
}

/*
resource "vercel_custom_environment" "test_bt" {
  project_id = vercel_project.test.id
  %[2]s
  name = "test-acc-bt-%[1]s"
  description = "with branch tracking"
  branch_tracking = {
    pattern = "staging-"
    type = "startsWith"
  }
}
*/
`, projectSuffix, teamIDConfig())
}

func testAccCustomEnvironmentUpdated(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-custom-env-%[1]s"
  %[2]s
}

resource "vercel_custom_environment" "test" {
  project_id = vercel_project.test.id
  %[2]s
  name = "test-acc-%[1]s-updated"
  description = "without branch tracking updated"
  branch_tracking = {
      pattern = "staging-"
      type = "endsWith"
  }
}
`, projectSuffix, teamIDConfig())
}
