package vercel_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_DevelopmentEnvironmentVariables(t *testing.T) {
	for _, kind := range []string{"single", "bulk", "inline"} {
		for _, classification := range []string{"secret", "legacy", "config"} {
			t.Run(kind+"/"+classification, func(t *testing.T) {
				testAccDevelopmentEnvironmentVariable(t, kind, classification, false)
			})
		}
	}
	t.Run("single/write_only", func(t *testing.T) {
		testAccDevelopmentEnvironmentVariable(t, "single", "secret", true)
	})
}

func testAccDevelopmentEnvironmentVariable(t *testing.T, kind, classification string, writeOnly bool) {
	name := "test-acc-dev-env-" + acctest.RandString(16)
	sensitive := classification != "config"
	visibility := ""
	if classification != "legacy" {
		visibility = fmt.Sprintf("visibility = %q", classification)
	}
	config := func(value, comment string, version int, preview bool) string {
		targets := `["development"]`
		if preview {
			targets = `["development", "preview"]`
		}
		valueAttribute := fmt.Sprintf("value = %q", value)
		if writeOnly {
			valueAttribute = fmt.Sprintf("value_wo = %q\nvalue_wo_version = %d", value, version)
		}
		env := fmt.Sprintf(`key = "DEV_VAR"
%s
target = %s
sensitive = %t
%s
comment = %q`, valueAttribute, targets, sensitive, visibility, comment)
		project := fmt.Sprintf(`resource "vercel_project" "development" { name = %q }`, name)
		switch kind {
		case "single":
			return cfg(project + "\n" + `resource "vercel_project_environment_variable" "development" {` + "\nproject_id = vercel_project.development.id\n" + env + "\n}")
		case "bulk":
			return cfg(project + "\n" + `resource "vercel_project_environment_variables" "development" {` + "\nproject_id = vercel_project.development.id\nvariables = [{\n" + env + "\n}]\n}")
		default:
			return cfg(fmt.Sprintf("resource \"vercel_project\" \"development\" {\nname = %q\nenvironment = [{\n%s\n}]\n}", name, env))
		}
	}
	var variableID string
	check := func(value, comment string, version int, preview bool) resource.TestCheckFunc {
		expectedType, expectedVisibility := "encrypted", "config"
		if sensitive {
			expectedType, expectedVisibility = "sensitive", "secret"
		}
		targets := []string{"development"}
		if preview {
			targets = append(targets, "preview")
		}
		checks := []resource.TestCheckFunc{func(s *terraform.State) error {
			project := s.RootModule().Resources["vercel_project.development"]
			if project == nil {
				return fmt.Errorf("project missing from state")
			}
			envs, err := testClient(t).GetEnvironmentVariables(context.Background(), project.Primary.ID, testTeam(t))
			if err != nil {
				return err
			}
			var count int
			for _, env := range envs {
				if env.Key != "DEV_VAR" {
					continue
				}
				count++
				sort.Strings(env.Target)
				if env.Type != expectedType || !reflect.DeepEqual(env.Target, targets) || env.Comment != comment {
					return fmt.Errorf("unexpected API classification, targets, or comment for DEV_VAR")
				}
				if env.Visibility != nil && *env.Visibility != expectedVisibility {
					return fmt.Errorf("unexpected API visibility %q", *env.Visibility)
				}
				if !sensitive && env.Value != value {
					return fmt.Errorf("readable Config value did not update")
				}
				if kind == "single" {
					if variableID != "" && variableID != env.ID {
						return fmt.Errorf("single variable was replaced during update")
					}
					variableID = env.ID
				}
			}
			if count != 1 {
				return fmt.Errorf("got %d DEV_VAR variables, want 1", count)
			}
			return nil
		}}
		if kind == "single" {
			address := "vercel_project_environment_variable.development"
			checks = append(checks,
				resource.TestCheckResourceAttr(address, "sensitive", fmt.Sprint(sensitive)),
				resource.TestCheckResourceAttr(address, "visibility", expectedVisibility),
				resource.TestCheckResourceAttr(address, "comment", comment),
				resource.TestCheckResourceAttr(address, "target.#", fmt.Sprint(len(targets))),
				resource.TestCheckTypeSetElemAttr(address, "target.*", "development"))
			if writeOnly {
				checks = append(checks, resource.TestCheckNoResourceAttr(address, "value"), resource.TestCheckNoResourceAttr(address, "value_wo"), resource.TestCheckResourceAttr(address, "value_wo_version", fmt.Sprint(version)))
			} else {
				checks = append(checks, resource.TestCheckResourceAttr(address, "value", value))
			}
		} else {
			address, attribute := "vercel_project.development", "environment.*"
			if kind == "bulk" {
				address, attribute = "vercel_project_environment_variables.development", "variables.*"
			}
			checks = append(checks, resource.TestCheckTypeSetElemNestedAttrs(address, attribute, map[string]string{
				"key": "DEV_VAR", "value": value, "sensitive": fmt.Sprint(sensitive), "visibility": expectedVisibility,
				"comment": comment, "target.#": fmt.Sprint(len(targets)), "target.0": "development",
			}))
		}
		return resource.ComposeAggregateTestCheckFunc(checks...)
	}
	initial := config("initial-development-value", "initial", 1, false)
	rotated := config("rotated-development-value", "rotated", 2, false)
	mixed := config("rotated-development-value", "metadata-only", 2, true)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.development", testTeam(t)),
		Steps: []resource.TestStep{
			{Config: initial, Check: check("initial-development-value", "initial", 1, false)},
			{Config: rotated, Check: check("rotated-development-value", "rotated", 2, false)},
			{Config: mixed, Check: check("rotated-development-value", "metadata-only", 2, true)},
			{RefreshState: true, Check: check("rotated-development-value", "metadata-only", 2, true)},
			{Config: mixed, PlanOnly: true, Check: check("rotated-development-value", "metadata-only", 2, true)},
		},
	})
}
