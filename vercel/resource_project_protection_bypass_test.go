package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestAcc_ProjectProtectionBypass(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	customSecret := acctest.RandString(32)
	replacementSecret := acctest.RandString(32)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Single bypass — first one is automatically the env-var default.
			{
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassExists(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.first"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "scope", "automation-bypass"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "note", "first bypass"),
					resource.TestCheckResourceAttrSet("vercel_project_protection_bypass.first", "secret"),
					resource.TestCheckResourceAttrSet("vercel_project_protection_bypass.first", "created_at"),
				),
			},
			// Add a second bypass with caller-supplied secret. First remains the env-var default.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, customSecret, "second bypass", false, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "false"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "secret", customSecret),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "note", "second bypass"),
				),
			},
			// Promote the second bypass to the env-var default. The API atomically demotes the first.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, customSecret, "second bypass", true, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "false"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "true"),
				),
			},
			// Update the note on the second bypass in place.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, customSecret, "renamed bypass", true, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "note", "renamed bypass"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "true"),
				),
			},
			// Import the second bypass by team_id/project_id/secret.
			{
				ResourceName:      "vercel_project_protection_bypass.second",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_protection_bypass.second"]
					if !ok {
						return "", fmt.Errorf("not found: vercel_project_protection_bypass.second")
					}
					projectID := rs.Primary.Attributes["project_id"]
					teamID := rs.Primary.Attributes["team_id"]
					secret := rs.Primary.Attributes["secret"]
					if teamID == "" {
						return fmt.Sprintf("%s/%s", projectID, secret), nil
					}
					return fmt.Sprintf("%s/%s/%s", teamID, projectID, secret), nil
				},
			},
			// Changing the secret triggers replacement: the old bypass is revoked
			// and a fresh one is created with the new secret. is_env_var persists.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, replacementSecret, "renamed bypass", true, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "secret", replacementSecret),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "false"),
					testAccProjectProtectionBypassRevoked(testClient(t), "vercel_project.test", customSecret),
				),
			},
			// Remove the `second` bypass while it is the env-var default.
			// The provider must promote `first` to isEnvVar=true before revoking
			// `second`, otherwise the API rejects the revoke with a 400 because
			// the project would be left with no env-var default.
			{
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassRevoked(testClient(t), "vercel_project.test", replacementSecret),
					// The live API check is what matters here: the sibling's Delete
					// triggered a direct PATCH that promotes `first`. Its own state
					// is not refreshed between the apply and the Check, so we deliberately
					// assert against the server rather than state.
					testAccProjectProtectionBypassIsEnvVarDefault(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.first"),
				),
			},
			// Remove the last bypass on the project — verify it's revoked cleanly.
			{
				Config: cfg(testAccProjectProtectionBypassEmpty(projectSuffix)),
			},
		},
	})
}

func testAccProjectProtectionBypassEmpty(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-%[1]s"
}
`, projectSuffix)
}

func testAccProjectProtectionBypassExists(testClient *client.Client, projectResource, bypassResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		project, ok := s.RootModule().Resources[projectResource]
		if !ok {
			return fmt.Errorf("not found: %s", projectResource)
		}
		bypass, ok := s.RootModule().Resources[bypassResource]
		if !ok {
			return fmt.Errorf("not found: %s", bypassResource)
		}
		secret := bypass.Primary.Attributes["secret"]
		teamID := bypass.Primary.Attributes["team_id"]
		_, err := testClient.GetProtectionBypass(context.TODO(), project.Primary.ID, teamID, secret)
		return err
	}
}

func testAccProjectProtectionBypassRevoked(testClient *client.Client, projectResource, secret string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		project, ok := s.RootModule().Resources[projectResource]
		if !ok {
			return fmt.Errorf("not found: %s", projectResource)
		}
		teamID := project.Primary.Attributes["team_id"]
		_, err := testClient.GetProtectionBypass(context.TODO(), project.Primary.ID, teamID, secret)
		if err == nil {
			return fmt.Errorf("expected bypass %s to be revoked but it still exists", secret)
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error reading revoked bypass: %s", err)
		}
		return nil
	}
}

func testAccProjectProtectionBypassSingle(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first bypass"
}
`, projectSuffix)
}

