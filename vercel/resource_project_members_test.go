package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectMembers(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectMembersConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
				),
			},
			{
				Config: cfg(testAccProjectMembersConfigUpdated(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "2"),
				),
			},
			{
				Config: cfg(testAccProjectMembersConfigUpdatedAgain(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_members.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
				),
			},
		},
	})
}

// TestAcc_ProjectMembersAddByEmail is a regression test for adding a second
// member by email while an existing, fully-resolved member is left unchanged.
// Previously the UseStateForUnknown plan modifiers on the nested set attributes
// grafted the existing member's computed user_id/username onto the new member,
// causing "Provider produced inconsistent result after apply". Each step also
// re-plans (the default empty-plan check) to ensure no perpetual diff.
func TestAcc_ProjectMembersAddByEmail(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectMembersByEmailConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project_members.test", "members.*", map[string]string{
						"email": "doug+test2@vercel.com",
						"role":  "PROJECT_VIEWER",
					}),
				),
			},
			{
				// Add a second member by email; the first member's config is
				// byte-for-byte unchanged. Both must be present and distinct.
				Config: cfg(testAccProjectMembersByEmailAddedConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project_members.test", "members.*", map[string]string{
						"email": "doug+test2@vercel.com",
						"role":  "PROJECT_VIEWER",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project_members.test", "members.*", map[string]string{
						"email": "doug+test3@vercel.com",
						"role":  "PROJECT_VIEWER",
					}),
				),
			},
		},
	})
}

func TestAcc_ProjectMembersUpdateRoleByEmail(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectMembersByEmailConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project_members.test", "members.*", map[string]string{
						"email": "doug+test2@vercel.com",
						"role":  "PROJECT_VIEWER",
					}),
				),
			},
			{
				Config: cfg(testAccProjectMembersByEmailRoleUpdatedConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_members.test", "members.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project_members.test", "members.*", map[string]string{
						"email": "doug+test2@vercel.com",
						"role":  "PROJECT_DEVELOPER",
					}),
				),
			},
		},
	})
}

func testAccProjectMembersByEmailConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-email-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [{
    email = "doug+test2@vercel.com"
    role  = "PROJECT_VIEWER"
  }]
}
`, projectSuffix)
}

func testAccProjectMembersByEmailRoleUpdatedConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-email-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [{
    email = "doug+test2@vercel.com"
    role  = "PROJECT_DEVELOPER"
  }]
}
`, projectSuffix)
}

func testAccProjectMembersByEmailAddedConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-email-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [
    {
      email = "doug+test2@vercel.com"
      role  = "PROJECT_VIEWER"
    },
    {
      email = "doug+test3@vercel.com"
      role  = "PROJECT_VIEWER"
    }
  ]
}
`, projectSuffix)
}

func testAccProjectMembersConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [{
    email = "doug+test2@vercel.com"
    role  = "PROJECT_VIEWER"
  }]
}
`, projectSuffix)
}

func testAccProjectMembersConfigUpdated(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [{
      email = "doug+test2@vercel.com"
      role  = "PROJECT_DEVELOPER"
    },
    {
      email = "doug+test3@vercel.com"
      role  = "PROJECT_VIEWER"
    }
  ]
}
`, projectSuffix)
}

func testAccProjectMembersConfigUpdatedAgain(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project-members-%[1]s"
}

resource "vercel_project_members" "test" {
  project_id = vercel_project.test.id

  members = [
    {
      email = "doug+test3@vercel.com"
      role  = "PROJECT_VIEWER"
    }
  ]
}
`, projectSuffix)
}
