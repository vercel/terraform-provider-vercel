package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func TestAcc_Project(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			// Ensure we get nice framework / serverless_function_region errors
			{
				Config: `
				                    resource "vercel_project" "test" {
				                        name = "foo"
				                        serverless_function_region = "notexist"
				                    }
				                `,
				ExpectError: regexp.MustCompile("Invalid Serverless Function Region"),
			},
			{
				Config: `
				                    resource "vercel_project" "test" {
				                        name = "foo"
				                        framework = "notexist"
				                    }
				                `,
				ExpectError: regexp.MustCompile("Invalid Framework"),
			},
			// Create and Read testing
			{
				Config: cfg(testAccProjectConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-project-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("vercel_project.test", "framework", "nextjs"),
					resource.TestCheckResourceAttr("vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "root_directory", "ui/src"),
					resource.TestCheckResourceAttr("vercel_project.test", "ignore_command", "echo 'wat'"),
					resource.TestCheckResourceAttr("vercel_project.test", "serverless_function_region", "syd1"),
					resource.TestCheckResourceAttr("vercel_project.test", "automatically_expose_system_environment_variables", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "foo",
						"value": "bar",
					}),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "environment.0.target.*", "production"),
					resource.TestCheckResourceAttr("vercel_project.test", "git_comments.on_pull_request", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "git_comments.on_commit", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "preview_comments", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "enable_preview_feedback", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "enable_production_feedback", "false"),
					resource.TestCheckResourceAttr("vercel_project.test", "preview_deployments_disabled", "false"),
					resource.TestCheckResourceAttr("vercel_project.test", "auto_assign_custom_domains", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "git_lfs", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "function_failover", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "customer_success_code_visibility", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "git_fork_protection", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "prioritise_production_builds", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "directory_listing", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "skew_protection", "7 days"),
					resource.TestCheckResourceAttr("vercel_project.test", "oidc_token_config.enabled", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "oidc_token_config.issuer_mode", "team"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_cpu_type", "standard"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_timeout", "60"),
					resource.TestCheckResourceAttr("vercel_project.test", "on_demand_concurrent_builds", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "build_machine_type", "enhanced"),
				),
			},
			// Update testing
			{
				Config: cfg(testAccProjectConfigUpdated(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-two-%s", projectSuffix)),
					resource.TestCheckNoResourceAttr("vercel_project.test", "build_command"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "bar",
						"value": "baz",
					}),
					resource.TestCheckResourceAttr("vercel_project.test", "oidc_token_config.enabled", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "preview_comments", "false"),
					resource.TestCheckResourceAttr("vercel_project.test", "enable_preview_feedback", "false"),
					resource.TestCheckResourceAttr("vercel_project.test", "enable_production_feedback", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "preview_deployments_disabled", "false"),
					resource.TestCheckResourceAttr("vercel_project.test", "on_demand_concurrent_builds", "false"),
					resource.TestCheckResourceAttr("vercel_project.test", "build_machine_type", ""),
				),
			},
			// Test mutual exclusivity validation
			{
				Config: cfg(testAccProjectConfigPreviewFeedbackConflict(projectSuffix)),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll("Attribute \"preview_comments\" cannot be specified when \"enable_preview_feedback\" is specified", " ", `\s*`),
				),
			},
			// Test using only the deprecated field
			{
				Config: cfg(testAccProjectConfigPreviewCommentsOnly(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "preview_comments", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "enable_preview_feedback", "true"),
				),
			},
			// Test updating from deprecated field to new field
			{
				Config: cfg(testAccProjectConfigPreviewFeedbackOnly(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "preview_comments", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "enable_preview_feedback", "true"),
				),
			},
		},
	})
}

