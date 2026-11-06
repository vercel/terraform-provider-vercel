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

// The fixture must be an existing test-mode Enterprise Flex organisation accessible to
// VERCEL_API_TOKEN. Platform children can be deleted; Enterprise children cannot.
func TestAcc_OrganisationTeamResource(t *testing.T) {
	organisationID := os.Getenv("VERCEL_TERRAFORM_TESTING_ORGANISATION")
	if organisationID == "" {
		t.Skip("VERCEL_TERRAFORM_TESTING_ORGANISATION is required for organisation team acceptance tests")
	}
	if token := os.Getenv("VERCEL_ORGANISATION_API_TOKEN"); token != "" {
		t.Setenv("VERCEL_API_TOKEN", token)
	}
	organisationClient := client.New(os.Getenv("VERCEL_API_TOKEN"))
	suffix := acctest.RandString(16)
	slug := "test-acc-org-vtest314-" + suffix
	const resourceName = "vercel_organisation_team.test"
	var teamID string
	config := func(name, slug, description string) string {
		return fmt.Sprintf(`
resource "vercel_organisation_team" "test" {
 organisation_id = %q
 name = %q
 slug = %q
 billing_plan = "platform"
}
resource "vercel_team_config" "test" {
 id = vercel_organisation_team.test.id
 name = vercel_organisation_team.test.name
 slug = vercel_organisation_team.test.slug
 description = %q
}
resource "vercel_project" "test" {
 team_id = vercel_organisation_team.test.id
 name = %q
}
`, organisationID, name, slug, description, "test-acc-org-project-"+suffix)
	}
	check := func(name, slug, description string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "organisation_id", organisationID),
			resource.TestCheckResourceAttr(resourceName, "billing_plan", "platform"),
			resource.TestCheckResourceAttr(resourceName, "name", name),
			resource.TestCheckResourceAttr(resourceName, "slug", slug),
			resource.TestCheckResourceAttrPair(resourceName, "id", "vercel_team_config.test", "id"),
			resource.TestCheckResourceAttrPair(resourceName, "id", "vercel_project.test", "team_id"),
			func(s *terraform.State) error {
				rs, ok := s.RootModule().Resources[resourceName]
				if !ok || rs.Primary.ID == "" {
					return fmt.Errorf("organisation team is missing from state")
				}
				if teamID != "" && teamID != rs.Primary.ID {
					return fmt.Errorf("team was replaced during an in-place update")
				}
				teamID = rs.Primary.ID
				team, err := organisationClient.GetTeam(context.Background(), teamID)
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
			// The root-scoped token loses access to a child after deletion, so check the organisation link.
			deadline := time.Now().Add(time.Minute)
			for {
				_, err := organisationClient.GetOrganisationTeam(context.Background(), organisationID, teamID)
				if client.NotFound(err) {
					return nil
				}
				if err != nil {
					return err
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
				return organisationID + "/" + teamID, nil
			}},
			{PreConfig: func() {
				_, err := organisationClient.UpdateTeam(context.Background(), client.UpdateTeamRequest{TeamID: teamID, Name: "Changed outside Terraform"})
				if err != nil {
					t.Fatal(err)
				}
			}, Config: updated, Check: check("Renamed Product vtest314", slug+"-updated", "Updated by acceptance test")},
		},
	})
}
