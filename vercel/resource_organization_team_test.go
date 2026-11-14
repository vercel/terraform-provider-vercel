package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// The fixture must be an existing test-mode Enterprise Flex organization accessible to
// VERCEL_API_TOKEN. Platform children can be deleted; Enterprise children cannot.
func TestAcc_OrganizationTeamResource(t *testing.T) {
	organizationID := os.Getenv("VERCEL_TERRAFORM_TESTING_ORGANIZATION")
	if organizationID == "" {
		t.Skip("VERCEL_TERRAFORM_TESTING_ORGANIZATION is required for organization team acceptance tests")
	}
	if token := os.Getenv("VERCEL_ORGANIZATION_API_TOKEN"); token != "" {
		t.Setenv("VERCEL_API_TOKEN", token)
	}
	organizationClient := client.New(os.Getenv("VERCEL_API_TOKEN"))
	suffix := acctest.RandString(16)
	slug := "test-acc-org-vtest314-" + suffix
	const resourceName = "vercel_organization_team.test"
	var teamID string
	config := func(name, slug, description string) string {
		return fmt.Sprintf(`
resource "vercel_organization_team" "test" {
 organization_id = %q
 name = %q
 slug = %q
 billing_plan = "platform"
}
resource "vercel_team_config" "test" {
 id = vercel_organization_team.test.id
 name = vercel_organization_team.test.name
 slug = vercel_organization_team.test.slug
 description = %q
}
resource "vercel_project" "test" {
 team_id = vercel_organization_team.test.id
 name = %q
}
`, organizationID, name, slug, description, "test-acc-org-project-"+suffix)
	}
	check := func(name, slug, description string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "organization_id", organizationID),
			resource.TestCheckResourceAttr(resourceName, "billing_plan", "platform"),
			resource.TestCheckResourceAttr(resourceName, "name", name),
			resource.TestCheckResourceAttr(resourceName, "slug", slug),
			resource.TestCheckResourceAttrPair(resourceName, "id", "vercel_team_config.test", "id"),
			resource.TestCheckResourceAttrPair(resourceName, "id", "vercel_project.test", "team_id"),
			func(s *terraform.State) error {
				rs, ok := s.RootModule().Resources[resourceName]
				if !ok || rs.Primary.ID == "" {
					return fmt.Errorf("organization team is missing from state")
				}
				if teamID != "" && teamID != rs.Primary.ID {
					return fmt.Errorf("team was replaced during an in-place update")
				}
				teamID = rs.Primary.ID
				team, err := organizationClient.GetTeam(context.Background(), teamID)
				if err != nil {
					return err
				}
				if team.Name != name || team.Slug != slug || team.Description == nil || *team.Description != description {
					return fmt.Errorf("team settings do not match the applied configuration")
				}
				return nil
			},
		)
	}
	updated := config("Renamed Product vtest314", slug+"-updated", "Updated by acceptance test")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			if teamID == "" {
				return nil
			}
			// Check the organization list directly: confirming absence through a child
			// endpoint requires access that a root-scoped token loses after deletion.
			deadline := time.Now().Add(time.Minute)
			for {
				listed, err := testOrganizationTeamListed(context.Background(), organizationClient, organizationID, teamID)
				if err != nil {
					return err
				}
				if !listed {
					return nil
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("team %s still exists after destroy", teamID)
				}
				time.Sleep(time.Second)
			}
		},
		Steps: []resource.TestStep{
			{Config: config("Product vtest314", slug, "Created by acceptance test"), Check: check("Product vtest314", slug, "Created by acceptance test")},
			{Config: updated, Check: check("Renamed Product vtest314", slug+"-updated", "Updated by acceptance test")},
			{ResourceName: resourceName, ImportState: true, ImportStateVerify: true, ImportStateIdFunc: func(s *terraform.State) (string, error) {
				return organizationID + "/" + teamID, nil
			}},
			{PreConfig: func() {
				name := "Changed outside Terraform"
				_, err := organizationClient.UpdateTeam(context.Background(), client.UpdateTeamRequest{TeamID: teamID, Name: &name})
				if err != nil {
					t.Fatal(err)
				}
			}, Config: updated, Check: check("Renamed Product vtest314", slug+"-updated", "Updated by acceptance test")},
		},
	})
}