func TestAcc_ProjectFluidCompute(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				// check we get a sensible error if fluid + invalid CPU combination.
				Config: cfg(fmt.Sprintf(`
                resource "vercel_project" "test" {
                    name = "test-acc-fluid-%[1]s"
                    resource_config = {
                        fluid = true
                        function_default_cpu_type = "standard_legacy"
                    }
                }
                `, projectSuffix)),
				ExpectError: regexp.MustCompile(strings.ReplaceAll("\"standard_legacy\" is not a valid memory type for Fluid compute", " ", `\s*`)),
			},
			{
				// check creating a project with Fluid
				Config: cfg(fmt.Sprintf(`
                    resource "vercel_project" "test" {
                        name = "test-acc-fluid-%[1]s"

                        resource_config = {
                            fluid = true
                        }
                    }
                `, projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-fluid-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.fluid", "true"),
				),
			},
			{
				// check updating Fluid on a project
				Config: cfg(fmt.Sprintf(`
                    resource "vercel_project" "test" {
                        name = "test-acc-fluid-%[1]s"

                        resource_config = {
                            fluid = false
                        }
                    }
                `, projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-fluid-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.fluid", "false"),
				),
			},
			{
				// check new projects without fluid specified shows fluid as false
				Config: cfg(fmt.Sprintf(`
                    resource "vercel_project" "test" {
                        name = "test-acc-fluid-disabled-%[1]s"
                    }
                `, projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-fluid-disabled-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.fluid", "false"),
				),
			},
		},
	})
}

func TestAcc_ProjectFunctionDefaultRegions(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				// check if legacy setting serverless_function_region conflicts with resource_config.function_default_regions
				Config: cfg(fmt.Sprintf(`
                resource "vercel_project" "test" {
                    name = "test-acc-regions-conflict-%[1]s"
                    serverless_function_region = "sfo1"
                    resource_config = {
                        function_default_regions = ["iad1", "fra1"]
                    }
                }
                `, projectSuffix)),
				ExpectError: regexp.MustCompile(strings.ReplaceAll("Attribute \"serverless_function_region\" cannot be specified when \"resource_config.function_default_regions\" is specified", " ", `\s*`)),
			},
			{
				// check invalid region value
				Config: cfg(fmt.Sprintf(`
                resource "vercel_project" "test" {
                    name = "test-acc-regions-invalid-%[1]s"
                    resource_config = {
                        function_default_regions = ["invalid-region"]
                    }
                }
                `, projectSuffix)),
				ExpectError: regexp.MustCompile(strings.ReplaceAll("Invalid Serverless Function Region", " ", `\s*`)),
			},
			{
				// check creating a project with function_default_regions
				Config: cfg(fmt.Sprintf(`
                resource "vercel_project" "test" {
                    name = "test-acc-regions-%[1]s"
                    resource_config = {
                        function_default_regions = ["sfo1", "iad1", "fra1"]
                        function_default_cpu_type = "standard"
                        function_default_timeout = 30
                    }
                }
                `, projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-regions-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_regions.#", "3"),
					// resource.TestCheckResourceAttr("vercel_project.test", "serverless_function_region", "sfo1"),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "resource_config.function_default_regions.*", "sfo1"),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "resource_config.function_default_regions.*", "iad1"),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "resource_config.function_default_regions.*", "fra1"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_cpu_type", "standard"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_timeout", "30"),
				),
			},
			{
				// check updating a projects function_default_regions
				Config: cfg(fmt.Sprintf(`
                resource "vercel_project" "test" {
                    name = "test-acc-regions-%[1]s"
                    resource_config = {
                        function_default_regions = ["hkg1", "sin1"]
                        function_default_cpu_type = "performance"
                        function_default_timeout = 60
                    }
                }
                `, projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-regions-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_regions.#", "2"),
					resource.TestCheckResourceAttr("vercel_project.test", "serverless_function_region", "hkg1"),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "resource_config.function_default_regions.*", "hkg1"),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "resource_config.function_default_regions.*", "sin1"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_cpu_type", "performance"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_timeout", "60"),
				),
			},
			{
				// check switching project from function_default_regions to serverless_function_region
				Config: cfg(fmt.Sprintf(`
                resource "vercel_project" "test" {
                    name = "test-acc-regions-%[1]s"
                    serverless_function_region = "syd1"
                }
                `, projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", fmt.Sprintf("test-acc-regions-%s", projectSuffix)),
					resource.TestCheckResourceAttr("vercel_project.test", "serverless_function_region", "syd1"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_regions.#", "1"),
					resource.TestCheckResourceAttr("vercel_project.test", "resource_config.function_default_regions.0", "syd1"),
				),
			},
		},
	})
}