func testAccProjectProtectionBypassDouble(projectSuffix, customSecret, secondNote string, secondIsEnvVar, firstIsEnvVar bool) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first bypass"
  is_env_var = %[4]t
}

resource "vercel_project_protection_bypass" "second" {
  project_id = vercel_project.test.id
  note       = "%[3]s"
  secret     = "%[2]s"
  is_env_var = %[5]t
}
`, projectSuffix, customSecret, secondNote, firstIsEnvVar, secondIsEnvVar)
}

func testAccProjectProtectionBypassIsEnvVarDefault(testClient *client.Client, projectResource, bypassResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p, ok := s.RootModule().Resources[projectResource]
		if !ok {
			return fmt.Errorf("not found: %s", projectResource)
		}
		b, ok := s.RootModule().Resources[bypassResource]
		if !ok {
			return fmt.Errorf("not found: %s", bypassResource)
		}
		bypass, err := testClient.GetProtectionBypass(
			context.TODO(),
			p.Primary.ID,
			b.Primary.Attributes["team_id"],
			b.Primary.Attributes["secret"],
		)
		if err != nil {
			return err
		}
		if bypass.IsEnvVar == nil || !*bypass.IsEnvVar {
			return fmt.Errorf("expected %s to be the env-var default, but isEnvVar=%v", bypassResource, bypass.IsEnvVar)
		}
		return nil
	}
}

// Covers the Create path where is_env_var=true is requested at creation time
// on a non-first bypass. The API assigns isEnvVar=false because a default
// already exists, so the provider must issue a follow-up promotion to honor
// the plan. Without that branch, state and config would disagree.
func TestAcc_ProjectProtectionBypass_PromoteOnCreate(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	customSecret := acctest.RandString(32)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// First bypass alone — the API marks it as the env-var default.
			// is_env_var is left unset so the computed value matches reality.
			{
				Config: cfg(testAccProjectProtectionBypassPromoteOnCreateFirstOnly(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
				),
			},
			// Add a second bypass with is_env_var=true set at creation. The API
			// initially makes it non-default (first still holds the slot), so the
			// provider must issue a follow-up promotion to honour the plan.
			{
				Config: cfg(testAccProjectProtectionBypassPromoteOnCreateBoth(projectSuffix, customSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.promoted", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.promoted", "secret", customSecret),
					testAccProjectProtectionBypassIsEnvVarDefault(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.promoted"),
				),
			},
		},
	})
}

func testAccProjectProtectionBypassPromoteOnCreateFirstOnly(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-promote-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first"
}
`, projectSuffix)
}

func testAccProjectProtectionBypassPromoteOnCreateBoth(projectSuffix, customSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-promote-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first"
}

