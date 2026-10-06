package vercel_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

func TestDevelopmentEnvironmentVariablePlanning(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(vercel.New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, resourceName := range []string{"vercel_project_environment_variable", "vercel_project_environment_variables", "vercel_shared_environment_variable", "vercel_project"} {
		for _, classification := range []struct {
			name       string
			sensitive  bool
			visibility string
		}{
			{"secret", true, "secret"}, {"legacy secret", true, ""}, {"config", false, "config"},
		} {
			t.Run(resourceName+"/"+classification.name, func(t *testing.T) {
				typ := schemas.ResourceSchemas[resourceName].ValueType()
				rootType := typ.(tftypes.Object)
				envType := rootType
				nested := ""
				switch resourceName {
				case "vercel_project":
					nested = "environment"
				case "vercel_project_environment_variables":
					nested = "variables"
				}
				if nested != "" {
					envType = rootType.AttributeTypes[nested].(tftypes.Set).ElementType.(tftypes.Object)
				}
				env := developmentSecretObject(envType)
				env["key"] = tftypes.NewValue(tftypes.String, "EXAMPLE")
				env["value"] = tftypes.NewValue(tftypes.String, "placeholder")
				env["target"] = tftypes.NewValue(envType.AttributeTypes["target"], []tftypes.Value{tftypes.NewValue(tftypes.String, "development"), tftypes.NewValue(tftypes.String, "preview")})
				env["sensitive"] = tftypes.NewValue(tftypes.Bool, classification.sensitive)
				if classification.visibility != "" {
					if _, ok := envType.AttributeTypes["visibility"]; ok {
						env["visibility"] = tftypes.NewValue(tftypes.String, classification.visibility)
					}
				}
				config := env
				if nested != "" {
					config = developmentSecretObject(rootType)
					config[nested] = tftypes.NewValue(rootType.AttributeTypes[nested], []tftypes.Value{tftypes.NewValue(envType, env)})
				}
				if _, ok := rootType.AttributeTypes["project_id"]; ok {
					config["project_id"] = tftypes.NewValue(tftypes.String, "prj_example")
				}
				if resourceName == "vercel_project" {
					config["name"] = tftypes.NewValue(tftypes.String, "development-secret-test")
				}
				configValue := developmentSecretDynamicValue(t, typ, tftypes.NewValue(typ, config))
				validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: resourceName, Config: configValue})
				if err != nil {
					t.Fatal(err)
				}
				assertDevelopmentSecretDiagnostics(t, validation.Diagnostics)
				response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
					TypeName: resourceName, Config: configValue, ProposedNewState: configValue,
					PriorState: developmentSecretDynamicValue(t, typ, tftypes.NewValue(typ, nil)),
				})
				if err != nil {
					t.Fatal(err)
				}
				assertDevelopmentSecretDiagnostics(t, response.Diagnostics)
				if response.PlannedState == nil {
					t.Fatal("missing planned state")
				}
				planned, err := response.PlannedState.Unmarshal(typ)
				if err != nil {
					t.Fatal(err)
				}
				var plannedRoot map[string]tftypes.Value
				if err := planned.As(&plannedRoot); err != nil {
					t.Fatal(err)
				}
				if nested != "" {
					var items []tftypes.Value
					if err := plannedRoot[nested].As(&items); err != nil {
						t.Fatal(err)
					}
					if len(items) != 1 {
						t.Fatalf("planned %d variables, want 1", len(items))
					}
					if err := items[0].As(&plannedRoot); err != nil {
						t.Fatal(err)
					}
				}
				var sensitive bool
				if err := plannedRoot["sensitive"].As(&sensitive); err != nil {
					t.Fatal(err)
				}
				if sensitive != classification.sensitive {
					t.Fatalf("planned sensitive = %t, want %t", sensitive, classification.sensitive)
				}
			})
		}
	}
}

func developmentSecretObject(typ tftypes.Object) map[string]tftypes.Value {
	result := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for key, attrType := range typ.AttributeTypes {
		result[key] = tftypes.NewValue(attrType, nil)
	}
	return result
}

func developmentSecretDynamicValue(t *testing.T, typ tftypes.Type, value tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	result, err := tfprotov6.NewDynamicValue(typ, value)
	if err != nil {
		t.Fatal(err)
	}
	return &result
}

func assertDevelopmentSecretDiagnostics(t *testing.T, diags []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diag := range diags {
		if diag.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("%s: %s", diag.Summary, diag.Detail)
		}
	}
	if t.Failed() {
		t.FailNow()
	}
}