func TestAcc_ProjectAddingEnvAfterInitialCreation(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigWithoutEnv(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
			},
			{
				Config: cfg(testAccProjectConfigUpdated(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
			},
		},
	})
}

func TestAcc_ProjectUpdateResourceConfig(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigBase(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"vercel_project.test",
						tfjsonpath.New("resource_config").AtMapKey("function_default_cpu_type"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"vercel_project.test",
						tfjsonpath.New("resource_config").AtMapKey("function_default_timeout"),
						knownvalue.Null(),
					),
				},
			},
			{
				Config: cfg(testAccProjectConfigBase(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: cfg(testAccProjectConfigWithResourceConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"vercel_project.test",
						tfjsonpath.New("resource_config").AtMapKey("function_default_cpu_type"),
						knownvalue.StringExact("standard_legacy"),
					),
					statecheck.ExpectKnownValue(
						"vercel_project.test",
						tfjsonpath.New("resource_config").AtMapKey("function_default_timeout"),
						knownvalue.Int64Exact(30),
					),
				},
			},
			{
				Config: cfg(testAccProjectConfigWithResourceConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAcc_ProjectWithGitRepository(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test_git", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigWithGitRepo(projectSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test_git", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.test_git", "git_repository.type", "github"),
					resource.TestCheckResourceAttr("vercel_project.test_git", "git_repository.repo", testGithubRepo(t)),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test_git", "environment.*", map[string]string{
						"key":        "foo",
						"value":      "bar",
						"git_branch": "staging",
						"comment":    "some comment",
					}),
				),
			},
			{
				Config: cfg(testAccProjectConfigWithGitRepoUpdated(projectSuffix, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test_git", testTeam(t)),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test_git", "environment.*", map[string]string{
						"key":     "foo",
						"value":   "bar2",
						"comment": "some updated comment",
					}),
				),
			},
			{
				Config: cfg(testAccProjectConfigWithGitRepoRemoved(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test_git", testTeam(t)),
					resource.TestCheckNoResourceAttr("vercel_project.test_git", "git_repository"),
				),
			},
		},
	})
}

