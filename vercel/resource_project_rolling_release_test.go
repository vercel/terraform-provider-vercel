package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
	"github.com/vercel/terraform-provider-vercel/v3/vercel"
)

// TestRollingReleaseRequestConversion tests the request conversion logic
func TestRollingReleaseRequestConversion(t *testing.T) {
	// Test manual-approval advancement type
	info := vercel.RollingReleaseInfo{
		AdvancementType: types.StringValue("manual-approval"),
		ProjectID:       types.StringValue("test-project"),
		TeamID:          types.StringValue("test-team"),
		Stages: types.ListValueMust(vercel.RollingReleaseStageElementType, []attr.Value{
			types.ObjectValueMust(vercel.RollingReleaseStageElementType.AttrTypes, map[string]attr.Value{
				"target_percentage": types.Int64Value(20),
				"duration":          types.Int64Null(),
			}),
			types.ObjectValueMust(vercel.RollingReleaseStageElementType.AttrTypes, map[string]attr.Value{
				"target_percentage": types.Int64Value(50),
				"duration":          types.Int64Null(),
			}),
		}),
	}

	request, diags := info.ToCreateRollingReleaseRequest()
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags)
	}

	// Should have 3 stages: 20%, 50%, and 100%
	if len(request.RollingRelease.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(request.RollingRelease.Stages))
	}

	// Check that the 100% stage is present
	found100 := false
	for _, stage := range request.RollingRelease.Stages {
		if stage.TargetPercentage == 100 {
			found100 = true
			if stage.RequireApproval {
				t.Error("100% stage should not require approval")
			}
			break
		}
	}
	if !found100 {
		t.Error("100% stage not found in request")
	}

	// Test automatic advancement type
	info2 := vercel.RollingReleaseInfo{
		AdvancementType: types.StringValue("automatic"),
		ProjectID:       types.StringValue("test-project"),
		TeamID:          types.StringValue("test-team"),
		Stages: types.ListValueMust(vercel.RollingReleaseStageElementType, []attr.Value{
			types.ObjectValueMust(vercel.RollingReleaseStageElementType.AttrTypes, map[string]attr.Value{
				"target_percentage": types.Int64Value(30),
				"duration":          types.Int64Value(60),
			}),
		}),
	}

	request2, diags2 := info2.ToCreateRollingReleaseRequest()
	if diags2.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags2)
	}

	// Should have 2 stages: 30% and 100%
	if len(request2.RollingRelease.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(request2.RollingRelease.Stages))
	}

	// Check that the 100% stage is present
	found100_2 := false
	for _, stage := range request2.RollingRelease.Stages {
		if stage.TargetPercentage == 100 {
			found100_2 = true
			if stage.RequireApproval {
				t.Error("100% stage should not require approval")
			}
			break
		}
	}
	if !found100_2 {
		t.Error("100% stage not found in request")
	}
}

func getRollingReleaseImportId(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"]), nil
	}
}

func testAccProjectRollingReleaseExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetRollingRelease(context.TODO(), rs.Primary.Attributes["project_id"], teamID)
		return err
	}
}

func TestAcc_ProjectRollingRelease(t *testing.T) {
	resourceName := "vercel_project_rolling_release.example"
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRollingReleasesConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr(resourceName, "stages.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "20",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "50",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "100",
					}),
				),
			},
			// Now, import the existing resource
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    getRollingReleaseImportId(resourceName),
				ImportStateVerifyIdentifierAttribute: "project_id",
			},
			// Then update to new configuration
			{
				Config: cfg(testAccProjectRollingReleasesConfigUpdated(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr(resourceName, "stages.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "20",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "50",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "80",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "100",
					}),
				),
			},
			// Then update to new configuration
			{
				Config: cfg(testAccProjectRollingReleasesConfigUpdatedAutomatic(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project.example", "id"),
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "advancement_type", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "stages.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "20",
						"duration":          "10",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "50",
						"duration":          "10",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "80",
						"duration":          "10",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "100",
					}),
				),
			},
			{
				Config: cfg(fmt.Sprintf(`
					resource "vercel_project" "example" {
						name = "test-acc-rr-auto-duration-%s"
					}
					resource "vercel_project_rolling_release" "example" {
						project_id = vercel_project.example.id
						advancement_type = "automatic"
						stages = [
							{
								target_percentage = 30
								// Duration is omitted here for the first stage
							},
							{
								target_percentage = 70
								duration          = 30 // Explicit duration for a middle stage
							}
						]
					}
					`, nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "advancement_type", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "stages.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "30",
						"duration":          "60", // Asserting the default value
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "70",
						"duration":          "30", // Asserting the explicit value
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "stages.*", map[string]string{
						"target_percentage": "100",
						// Duration for the last stage is expected to be null or not present
					}),
				),
			},
		},
	})
}

func testAccProjectRollingReleasesConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-rolling-releases-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	advancement_type = "manual-approval"
	stages = [
		{
			target_percentage = 20
		},
		{
			target_percentage = 50
		}
	]
}
`, nameSuffix)
}

func testAccProjectRollingReleasesConfigUpdated(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-rolling-releases-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	advancement_type = "manual-approval"
	stages = [
		{
			target_percentage = 20
		},
		{
			target_percentage = 50
		},
		{
			target_percentage = 80
		}
	]
}
`, nameSuffix)
}
func testAccProjectRollingReleasesConfigUpdatedAutomatic(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-rolling-releases-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	advancement_type = "automatic"
	stages = [
		{
			target_percentage = 20
			duration          = 10
		},
		{
			target_percentage = 50
			duration          = 10
		},
		{
			target_percentage = 80
			duration          = 10
		}
	]
}
`, nameSuffix)
}
