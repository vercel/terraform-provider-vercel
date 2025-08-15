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

func testAccSharedEnvironmentVariableExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetSharedEnvironmentVariable(context.TODO(), teamID, rs.Primary.ID)
		return err
	}
}

func TestAcc_SharedEnvironmentVariables(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccSharedEnvironmentVariablesConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "key", fmt.Sprintf("test_acc_foo_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "value", "bar"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "comment", "Test comment for example"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "apply_to_all_custom_environments", "true"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "production"),

					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.sensitive_example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "key", fmt.Sprintf("test_acc_foo_sensitive_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "value", "bar"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "comment", "Test comment for sensitive example"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.sensitive_example", "target.*", "production"),

					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.no_comment_example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.no_comment_example", "key", fmt.Sprintf("test_acc_foo_no_comment_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.no_comment_example", "value", "baz"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.no_comment_example", "comment", ""),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.no_comment_example", "target.*", "production"),
				),
			},
			{
				Config: cfg(testAccSharedEnvironmentVariablesConfigUpdated(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "key", fmt.Sprintf("test_acc_foo_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "value", "updated-bar"),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.example", "apply_to_all_custom_environments", "false"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "development"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.example", "target.*", "preview"),

					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "key", fmt.Sprintf("test_acc_foo_sensitive_%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.sensitive_example", "value", "bar-updated"),
					resource.TestCheckTypeSetElemAttr("vercel_shared_environment_variable.sensitive_example", "target.*", "production"),
				),
			},
			{
				ResourceName:      "vercel_shared_environment_variable.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getSharedEnvironmentVariableImportID("vercel_shared_environment_variable.example"),
			},
			{
				Config: cfg(testAccSharedEnvironmentVariablesConfigDeleted(nameSuffix)),
			},
		},
	})
}

func getSharedEnvironmentVariableImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func testAccSharedEnvironmentVariablesConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
}

resource "vercel_shared_environment_variable" "example" {
	key         = "test_acc_foo_%[1]s"
	value       = "bar"
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
    comment     = "Test comment for example"
	apply_to_all_custom_environments = true
}

resource "vercel_shared_environment_variable" "sensitive_example" {
	key         = "test_acc_foo_sensitive_%[1]s"
	value       = "bar"
    sensitive   = true
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
    comment     = "Test comment for sensitive example"
}

resource "vercel_shared_environment_variable" "no_comment_example" {
	key         = "test_acc_foo_no_comment_%[1]s"
	value       = "baz"
	target      = ["production"]
    project_ids = [
        vercel_project.example.id
    ]
}
`, projectName)
}

func testAccSharedEnvironmentVariablesConfigUpdated(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
}

resource "vercel_project" "example2" {
	name = "test-acc-example-project-2-%[1]s"
}

resource "vercel_shared_environment_variable" "example" {
	key         = "test_acc_foo_%[1]s"
	value       = "updated-bar"
	target      = ["preview", "development"]
    project_ids = [
        vercel_project.example.id,
        vercel_project.example2.id
    ]
	apply_to_all_custom_environments = false
}

resource "vercel_shared_environment_variable" "sensitive_example" {
	key         = "test_acc_foo_sensitive_%[1]s"
	value       = "bar-updated"
    sensitive   = true
	target      = ["production"]
    project_ids = [
        vercel_project.example.id,
        vercel_project.example2.id
    ]
}
`, projectName)
}

func testAccSharedEnvironmentVariablesConfigDeleted(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
}

resource "vercel_project" "example2" {
	name = "test-acc-example-project-2-%[1]s"
}
    `, projectName)
}

func TestAcc_SharedEnvironmentVariables_CustomOnly_OmitTarget(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccSharedEnvironmentVariablesCustomOnlyOmitTargetConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.custom_only_omit", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.custom_only_omit", "apply_to_all_custom_environments", "true"),
					testAccCheckSetEmpty("vercel_shared_environment_variable.custom_only_omit", "target.#"),
				),
			},
		},
	})
}

func TestAcc_SharedEnvironmentVariables_CustomOnly_EmptyTarget(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccSharedEnvironmentVariablesCustomOnlyEmptyTargetConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccSharedEnvironmentVariableExists(testClient(t), "vercel_shared_environment_variable.custom_only_empty", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_shared_environment_variable.custom_only_empty", "apply_to_all_custom_environments", "true"),
					testAccCheckSetEmpty("vercel_shared_environment_variable.custom_only_empty", "target.#"),
				),
			},
		},
	})
}

func testAccSharedEnvironmentVariablesCustomOnlyOmitTargetConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
}

resource "vercel_shared_environment_variable" "custom_only_omit" {
	key         = "test_acc_custom_only_omit_%[1]s"
	value       = "bar"
	project_ids = [
		vercel_project.example.id
	]
	apply_to_all_custom_environments = true
	# target intentionally omitted
}
`, projectName)
}

func testAccSharedEnvironmentVariablesCustomOnlyEmptyTargetConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%[1]s"
}

resource "vercel_shared_environment_variable" "custom_only_empty" {
	key         = "test_acc_custom_only_empty_%[1]s"
	value       = "bar"
	# explicitly pass an empty set
	target      = []
	project_ids = [
		vercel_project.example.id
	]
	apply_to_all_custom_environments = true
}
`, projectName)
}

// testAccCheckSetEmpty asserts that either the attribute is absent or equals "0".
func testAccCheckSetEmpty(n, attr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if v, ok := rs.Primary.Attributes[attr]; ok {
			if v == "0" {
				return nil
			}
			return fmt.Errorf("expected %s to be empty, got %s", attr, v)
		}
		return nil
	}
}