func TestAcc_ProjectWithVercelAuthAndPasswordProtectionAndTrustedIps(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.enabled_to_start", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigWithVercelAuthAndPasswordAndTrustedIpsAndOptionsAllowlist(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.enabled_to_start", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "vercel_authentication.deployment_type", "all_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "password_protection.deployment_type", "all_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "password_protection.password", "password"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "trusted_ips.addresses.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.enabled_to_start", "trusted_ips.addresses.*", map[string]string{
						"value": "1.1.1.1",
						"note":  "notey note",
					}),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "trusted_ips.deployment_type", "all_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "trusted_ips.protection_mode", "trusted_ip_required"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "options_allowlist.paths.#", "1"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "options_allowlist.paths.0.value", "/foo"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.enabled_to_start", "protection_bypass_for_automation_secret"),
					testAccProjectExists(testClient(t), "vercel_project.disabled_to_start", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "vercel_authentication.deployment_type", "standard_protection_new"),
					resource.TestCheckNoResourceAttr("vercel_project.disabled_to_start", "password_protection"),
					resource.TestCheckNoResourceAttr("vercel_project.disabled_to_start", "trusted_ips"),
					resource.TestCheckNoResourceAttr("vercel_project.disabled_to_start", "protection_bypass_for_automation"),
					resource.TestCheckNoResourceAttr("vercel_project.disabled_to_start", "protection_bypass_for_automation_secret"),
					testAccProjectExists(testClient(t), "vercel_project.enabled_to_update", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "vercel_authentication.deployment_type", "only_preview_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "password_protection.deployment_type", "only_preview_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "password_protection.password", "password"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "trusted_ips.addresses.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.enabled_to_update", "trusted_ips.addresses.*", map[string]string{
						"value": "1.1.1.3",
						"note":  "notey notey note",
					}),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "trusted_ips.deployment_type", "only_production_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "trusted_ips.protection_mode", "trusted_ip_required"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "options_allowlist.paths.#", "1"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "options_allowlist.paths.0.value", "/bar"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.enabled_to_update", "protection_bypass_for_automation_secret"),
				),
			},
			{
				Config: cfg(testAccProjectConfigWithVercelAuthAndPasswordAndTrustedIpsAndOptionsAllowlistUpdated(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.enabled_to_start", "vercel_authentication.deployment_type", "standard_protection"),
					resource.TestCheckNoResourceAttr("vercel_project.enabled_to_start", "password_protection"),
					resource.TestCheckNoResourceAttr("vercel_project.enabled_to_start", "protection_bypass_for_automation"),
					resource.TestCheckNoResourceAttr("vercel_project.enabled_to_start", "trusted_ips"),
					resource.TestCheckNoResourceAttr("vercel_project.enabled_to_start", "protection_bypass_for_automation_secret"),

					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "vercel_authentication.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "password_protection.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "password_protection.password", "password"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "trusted_ips.addresses.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.disabled_to_start", "trusted_ips.addresses.*", map[string]string{
						"value": "1.1.1.1",
						"note":  "notey note",
					}),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "trusted_ips.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "trusted_ips.protection_mode", "trusted_ip_required"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "options_allowlist.paths.#", "1"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "options_allowlist.paths.0.value", "/foo"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_start", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.disabled_to_start", "protection_bypass_for_automation_secret"),

					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "vercel_authentication.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "password_protection.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "password_protection.password", "password2"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "trusted_ips.addresses.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.enabled_to_update", "trusted_ips.addresses.*", map[string]string{
						"value": "1.1.1.3",
						"note":  "notey notey",
					}),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "trusted_ips.deployment_type", "all_deployments"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "trusted_ips.protection_mode", "trusted_ip_required"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "protection_bypass_for_automation", "false"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "options_allowlist.paths.#", "1"),
					resource.TestCheckResourceAttr("vercel_project.enabled_to_update", "options_allowlist.paths.0.value", "/bar"),
					resource.TestCheckNoResourceAttr("vercel_project.enabled_to_update", "protection_bypass_for_automation_secret"),
				),
			},
		},
	})
}

func TestAcc_ProjectWithAutomationBypass(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.disabled_to_enabled_generated_secret", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigAutomationBypass(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.disabled_to_enabled_generated_secret", testTeam(t)),
					resource.TestCheckNoResourceAttr("vercel_project.disabled_to_enabled_generated_secret", "protection_bypass_for_automation"),
					testAccProjectExists(testClient(t), "vercel_project.disabled_to_enabled_custom_secret", testTeam(t)),
					resource.TestCheckNoResourceAttr("vercel_project.disabled_to_enabled_custom_secret", "protection_bypass_for_automation"),
					testAccProjectExists(testClient(t), "vercel_project.enabled_generated_secret_to_enabled_custom_secret", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.enabled_generated_secret_to_enabled_custom_secret", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.enabled_generated_secret_to_enabled_custom_secret", "protection_bypass_for_automation_secret"),
					testAccProjectExists(testClient(t), "vercel_project.enabled_generated_secret_to_disabled", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.enabled_generated_secret_to_disabled", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.enabled_generated_secret_to_disabled", "protection_bypass_for_automation_secret"),
					testAccProjectExists(testClient(t), "vercel_project.enabled_custom_secret_to_disabled", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.enabled_custom_secret_to_disabled", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttr("vercel_project.enabled_custom_secret_to_disabled", "protection_bypass_for_automation_secret", "12345678912345678912345678912345"),
				),
			},
			{
				Config: cfg(testAccProjectConfigAutomationBypassUpdate(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.disabled_to_enabled_generated_secret", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.disabled_to_enabled_generated_secret", "protection_bypass_for_automation_secret"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_enabled_custom_secret", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttr("vercel_project.disabled_to_enabled_custom_secret", "protection_bypass_for_automation_secret", "12345678912345678912345678912345"),
					resource.TestCheckResourceAttr("vercel_project.enabled_generated_secret_to_enabled_custom_secret", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttr("vercel_project.enabled_generated_secret_to_enabled_custom_secret", "protection_bypass_for_automation_secret", "12345678912345678912345678912345"),
					resource.TestCheckResourceAttr("vercel_project.enabled_generated_secret_to_disabled", "protection_bypass_for_automation", "false"),
					resource.TestCheckResourceAttr("vercel_project.enabled_custom_secret_to_disabled", "protection_bypass_for_automation", "false"),
				),
			},
		},
	})
}

func getProjectImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return rs.Primary.ID, nil
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func TestAcc_ProjectImport(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(projectConfigWithoutEnv(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
				),
			},
			{
				ResourceName:      "vercel_project.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getProjectImportID("vercel_project.test"),
			},
		},
	})
}

func TestAcc_ProjectEnablingAffectedProjectDeployments(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigWithoutEnableAffectedSet(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Off by default
					resource.TestCheckResourceAttr("vercel_project.test", "enable_affected_projects_deployments", "false"),
				),
			},
			{
				Config: cfg(testAccProjectConfigWithEnableAffectedTrue(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "enable_affected_projects_deployments", "true"),
				),
			},
			{
				Config: cfg(testAccProjectConfigWithEnableAffectedFalse(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "enable_affected_projects_deployments", "false"),
				),
			},
		},
	})
}

func testAccProjectExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient.GetProject(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testAccProjectDestroy(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		_, err := testClient.GetProject(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted project: %s", err)
		}

		return nil
	}
}

func testAccProjectConfigBase(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
}
`, projectSuffix)
}

func testAccProjectConfigWithResourceConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  resource_config = {
    function_default_cpu_type = "standard_legacy"
    function_default_timeout = 30
    fluid = false
  }
}
`, projectSuffix)
}

func testAccProjectConfigWithoutEnv(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
}
`, projectSuffix)
}

func testAccProjectConfigUpdated(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  environment = [
    {
      key    = "two"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "baz"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "three"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "oh_no"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "bar"
      value  = "baz"
      target = ["production"]
    },
    {
      key       = "sensitive_thing"
      value     = "bar_updated"
      target    = ["production"]
      sensitive = true
    }
  ]
  on_demand_concurrent_builds = false
  enable_preview_feedback = false
  enable_production_feedback = true
  preview_deployments_disabled = false
}
`, projectSuffix)
}

func testAccProjectConfigPreviewFeedbackConflict(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  preview_comments = true
  enable_preview_feedback = true
}
`, projectSuffix)
}

func testAccProjectConfigPreviewCommentsOnly(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  preview_comments = true
}
`, projectSuffix)
}

func testAccProjectConfigPreviewFeedbackOnly(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two-%s"
  enable_preview_feedback = true
}
`, projectSuffix)
}

func testAccProjectConfigWithVercelAuthAndPasswordAndTrustedIpsAndOptionsAllowlist(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "enabled_to_start" {
  name = "test-acc-protection-one-%[1]s"
  vercel_authentication = {
    deployment_type = "all_deployments"
  }
  password_protection = {
    deployment_type = "all_deployments"
    password           = "password"
  }
  trusted_ips = {
	addresses = [
		{
			value = "1.1.1.1"
			note = "notey note"
		}
	]
	deployment_type = "all_deployments"
	protection_mode = "trusted_ip_required"
  }
  options_allowlist = {
    paths = [
      {
        value = "/foo"
      }
    ]
  }
  protection_bypass_for_automation = true
}

resource "vercel_project" "disabled_to_start" {
  name = "test-acc-protection-two-%[1]s"
}

resource "vercel_project" "enabled_to_update" {
  name = "test-acc-protection-three-%[1]s"
  vercel_authentication = {
    deployment_type = "only_preview_deployments"
  }
  password_protection = {
    deployment_type = "only_preview_deployments"
    password           = "password"
  }
  trusted_ips = {
	addresses = [
		{
			value = "1.1.1.1"
			note = "notey notey"
		},
		{
			value = "1.1.1.3"
			note = "notey notey note"
		}
	]
	deployment_type = "only_production_deployments"
  }
  options_allowlist = {
    paths = [
      {
        value = "/bar"
      }
    ]
  }
  protection_bypass_for_automation = true
}
    `, projectSuffix)
}