resource "vercel_project_protection_bypass" "promoted" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "promote at creation"
  is_env_var = true
}
`, projectSuffix, customSecret)
}

// Covers out-of-band revocation. A bypass deleted via the API outside of
// Terraform should be removed from state on refresh and re-created on the
// next apply. Without that, `terraform plan` silently stays clean against
// a broken project.
func TestAcc_ProjectProtectionBypass_ExternalDrift(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	var capturedProjectID, capturedTeamID, capturedSecret string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassExists(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.first"),
					func(s *terraform.State) error {
						b := s.RootModule().Resources["vercel_project_protection_bypass.first"]
						capturedProjectID = b.Primary.Attributes["project_id"]
						capturedTeamID = b.Primary.Attributes["team_id"]
						capturedSecret = b.Primary.Attributes["secret"]
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					err := testClient(t).DeleteProtectionBypass(context.TODO(), client.DeleteProtectionBypassRequest{
						TeamID:    capturedTeamID,
						ProjectID: capturedProjectID,
						Secret:    capturedSecret,
					})
					if err != nil {
						t.Fatalf("failed to revoke bypass out-of-band: %s", err)
					}
				},
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassExists(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.first"),
					testAccProjectProtectionBypassRevoked(testClient(t), "vercel_project.test", capturedSecret),
					func(s *terraform.State) error {
						newSecret := s.RootModule().Resources["vercel_project_protection_bypass.first"].Primary.Attributes["secret"]
						if newSecret == capturedSecret {
							return fmt.Errorf("expected secret to change after external revoke, got same value %q", newSecret)
						}
						return nil
					},
				),
			},
		},
	})
}

// Covers the two-segment import form (project_id/secret) used when a default
// team is configured on the provider. The three-segment form is exercised in
// the main test; this one makes sure splitInto2Or3 handles the shorter id.
func TestAcc_ProjectProtectionBypass_TwoSegmentID(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
			},
			{
				ResourceName: "vercel_project_protection_bypass.first",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_protection_bypass.first"]
					if !ok {
						return "", fmt.Errorf("not found: vercel_project_protection_bypass.first")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["project_id"], rs.Primary.Attributes["secret"]), nil
				},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["scope"] != "automation-bypass" {
						return fmt.Errorf("expected scope=automation-bypass on import, got %q", attrs["scope"])
					}
					if attrs["is_env_var"] != "true" {
						return fmt.Errorf("expected is_env_var=true on import, got %q", attrs["is_env_var"])
					}
					if attrs["secret"] == "" {
						return fmt.Errorf("expected secret to be set on import")
					}
					if attrs["project_id"] == "" {
						return fmt.Errorf("expected project_id to be set on import")
					}
					return nil
				},
			},
		},
	})
}

// Covers coexistence of the deprecated project-level attribute with the new
// resource on the same project. Guards convertResponseToProject's identity-match:
// once the legacy attribute is bound to a specific bypass, adding or promoting
// sibling bypasses via vercel_project_protection_bypass must not flip its state.
func TestAcc_ProjectProtectionBypass_CoexistWithDeprecated(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	extraSecret := acctest.RandString(32)

	var legacySecretAfterCreate string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Step 1: project-managed bypass created first, exposed via the legacy attr.
			// It holds the env-var default because it is the only bypass on the project.
			{
				Config: cfg(testAccProjectProtectionBypassCoexistProjectOnly(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "protection_bypass_for_automation", "true"),
					resource.TestCheckResourceAttrSet("vercel_project.test", "protection_bypass_for_automation_secret"),
					func(s *terraform.State) error {
						legacySecretAfterCreate = s.RootModule().Resources["vercel_project.test"].Primary.Attributes["protection_bypass_for_automation_secret"]
						if legacySecretAfterCreate == "" {
							return fmt.Errorf("expected legacy secret to be populated")
						}
						return nil
					},
				),
			},
			// Step 2: add a sibling via the new resource. The legacy attr must still
			// point at the original bypass — not the new one — across refresh.
			{
				Config: cfg(testAccProjectProtectionBypassCoexistBoth(projectSuffix, extraSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.extra", "secret", extraSecret),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.extra", "is_env_var", "false"),
					testAccProjectProtectionBypassExists(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.extra"),
					func(s *terraform.State) error {
						legacy := s.RootModule().Resources["vercel_project.test"].Primary.Attributes["protection_bypass_for_automation_secret"]
						if legacy != legacySecretAfterCreate {
							return fmt.Errorf("legacy secret flipped after adding sibling: was %q, now %q", legacySecretAfterCreate, legacy)
						}
						if legacy == extraSecret {
							return fmt.Errorf("legacy secret must not equal the new resource's secret")
						}
						return nil
					},
				),
			},
			// Step 3: promote the sibling to the env-var default. The legacy attr
			// still points at the original project-managed bypass (identity-match),
			// even though it is no longer the env-var default.
			{
				Config: cfg(testAccProjectProtectionBypassCoexistPromoted(projectSuffix, extraSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.extra", "is_env_var", "true"),
					func(s *terraform.State) error {
						legacy := s.RootModule().Resources["vercel_project.test"].Primary.Attributes["protection_bypass_for_automation_secret"]
						if legacy != legacySecretAfterCreate {
							return fmt.Errorf("legacy secret moved to sibling after promotion: was %q, now %q", legacySecretAfterCreate, legacy)
						}
						return nil
					},
				),
			},
		},
	})
}

// Covers the identity-match miss → isEnvVar=true fallback. After the
// project-managed bypass is revoked out-of-band, the next refresh can no longer
// identity-match; it must deterministically pick the env-var default (the only
// automation-bypass with isEnvVar=true) instead of hitting random map iteration.
func TestAcc_ProjectProtectionBypass_CoexistFallbackToDefault(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	extraSecret := acctest.RandString(32)

	var capturedProjectID, capturedTeamID, capturedLegacySecret string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Two bypasses: legacy-created (will be revoked out-of-band) and
			// new-resource-created (explicitly the env-var default).
			{
				Config: cfg(testAccProjectProtectionBypassCoexistPromoted(projectSuffix, extraSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						p := s.RootModule().Resources["vercel_project.test"]
						capturedProjectID = p.Primary.ID
						capturedTeamID = p.Primary.Attributes["team_id"]
						capturedLegacySecret = p.Primary.Attributes["protection_bypass_for_automation_secret"]
						if capturedLegacySecret == "" {
							return fmt.Errorf("expected legacy secret to be set before out-of-band revoke")
						}
						return nil
					},
				),
			},
			// Revoke the legacy-bound bypass out-of-band, then re-apply the same config.
			// identity-match misses; fallback must pick the env-var default (extraSecret)
			// because that is now the only bypass with isEnvVar=true.
			{
				PreConfig: func() {
					err := testClient(t).DeleteProtectionBypass(context.TODO(), client.DeleteProtectionBypassRequest{
						TeamID:    capturedTeamID,
						ProjectID: capturedProjectID,
						Secret:    capturedLegacySecret,
					})
					if err != nil {
						t.Fatalf("failed to revoke legacy bypass out-of-band: %s", err)
					}
				},
				Config: cfg(testAccProjectProtectionBypassCoexistPromoted(projectSuffix, extraSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// After fallback, the legacy attr points at the env-var default,
					// which is the sibling bypass. The project resource's next Update
					// will re-provision its own bypass — here we just pin the state.
					func(s *terraform.State) error {
						legacy := s.RootModule().Resources["vercel_project.test"].Primary.Attributes["protection_bypass_for_automation_secret"]
						if legacy == capturedLegacySecret {
							return fmt.Errorf("legacy secret %q still present in state after out-of-band revoke", legacy)
						}
						return nil
					},
				),
			},
		},
	})
}

// Covers the new solo-bypass error path. A user who writes is_env_var = false
// on the only bypass for a project should get an actionable error instead of
// Terraform's generic "inconsistent result" diagnostic.
func TestAcc_ProjectProtectionBypass_RejectsSoloFalse(t *testing.T) {
	projectSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config:      cfg(testAccProjectProtectionBypassSoloFalse(projectSuffix)),
				ExpectError: regexp.MustCompile(`(?s)Invalid is_env_var = false for a solo protection bypass`),
			},
			// The project still applies cleanly once the invalid resource is removed,
			// which confirms the rejected bypass was cleaned up server-side (a lingering
			// solo bypass with is_env_var=true would not block this apply, but leaking
			// state would be visible in the next test run's team if the cleanup failed).
			{
				Config: cfg(testAccProjectProtectionBypassEmpty(projectSuffix)),
			},
		},
	})
}

// Covers the update path where the current env-var default is set to
// is_env_var=false without any sibling resource being updated in the same
// apply. The provider must promote a live replacement instead of rewriting
// local state to false while the server still keeps this bypass as default.
func TestAcc_ProjectProtectionBypass_DemoteWithoutSiblingPlanChange(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	secondSecret := acctest.RandString(32)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectProtectionBypassDemoteWithoutSiblingPlanChangeInitial(projectSuffix, secondSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "false"),
				),
			},
			{
				Config: cfg(testAccProjectProtectionBypassDemoteWithoutSiblingPlanChangeUpdated(projectSuffix, secondSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "false"),
					testAccProjectProtectionBypassIsEnvVarDefault(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.second"),
				),
			},
		},
	})
}

// Covers deleting a bypass that becomes the env-var default earlier in the
// same apply. The second delete must consult live server state rather than the
// resource's stale pre-plan is_env_var value.
func TestAcc_ProjectProtectionBypass_DeletePromotedSibling(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	firstSecret := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	secondSecret := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	thirdSecret := "cccccccccccccccccccccccccccccccc"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectProtectionBypassDeletePromotedSiblingInitial(projectSuffix, firstSecret, secondSecret, thirdSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "false"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.third", "is_env_var", "false"),
				),
			},
			{
				Config: cfg(testAccProjectProtectionBypassDeletePromotedSiblingUpdated(projectSuffix, thirdSecret)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassRevoked(testClient(t), "vercel_project.test", firstSecret),
					testAccProjectProtectionBypassRevoked(testClient(t), "vercel_project.test", secondSecret),
					testAccProjectProtectionBypassIsEnvVarDefault(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.third"),
				),
			},
		},
	})
}

func testAccProjectProtectionBypassSoloFalse(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-solo-%[1]s"
}

resource "vercel_project_protection_bypass" "solo" {
  project_id = vercel_project.test.id
  note       = "solo false"
  is_env_var = false
}
`, projectSuffix)
}

