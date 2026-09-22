package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func testAccProjectDeploymentCheckExists(testClient *client.Client, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		_, err := testClient.GetProjectDeploymentCheck(
			context.Background(),
			resourceState.Primary.Attributes["project_id"],
			resourceState.Primary.ID,
			resourceState.Primary.Attributes["team_id"],
		)
		return err
	}
}

func projectDeploymentCheckImportID(name string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("not found: %s", name)
		}
		return fmt.Sprintf(
			"%s/%s/%s",
			resourceState.Primary.Attributes["team_id"],
			resourceState.Primary.Attributes["project_id"],
			resourceState.Primary.ID,
		), nil
	}
}

func TestAcc_ProjectDeploymentCheck(t *testing.T) {
	resourceName := "vercel_project_deployment_check.example"
	nameSuffix := acctest.RandString(16)
	var checkID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDeploymentCheckConfig(nameSuffix, "Deployment check", 300)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "Deployment check"),
					resource.TestCheckResourceAttr(resourceName, "blocks", "deployment-alias"),
					resource.TestCheckResourceAttr(resourceName, "source.kind", "webhook"),
					testAccProjectDeploymentCheckExists(testClient(t), resourceName),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    projectDeploymentCheckImportID(resourceName),
				ImportStateVerifyIdentifierAttribute: "project_id",
			},
			{
				Config: cfg(testAccProjectDeploymentCheckConfig(nameSuffix, "Updated deployment check", 600)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Updated deployment check"),
					resource.TestCheckResourceAttr(resourceName, "timeout", "600"),
					resource.TestCheckResourceAttrWith(resourceName, "id", func(value string) error {
						checkID = value
						return nil
					}),
					testAccProjectDeploymentCheckExists(testClient(t), resourceName),
				),
			},
			{
				Config: cfg(testAccProjectDeploymentCheckProjectConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.example", testTeam(t)),
					func(state *terraform.State) error {
						if checkID == "" {
							return fmt.Errorf("no deployment check ID was captured before removal")
						}
						project, ok := state.RootModule().Resources["vercel_project.example"]
						if !ok {
							return fmt.Errorf("parent project is missing from state")
						}
						_, err := testClient(t).GetProjectDeploymentCheck(context.Background(), project.Primary.ID, checkID, testTeam(t))
						if err == nil {
							return fmt.Errorf("deployment check %s still exists after removal", checkID)
						}
						if !client.NotFound(err) {
							return fmt.Errorf("checking for deleted deployment check: %w", err)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccProjectDeploymentCheckConfig(nameSuffix, checkName string, timeout int) string {
	return testAccProjectDeploymentCheckProjectConfig(nameSuffix) + fmt.Sprintf(`
resource "vercel_project_deployment_check" "example" {
  project_id = vercel_project.example.name
  name       = %q
  requires   = "deployment-url"
  blocks     = "deployment-alias"
  targets    = ["production"]
  timeout    = %d

  source = {
    kind = "webhook"
  }
}
`, checkName, timeout)
}

func testAccProjectDeploymentCheckProjectConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
  name = "test-acc-project-deployment-check-%s"
}
`, nameSuffix)
}
