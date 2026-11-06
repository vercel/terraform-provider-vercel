package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProjectNullableBooleanPlans(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["vercel_project"].ValueType().(tftypes.Object)
	fields := func() map[string]tftypes.Value {
		values := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for name, attrType := range typ.AttributeTypes {
			values[name] = tftypes.NewValue(attrType, nil)
		}
		values["name"] = tftypes.NewValue(tftypes.String, "original")
		values["node_version"] = tftypes.NewValue(tftypes.String, "22.x")
		for _, name := range []string{"auto_assign_custom_domains", "git_fork_protection", "manage_passport", "manage_deployment_protection"} {
			values[name] = tftypes.NewValue(tftypes.Bool, true)
		}
		return values
	}
	dynamic := func(values map[string]tftypes.Value) *tfprotov6.DynamicValue {
		t.Helper()
		var raw any
		if values != nil {
			raw = values
		}
		value, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, raw))
		if err != nil {
			t.Fatal(err)
		}
		return &value
	}
	for _, attribute := range []struct{ name, alias string }{
		{name: "automatically_expose_system_environment_variables"},
		{name: "enable_affected_projects_deployments"},
		{name: "enable_production_feedback"},
		{name: "enable_preview_feedback", alias: "preview_comments"},
		{name: "preview_comments", alias: "enable_preview_feedback"},
	} {
		t.Run(attribute.name, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				prior         any
				configured    any
				alias         any
				want          any
				create        bool
				unchangedName bool
			}{
				{name: "null no-op", unchangedName: true},
				{name: "null unrelated update"},
				{name: "true unrelated update", prior: true, want: true},
				{name: "false unrelated update", prior: false, want: false},
				{name: "explicit false", prior: true, configured: false, want: false},
				{name: "explicit true", prior: false, configured: true, want: true},
				{name: "unknown configuration with known state", prior: true, configured: tftypes.UnknownValue, want: tftypes.UnknownValue},
				{name: "unknown configuration with null state", configured: tftypes.UnknownValue, want: tftypes.UnknownValue},
				{name: "creation keeps server default unknown", create: true, want: tftypes.UnknownValue},
				{name: "creation explicit false", create: true, configured: false, want: false},
				{name: "creation explicit true", create: true, configured: true, want: true},
				{name: "configured alias with null state", alias: false, want: tftypes.UnknownValue},
				{name: "configured alias with known state", prior: false, alias: true, want: tftypes.UnknownValue},
				{name: "unknown alias", prior: true, alias: tftypes.UnknownValue, want: tftypes.UnknownValue},
			} {
				if tc.alias != nil && attribute.alias == "" {
					continue
				}
				t.Run(tc.name, func(t *testing.T) {
					prior, config, proposed := fields(), fields(), fields()
					prior["id"] = tftypes.NewValue(tftypes.String, "prj_test")
					proposed["id"] = prior["id"]
					if !tc.unchangedName {
						config["name"] = tftypes.NewValue(tftypes.String, "renamed")
						proposed["name"] = config["name"]
					}
					prior[attribute.name] = tftypes.NewValue(tftypes.Bool, tc.prior)
					config[attribute.name] = tftypes.NewValue(tftypes.Bool, tc.configured)
					proposed[attribute.name] = prior[attribute.name]
					if tc.configured != nil {
						proposed[attribute.name] = config[attribute.name]
					}
					if attribute.alias != "" {
						prior[attribute.alias] = prior[attribute.name]
						config[attribute.alias] = tftypes.NewValue(tftypes.Bool, tc.alias)
						proposed[attribute.alias] = prior[attribute.alias]
						if tc.alias != nil {
							proposed[attribute.alias] = config[attribute.alias]
						}
					}
					if tc.create {
						prior = nil
						proposed["id"] = tftypes.NewValue(tftypes.String, nil)
					}
					response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
						TypeName: "vercel_project", Config: dynamic(config),
						PriorState: dynamic(prior), ProposedNewState: dynamic(proposed),
					})
					if err != nil {
						t.Fatal(err)
					}
					for _, diagnostic := range response.Diagnostics {
						if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
							t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
						}
					}
					planned, err := response.PlannedState.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					var values map[string]tftypes.Value
					if err := planned.As(&values); err != nil {
						t.Fatal(err)
					}
					if want := tftypes.NewValue(tftypes.Bool, tc.want); !values[attribute.name].Equal(want) {
						t.Fatalf("planned %s = %s, want %s", attribute.name, values[attribute.name], want)
					}
				})
			}
		})
	}
}