func testAccProjectProtectionBypassDemoteWithoutSiblingPlanChangeInitial(projectSuffix, secondSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-demote-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first default"
}

resource "vercel_project_protection_bypass" "second" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "replacement"
}
`, projectSuffix, secondSecret)
}

func testAccProjectProtectionBypassDemoteWithoutSiblingPlanChangeUpdated(projectSuffix, secondSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-demote-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first default"
  is_env_var = false
}

resource "vercel_project_protection_bypass" "second" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "replacement"
}
`, projectSuffix, secondSecret)
}

func testAccProjectProtectionBypassDeletePromotedSiblingInitial(projectSuffix, firstSecret, secondSecret, thirdSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-delete-promoted-%[1]s"
}

resource "vercel_project_protection_bypass" "second" {
  project_id = vercel_project.test.id
  secret     = "%[3]s"
  note       = "second"
}

resource "vercel_project_protection_bypass" "third" {
  project_id = vercel_project.test.id
  secret     = "%[4]s"
  note       = "third"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "first"
  is_env_var = true
  depends_on = [
    vercel_project_protection_bypass.second,
    vercel_project_protection_bypass.third,
  ]
}
`, projectSuffix, firstSecret, secondSecret, thirdSecret)
}

func testAccProjectProtectionBypassDeletePromotedSiblingUpdated(projectSuffix, thirdSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-delete-promoted-%[1]s"
}

resource "vercel_project_protection_bypass" "third" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "third"
}
`, projectSuffix, thirdSecret)
}

func testAccProjectProtectionBypassCoexistProjectOnly(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name                             = "test-acc-bypass-coexist-%[1]s"
  protection_bypass_for_automation = true
}
`, projectSuffix)
}

func testAccProjectProtectionBypassCoexistBoth(projectSuffix, extraSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name                             = "test-acc-bypass-coexist-%[1]s"
  protection_bypass_for_automation = true
}

resource "vercel_project_protection_bypass" "extra" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "managed by new resource"
  is_env_var = false
  depends_on = [vercel_project.test]
}
`, projectSuffix, extraSecret)
}

func testAccProjectProtectionBypassCoexistPromoted(projectSuffix, extraSecret string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name                             = "test-acc-bypass-coexist-%[1]s"
  protection_bypass_for_automation = true
}

resource "vercel_project_protection_bypass" "extra" {
  project_id = vercel_project.test.id
  secret     = "%[2]s"
  note       = "promoted sibling"
  is_env_var = true
  depends_on = [vercel_project.test]
}
`, projectSuffix, extraSecret)
}
