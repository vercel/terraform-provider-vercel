package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testCheckMicrofrontendGroupExists(teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetMicrofrontendGroup(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckMicrofrontendGroupDeleted(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetMicrofrontendGroup(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !(err.Error() == "microfrontend group not found") {
			return fmt.Errorf("Unexpected error checking for deleted microfrontend group: %s", err)
		}

		return nil
	}
}

func TestAcc_MicrofrontendGroupResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckMicrofrontendGroupDeleted("vercel_microfrontend_group.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: `
                    resource "vercel_microfrontend_group" "test" {
                        name = "foo"
                    }
                `,
				ExpectError: regexp.MustCompile(`The argument "projects" is required, but no definition was found.`),
			},
			{
				Config: fmt.Sprintf(`
					resource "vercel_project" "test" {
						name = "test-acc-project-%[1]s"
						%[2]s
					}
					resource "vercel_project" "test-2" {
						name = "test-acc-project-2-%[1]s"
						%[2]s
					}
                    resource "vercel_microfrontend_group" "test" {
                        name = "foo-%[1]s"
						%[2]s
						projects = {
							(vercel_project.test.id) = {}
							(vercel_project.test-2.id) = {}
						}
                    }
                `, name, teamIDConfig()),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: fmt.Sprintf(`
					resource "vercel_project" "test" {
						name = "test-acc-project-%[1]s"
						%[2]s
					}
					resource "vercel_project" "test-2" {
						name = "test-acc-project-2-%[1]s"
						%[2]s
					}
                    resource "vercel_microfrontend_group" "test" {
                        name = "foo-%[1]s"
						%[2]s
						projects = {
							(vercel_project.test.id) = {
								is_default_app = true
							}
							(vercel_project.test-2.id) = {
								is_default_app = true
							}
						}
                    }
                `, name, teamIDConfig()),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: fmt.Sprintf(`
				resource "vercel_project" "test" {
				  name = "test-acc-project-%[1]s"
				  %[2]s
				}
				resource "vercel_project" "test-2" {
				  name = "test-acc-project-2-%[1]s"
				  %[2]s
				}
				resource "vercel_microfrontend_group" "test" {
					name         = "test-acc-microfrontend-group-%[1]s"
					%[2]s
					projects = {
						(vercel_project.test.id) = {
							is_default_app = true
						}
						(vercel_project.test-2.id) = {}
					}
				}
				`, name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckMicrofrontendGroupExists(testTeam(), "vercel_microfrontend_group.test"),
					resource.TestCheckResourceAttr("vercel_microfrontend_group.test", "name", "test-acc-microfrontend-group-"+name),
					resource.TestCheckResourceAttrSet("vercel_microfrontend_group.test", "id"),
					resource.TestCheckResourceAttr("vercel_microfrontend_group.test", "projects.%", "2"),
				),
			},
		},
	})
}