func testAccProjectConfigWithVercelAuthAndPasswordAndTrustedIpsAndOptionsAllowlistUpdated(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "enabled_to_start" {
  name = "test-acc-protection-one-%[1]s"
}

resource "vercel_project" "disabled_to_start" {
  name = "test-acc-protection-two-%[1]s"
  vercel_authentication = {
    deployment_type = "standard_protection"
  }
  password_protection = {
    deployment_type = "standard_protection"
    password           = "password"
  }
  trusted_ips = {
	addresses = [
		{
			value = "1.1.1.1"
			note = "notey note"
		}
	]
	deployment_type = "standard_protection"
  }
  options_allowlist = {
    paths = [
      {
        value = "/foo"
      }
    ]
  }
  protection_bypass_for_automation = true
}

resource "vercel_project" "enabled_to_update" {
  name = "test-acc-protection-three-%[1]s"
  vercel_authentication = {
    deployment_type = "standard_protection"
  }
  password_protection = {
    deployment_type = "standard_protection"
    password           = "password2"
  }
  trusted_ips = {
	addresses = [
		{
			value = "1.1.1.3"
			note = "notey notey"
		}
	]
	deployment_type = "all_deployments"
	protection_mode = "trusted_ip_required"
  }
  options_allowlist = {
    paths = [
      {
        value = "/bar"
      }
    ]
  }
  protection_bypass_for_automation = false
}
    `, projectSuffix)
}

func testAccProjectConfigAutomationBypass(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "disabled_to_enabled_generated_secret" {
  name = "test-acc-automation-bypass-one-%[1]s"
}

resource "vercel_project" "disabled_to_enabled_custom_secret" {
  name = "test-acc-automation-bypass-two-%[1]s"
}

resource "vercel_project" "enabled_generated_secret_to_enabled_custom_secret" {
  name = "test-acc-automation-bypass-three-%[1]s"
  protection_bypass_for_automation = true
}

resource "vercel_project" "enabled_generated_secret_to_disabled" {
  name = "test-acc-automation-bypass-four-%[1]s"
  protection_bypass_for_automation = true
}

resource "vercel_project" "enabled_custom_secret_to_disabled" {
  name = "test-acc-automation-bypass-five-%[1]s"
  protection_bypass_for_automation = true
  protection_bypass_for_automation_secret = "12345678912345678912345678912345"
}
    `, projectSuffix)
}

func testAccProjectConfigAutomationBypassUpdate(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "disabled_to_enabled_generated_secret" {
  name = "test-acc-automation-bypass-one-%[1]s"
  protection_bypass_for_automation = true
}

resource "vercel_project" "disabled_to_enabled_custom_secret" {
  name = "test-acc-automation-bypass-two-%[1]s"
  protection_bypass_for_automation = true
  protection_bypass_for_automation_secret = "12345678912345678912345678912345"
}

resource "vercel_project" "enabled_generated_secret_to_enabled_custom_secret" {
  name = "test-acc-automation-bypass-three-%[1]s"
  protection_bypass_for_automation = true
  protection_bypass_for_automation_secret = "12345678912345678912345678912345"
}

resource "vercel_project" "enabled_generated_secret_to_disabled" {
  name = "test-acc-automation-bypass-four-%[1]s"
  protection_bypass_for_automation = false
}

resource "vercel_project" "enabled_custom_secret_to_disabled" {
  name = "test-acc-automation-bypass-five-%[1]s"
  protection_bypass_for_automation = false
}
    `, projectSuffix)
}

func testAccProjectConfigWithGitRepo(projectSuffix, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
  git_repository = {
    type = "github"
    repo = "%s"
    deploy_hooks = [
        {
            ref = "main"
            name = "some deploy hook"
        }
    ]
  }
  environment = [
    {
      key        = "foo"
      value      = "bar"
      target     = ["preview"]
      git_branch = "staging"
      comment    = "some comment"
    }
  ]
}
    `, projectSuffix, githubRepo)
}

