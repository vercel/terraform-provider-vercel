package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestAcc_OIDCFederationPolicyResource(t *testing.T) {
	clientName := os.Getenv("VERCEL_TERRAFORM_TESTING_OIDC_CLIENT")
	if clientName == "" {
		t.Skip("VERCEL_TERRAFORM_TESTING_OIDC_CLIENT must identify an OIDC-federation-enabled CLI client")
	}

	name := acctest.RandString(16)
	resourceName := "vercel_oidc_federation_policy.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckOIDCFederationPolicyDeleted(testClient(t), resourceName),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccOIDCFederationPolicyConfig(clientName, name, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOIDCFederationPolicyExists(testClient(t), resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "team_id"),
					resource.TestCheckResourceAttr(resourceName, "name", "test-acc-"+name),
					resource.TestCheckResourceAttr(resourceName, "issuer_url", "https://token.actions.githubusercontent.com"),
					resource.TestCheckResourceAttr(resourceName, "claims.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "permissions.*", "*"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: oidcFederationPolicyImportID(resourceName),
			},
			{
				Config: cfg(testAccOIDCFederationPolicyConfig(clientName, name, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOIDCFederationPolicyExists(testClient(t), resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "test-acc-"+name+"-updated"),
				),
			},
		},
	})
}

func oidcFederationPolicyImportID(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}
		if resourceState.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}
		return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.ID), nil
	}
}

func testCheckOIDCFederationPolicyExists(testClient *client.Client, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		_, err := testClient.GetOIDCFederationPolicy(context.Background(), resourceState.Primary.ID, resourceState.Primary.Attributes["team_id"])
		return err
	}
}

func testCheckOIDCFederationPolicyDeleted(testClient *client.Client, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		_, err := testClient.GetOIDCFederationPolicy(context.Background(), resourceState.Primary.ID, resourceState.Primary.Attributes["team_id"])
		if client.NotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking deleted OIDC federation policy: %w", err)
		}
		return fmt.Errorf("OIDC federation policy %s still exists", resourceState.Primary.ID)
	}
}

func testAccOIDCFederationPolicyConfig(clientName, name string, updated bool) string {
	policyName := "test-acc-" + name
	if updated {
		policyName += "-updated"
	}
	return fmt.Sprintf(`
resource "vercel_oidc_federation_policy" "test" {
  name       = %[1]q
  client     = %[2]q
  issuer_url = "https://token.actions.githubusercontent.com"

  claims = [
    {
      name = "aud"
      values = [{ value = "https://vercel.com/api/login/oauth/token" }]
    },
    {
      name = "repository"
      values = [{ value = "vercel/terraform-provider-vercel-%[3]s" }]
    }
  ]

  permissions = ["*"]
  resources = {
    project_ids = ["*"]
  }
}
`, policyName, clientName, name)
}
