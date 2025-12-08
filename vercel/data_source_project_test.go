package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDataSourceConfig(name, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project.test", "name", "test-acc-"+name),
					resource.TestCheckResourceAttr("data.vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "framework", "nextjs"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "root_directory", "ui/src"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "vercel_authentication.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "password_protection.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "trusted_ips.addresses.#", "1"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "automatically_expose_system_environment_variables", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_project.test", "trusted_ips.addresses.*", map[string]string{
						"value": "1.1.1.1",
					}),
					resource.TestCheckResourceAttr("data.vercel_project.test", "trusted_ips.deployment_type", "only_production_deployments"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "trusted_ips.protection_mode", "trusted_ip_required"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "options_allowlist.paths.#", "1"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "options_allowlist.paths.0.value", "/api"),

					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_project.test", "environment.*", map[string]string{
						"key":   "foo",
						"value": "bar",
					}),
					resource.TestCheckTypeSetElemAttr("data.vercel_project.test", "environment.0.target.*", "production"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "git_comments.on_pull_request", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "git_comments.on_commit", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "preview_comments", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "enable_preview_feedback", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "enable_production_feedback", "false"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "preview_deployments_disabled", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "auto_assign_custom_domains", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "git_lfs", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "function_failover", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "customer_success_code_visibility", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "git_fork_protection", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "prioritise_production_builds", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "directory_listing", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "skew_protection", "7 days"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "resource_config.function_default_cpu_type", "standard_legacy"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "resource_config.function_default_timeout", "30"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "resource_config.function_default_regions.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.vercel_project.test", "resource_config.function_default_regions.*", "sfo1"),
					resource.TestCheckTypeSetElemAttr("data.vercel_project.test", "resource_config.function_default_regions.*", "iad1"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "oidc_token_config.issuer_mode", "team"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "on_demand_concurrent_builds", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "build_machine_type", "enhanced"),
				),
			},
		},
	})
}

func testAccProjectDataSourceConfig(name, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%s"
  build_command = "npm run build"
  dev_command = "npm run serve"
  framework = "nextjs"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  preview_deployments_disabled = true
  vercel_authentication = {
    deployment_type = "standard_protection"
  }
  password_protection = {
    password = "foo"
    deployment_type = "standard_protection"
  }
  trusted_ips = {
	addresses = [
		{
			value = "1.1.1.1"
		}
	]
	deployment_type = "only_production_deployments"
	protection_mode = "trusted_ip_required"
  }
  options_allowlist = {
    paths = [
      {
        value = "/api"
      }
    ]
  }
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    }
  ]
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
  git_repository = {
    type = "github"
    repo = "%[2]s"
    deploy_hooks = [
        {
            ref = "main"
            name = "some deploy hook"
        }
    ]
  }
  resource_config = {
    function_default_cpu_type = "standard_legacy"
    function_default_timeout = 30
    function_default_regions = ["sfo1", "iad1"]
	fluid = false
  }
  oidc_token_config = {
    enabled = true
    issuer_mode = "team"
  }
  on_demand_concurrent_builds = true
  build_machine_type = "enhanced"
}

data "vercel_project" "test" {
    name = vercel_project.test.name
}
`, name, githubRepo)
}

func TestAcc_ProjectDataSourcePreviewDeploymentSuffix(t *testing.T) {
	name := acctest.RandString(16)
	domain := testDomain(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDataSourceConfigWithPreviewDeploymentSuffix(name, domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project.test", "name", "test-acc-suffix-ds-"+name),
					resource.TestCheckResourceAttr("data.vercel_project.test", "preview_deployment_suffix", domain),
				),
			},
		},
	})
}

func testAccProjectDataSourceConfigWithPreviewDeploymentSuffix(name, domain string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name                       = "test-acc-suffix-ds-%s"
  preview_deployment_suffix  = "%s"
}

data "vercel_project" "test" {
    name = vercel_project.test.name
}
`, name, domain)
}
