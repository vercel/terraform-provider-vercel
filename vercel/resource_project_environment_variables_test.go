package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectEnvironmentVariables(t *testing.T) {
	projectName := "test-acc-example-env-vars-" + acctest.RandString(16)
	resourceName := "vercel_project_environment_variables.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectEnvironmentVariablesConfig(projectName, testGithubRepo(t))),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"key":   "TEST_VAR_1",
						"value": "test_value_1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"key":        "TEST_VAR_2",
						"value":      "test_value_2",
						"git_branch": "staging",
					}),
					resource.TestCheckResourceAttrSet(resourceName, "variables.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "variables.1.id"),
				),
			},
			{
				Config: cfg(testAccProjectEnvironmentVariablesConfigUpdated(projectName, testGithubRepo(t))),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"key":   "TEST_VAR_1",
						"value": "test_value_1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"key":   "TEST_VAR_2",
						"value": "test_value_2_updated",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"key":   "TEST_VAR_3",
						"value": "test_value_3",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"key":       "TEST_VAR_4",
						"value":     "sensitive_value",
						"sensitive": "true",
					}),
					resource.TestCheckResourceAttrSet(resourceName, "variables.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "variables.1.id"),
					resource.TestCheckResourceAttrSet(resourceName, "variables.2.id"),
				),
			},
		},
	})
}

func testAccProjectEnvironmentVariablesConfig(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "%s"

  git_repository = {
    type = "github"
    repo = "%[2]s"
  }
}

resource "vercel_project_environment_variables" "test" {
  project_id = vercel_project.test.id
  variables = [
    {
      key   = "TEST_VAR_1"
      value = "test_value_1"
      target = ["production", "preview"]
    },
    {
      key   = "TEST_VAR_2"
      value = "test_value_2"
      git_branch = "staging"
      target = ["preview"]
    }
  ]
}
`, projectName, githubRepo)
}

func testAccProjectEnvironmentVariablesConfigUpdated(projectName string, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "%s"

  git_repository = {
    type = "github"
    repo = "%[2]s"
  }
}

resource "vercel_project_environment_variables" "test" {
  project_id = vercel_project.test.id
  variables = [
    {
      key   = "TEST_VAR_1" // unchanged
      value = "test_value_1"
      target = ["production", "preview"]
    },
    {
      key    = "TEST_VAR_2"
      value  = "test_value_2_updated"
      target = ["preview", "development"]
    },
    {
      key = "TEST_VAR_3"
      value = "test_value_3"
      target = ["production"]
    },
    {
      key = "TEST_VAR_4"
      value = "sensitive_value"
      target = ["production"]
      sensitive = true
    }
  ]
}
`, projectName, githubRepo)
}
