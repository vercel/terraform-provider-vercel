package vercel_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func testAccDeploymentExists(testClient *client.Client, n string, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no DeploymentID is set")
		}

		_, err := testClient.GetDeployment(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func contains(items []string, i string) bool {
	for _, j := range items {
		if j == i {
			return true
		}
	}
	return false
}

func testAccEnvironmentSet(testClient *client.Client, n, teamID string, envs ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no DeploymentID is set")
		}

		dpl, err := testClient.GetDeployment(context.TODO(), rs.Primary.ID, teamID)
		if err != nil {
			return err
		}

		for _, e := range envs {
			if !contains(dpl.Build.Environment, e) {
				things := strings.Join(dpl.Build.Environment, ",")
				return fmt.Errorf("Deployment should include environment variable %s, but only had '%s'", e, things)
			}
		}

		return nil
	}
}

func noopDestroyCheck(*terraform.State) error {
	return nil
}

func TestAcc_Deployment(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	testTeamID := resource.TestCheckNoResourceAttr("vercel_deployment.test", "team_id")
	if testTeam(t) != "" {
		testTeamID = resource.TestCheckResourceAttr("vercel_deployment.test", "team_id", testTeam(t))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentConfig(projectSuffix, "")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testTeamID,
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
				),
			},
			{
				Config: cfg(deploymentWithPrebuiltProject(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testTeamID,
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithEnvironment(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentConfig(projectSuffix, `environment = {
                    FOO = "baz",
                    BAR = "qux",
                    BAZ = null
                }`)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
					testAccEnvironmentSet(testClient(t), "vercel_deployment.test", "", "FOO", "BAR"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "environment.FOO", "baz"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "environment.BAR", "qux"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithProjectSettings(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{

		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentConfig(projectSuffix, `project_settings = {
                    output_directory = ".",
                }`)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
					resource.TestCheckResourceAttr("vercel_deployment.test", "project_settings.output_directory", "."),
					// resource.TestCheckResourceAttr("vercel_deployment.test", "project_settings.build_command", "npm run build"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithRootDirectoryOverride(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccRootDirectoryOverride(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
					resource.TestCheckResourceAttr("vercel_deployment.test", "production", "true"),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithPathPrefix(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccRootDirectoryWithPathPrefix(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithDeleteOnDestroy(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	extraConfig := "delete_on_destroy = true"
	deploymentID := ""
	storeDeploymentID := func(n string, did *string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources[n]
			if !ok {
				return fmt.Errorf("not found: %s", n)
			}
			*did = rs.Primary.ID
			return nil
		}
	}
	testDeploymentGone := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			_, err := testClient(t).GetDeployment(context.TODO(), deploymentID, "")
			if err == nil {
				return fmt.Errorf("expected not_found error, but got no error")
			}
			if !client.NotFound(err) {
				return fmt.Errorf("Unexpected error checking for deleted deployment: %s", err)
			}

			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentConfig(projectSuffix, extraConfig)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
					storeDeploymentID("vercel_deployment.test", &deploymentID),
				),
			},
			{
				Config: cfg(testAccDeploymentConfigWithNoDeployment(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testDeploymentGone(),
				),
			},
		},
	})
}

func TestAcc_DeploymentWithGitSource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeployFromGitSource(projectSuffix, testGithubRepo(t), testBitbucketRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.bitbucket", testTeam(t)),
					testAccDeploymentExists(testClient(t), "vercel_deployment.github", testTeam(t)),
				),
			},
		},
	})
}

// This test executes the path where we handle the `missing_files` error.
// To do that, we need to create a new file with random contents to trigger the
// `missing_files` error. Otherwise, if the contents do not change, we will use
// the cached deployments files
func TestAcc_DeploymentWithMissingFilesPath(t *testing.T) {
	tmpFilePath := "../vercel/examples/one/random-file.html"

	createRandomFilePreConfig := func(t *testing.T) {
		min := 1
		max := 1_000_000
		randomInt := rand.Intn(max-min) + min

		fileBody := []byte(fmt.Sprintf("<html>\n<body>\nRandom integer: %d\n</body>\n</html>\n", randomInt))
		err := os.WriteFile(tmpFilePath, fileBody, 0644)
		if err != nil {
			t.Fatalf("Could not create the temporal file path %s. Error: %s", tmpFilePath, err)
		}
	}

	cleanup := func(t *testing.T) {
		if err := os.Remove(tmpFilePath); err != nil {
			t.Logf("Could not remove the random file %s. Error: %s", tmpFilePath, err)
		}
	}
	defer cleanup(t)

	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		CheckDestroy:             noopDestroyCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { createRandomFilePreConfig(t) },
				Config:    cfg(testAccWithDirectoryUpload(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentExists(testClient(t), "vercel_deployment.test", ""),
				),
			},
		},
	})
}

func testAccDeploymentConfigWithNoDeployment(projectSuffix string) string {
	return fmt.Sprintf(`
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
`, projectSuffix)
}

func deploymentWithPrebuiltProject(projectSuffix string) string {
	return fmt.Sprintf(`
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

  files       = data.vercel_prebuilt_project.test.output
  path_prefix = data.vercel_prebuilt_project.test.path
}
`, projectSuffix)
}

func testAccDeploymentConfig(projectSuffix, deploymentExtras string) string {
	return fmt.Sprintf(`
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

data "vercel_file" "index" {
    path = "examples/one/index.html"
}

data "vercel_file" "windows_line_ending" {
    path = "examples/one/windows_line_ending.png"
}

resource "vercel_deployment" "test" {
  %[2]s
  project_id = vercel_project.test.id

  files = merge(
      data.vercel_file.index.file,
      data.vercel_file.windows_line_ending.file,
  )

  production = true
}
`, projectSuffix, deploymentExtras)
}

func testAccRootDirectoryOverride(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%[1]s"
}

data "vercel_file" "index" {
    path = "../vercel/examples/one/index.html"
}

resource "vercel_deployment" "test" {
  project_id = vercel_project.test.id
  files      = data.vercel_file.index.file
  production = true
  project_settings = {
      root_directory = "vercel/example"
  }
}`, projectSuffix)
}

func testAccRootDirectoryWithPathPrefix(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%[1]s"
}

data "vercel_file" "index" {
  path = "../vercel/examples/one/index.html"
}

resource "vercel_deployment" "test" {
  project_id    = vercel_project.test.id
  files         = data.vercel_file.index.file
  path_prefix   = "../vercel/example"
}`, projectSuffix)
}

func testAccDeployFromGitSource(projectSuffix, githubRepo, bitbucketRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "github" {
  name = "test-acc-deployment-%[1]s-github"
  git_repository = {
      type = "github"
      repo = "%[2]s"
  }
}
resource "vercel_project" "bitbucket" {
  name = "test-acc-deployment-%[1]s-bitbucket"
  git_repository = {
      type = "bitbucket"
      repo = "%[3]s"
  }
}
resource "vercel_deployment" "github" {
  project_id = vercel_project.github.id
  ref        = "main"
}
resource "vercel_deployment" "bitbucket" {
  project_id = vercel_project.bitbucket.id
  ref        = "main"
}
`, projectSuffix, githubRepo, bitbucketRepo)
}

func testAccWithDirectoryUpload(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-%[1]s"
}

data "vercel_project_directory" "test" {
  path = "../vercel/examples/one"
}

resource "vercel_deployment" "test" {
  project_id    = vercel_project.test.id
  files         = data.vercel_project_directory.test.files
  path_prefix   = data.vercel_project_directory.test.path
}`, projectSuffix)
}