func testAccProjectConfigWithGitRepoUpdated(projectSuffix, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
  public_source = false
  git_repository = {
    type = "github"
    repo = "%s"
    production_branch = "production"
    deploy_hooks = [
        {
            ref = "main"
            name = "some deploy hook"
        },
        {
            ref = "main"
            name = "some other hook"
        }
    ]
  }
  environment = [
    {
      key        = "foo"
      value      = "bar2"
      target     = ["preview"]
      comment    = "some updated comment"
    }
  ]
}
    `, projectSuffix, githubRepo)
}

func testAccProjectConfigWithGitRepoRemoved(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test_git" {
  name = "test-acc-two-%s"
  public_source = false
  environment = [
    {
      key        = "foo"
      value      = "bar2"
      target     = ["preview"]
    }
  ]
}
    `, projectSuffix)
}

func projectConfigWithoutEnv(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  build_command = "npm run build"
  dev_command = "npm run serve"
  ignore_command = "echo 'wat'"
  framework = "nextjs"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  preview_deployments_disabled = true
	resource_config = {
		function_default_regions = ["syd1"]
	}
}
`, projectSuffix)
}

func testAccProjectConfigWithoutEnableAffectedSet(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
}
`, projectSuffix)
}

func testAccProjectConfigWithEnableAffectedFalse(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  enable_affected_projects_deployments = false
}
`, projectSuffix)
}

func testAccProjectConfigWithEnableAffectedTrue(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  enable_affected_projects_deployments = true
}
`, projectSuffix)
}

func testAccProjectConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-%s"
  build_command = "npm run build"
  dev_command = "npm run serve"
  ignore_command = "echo 'wat'"
  framework = "nextjs"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  automatically_expose_system_environment_variables = true
  git_comments = {
      on_pull_request = true,
      on_commit = true
  }
  enable_preview_feedback = true
  enable_production_feedback = false
  auto_assign_custom_domains = true
  git_lfs = true
  function_failover = true
  customer_success_code_visibility = true
  git_fork_protection = true
  prioritise_production_builds = true
  directory_listing = true
  skew_protection = "7 days"
  preview_deployments_disabled = false
  oidc_token_config = {
    enabled = true
    issuer_mode = "team"
  }
  resource_config = {
		function_default_regions = ["syd1"]
		function_default_cpu_type = "standard"
		function_default_timeout = 60
  }
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "two"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "three"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "baz"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "bar"
      value  = "bar"
      target = ["production"]
    },
    {
      key    = "oh_no"
      value  = "bar"
      target = ["production"]
    },
    {
      key       = "sensitive_thing"
      value     = "bar"
      target    = ["production"]
      sensitive = true
    }
  ]
  on_demand_concurrent_builds = true
  build_machine_type = "enhanced"
}
`, projectSuffix)
}

func TestAcc_Project_OIDCToken(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectConfigOIDCToken(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project.test", "oidc_token_config.enabled", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "oidc_token_config.issuer_mode", "global"),
				),
			},
		},
	})
}

func testAccProjectConfigOIDCToken(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name      = "test-acc-oidc-%s"
  framework = "nextjs"
  oidc_token_config = {
    issuer_mode = "global"
  }
}
`, projectSuffix)
}
