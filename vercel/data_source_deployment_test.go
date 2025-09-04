package vercel_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_DeploymentDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentDataSourceConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "url"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "production", "true"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "domains.#", "2"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "meta.build", "123"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "meta.env", "staging"),

					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "url"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "production", "true"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "domains.#", "2"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "meta.build", "123"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "meta.env", "staging"),
				),
			}},
	})
}

func testAccDeploymentDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "vercel_deployment" "by_id" {
   id = vercel_deployment.test.id
}

data "vercel_deployment" "by_url" {
   id = vercel_deployment.test.url
}

resource "vercel_project" "test" {
  name = "test-acc-deployment-%[1]s"
  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}

data "vercel_prebuilt_project" "test" {
    path = "examples/two"
}

resource "vercel_deployment" "test" {
  project_id = vercel_project.test.id

  production  = true
  files       = data.vercel_prebuilt_project.test.output
  path_prefix = data.vercel_prebuilt_project.test.path

  meta = {
    build = "123"
    env   = "staging"
  }
}
`, name)
}

func TestAcc_DeploymentDataSourceWithCustomEnvironment(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentDataSourceWithCustomEnvironmentConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "url"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "custom_environment_id"),
					resource.TestCheckResourceAttrPair("data.vercel_deployment.test", "custom_environment_id", "vercel_custom_environment.test", "id"),
				),
			},
		},
	})
}

func testAccDeploymentDataSourceWithCustomEnvironmentConfig(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-ds-custom-env-%[1]s"
}

resource "vercel_custom_environment" "test" {
  project_id = vercel_project.test.id
  name = "test-custom-env-%[1]s"
  description = "test custom environment for deployment data source"
}

data "vercel_prebuilt_project" "test" {
    path = "examples/two"
}

resource "vercel_deployment" "test" {
  project_id = vercel_project.test.id
  custom_environment_id = vercel_custom_environment.test.id

  files       = data.vercel_prebuilt_project.test.output
  path_prefix = data.vercel_prebuilt_project.test.path
}

data "vercel_deployment" "test" {
   id = vercel_deployment.test.id
}
`, name)
}

// runGitDS executes a git command in the given directory for the data source tests.
func runGitDS(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func TestAcc_DeploymentDataSource_GitMetadata(t *testing.T) {
	name := acctest.RandString(8)

	// Prepare a real git repo in the examples/one directory
	repoDir := filepath.Join("..", "vercel", "examples", "one")
	_ = os.RemoveAll(filepath.Join(repoDir, ".git"))
	runGitDS(t, repoDir, "init")
	runGitDS(t, repoDir, "checkout", "-b", "main")
	runGitDS(t, repoDir, "config", "user.email", "test@example.com")
	runGitDS(t, repoDir, "config", "user.name", "Test User")
	runGitDS(t, repoDir, "add", ".")
	runGitDS(t, repoDir, "commit", "-m", "e2e: git metadata ds test")
	runGitDS(t, repoDir, "remote", "add", "origin", fmt.Sprintf("https://github.com/%s", testGithubRepo(t)))
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(repoDir, ".git"))
	})

	cfgHCL := fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-ds-gitmeta-%[1]s"
  git_repository = {
    type = "github"
    repo = "%[2]s"
  }
}

data "vercel_project_directory" "test" {
  path = "%[3]s"
}

resource "vercel_deployment" "test" {
  project_id  = vercel_project.test.id
  files       = data.vercel_project_directory.test.files
  path_prefix = data.vercel_project_directory.test.path
}

data "vercel_deployment" "by_id" {
  id = vercel_deployment.test.id
}

data "vercel_deployment" "by_url" {
  id = vercel_deployment.test.url
}
`, name, testGithubRepo(t), filepath.Clean(repoDir))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(cfgHCL),
				Check: resource.ComposeAggregateTestCheckFunc(
					// by_id assertions
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "meta.githubCommitSha"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "meta.githubCommitMessage"),
					// by_url assertions
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "meta.githubCommitSha"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "meta.githubCommitMessage"),
				),
			},
		},
	})
}
