package vercel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestPassportPatchSemantics(t *testing.T) {
	ctx := context.Background()
	configured := &client.Passport{ConnectorID: "scl_oidc", DeploymentType: "preview"}
	for _, tc := range []struct {
		name  string
		value types.Object
		want  string
	}{
		{"omitted", types.ObjectNull(passportAttrTypes), ""},
		{"unknown", types.ObjectUnknown(passportAttrTypes), ""},
		{"disabled", passportState(nil), "null"},
		{"enabled", passportState(configured), `{"connectorId":"scl_oidc","deploymentType":"preview"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := projectForUpdateRequestTests()
			project.Passport = tc.value
			projectRequest, diags := project.toUpdateProjectRequest(ctx, project.Name.ValueString())
			if diags.HasError() {
				t.Fatal(diags)
			}
			team := TeamConfig{DefaultPassport: tc.value, RemoteCaching: types.ObjectNull(remoteCachingAttrTypes), Saml: types.ObjectNull(samlAttrTypes)}
			teamRequest, diags := team.toUpdateTeamRequest(ctx, "", types.StringNull())
			if diags.HasError() {
				t.Fatal(diags)
			}
			for key, request := range map[string]any{"passport": projectRequest, "defaultPassport": teamRequest} {
				body, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(body, &fields); err != nil {
					t.Fatal(err)
				}
				if got := string(fields[key]); got != tc.want {
					t.Fatalf("%s = %s, want %s", key, got, tc.want)
				}
			}
			if got := project.RequiresUpdateAfterCreation(); got != (tc.want != "") {
				t.Fatalf("post-create update = %v", got)
			}
		})
	}
}

func TestPassportReadAndSchemaDecoding(t *testing.T) {
	ctx := context.Background()
	for _, passport := range []*client.Passport{nil, {ConnectorID: "scl_oidc", DeploymentType: "all_except_custom_domains"}} {
		project, err := convertResponseToProject(ctx, client.ProjectResponse{ID: "prj_1", Passport: passport}, nullProject, nil)
		if err != nil {
			t.Fatal(err)
		}
		projectData, err := convertResponseToProjectDataSource(ctx, client.ProjectResponse{ID: "prj_1", Passport: passport}, nullProject, nil)
		if err != nil {
			t.Fatal(err)
		}
		team, diags := convertResponseToTeamConfig(ctx, client.Team{ID: "team_1", DefaultPassport: passport}, types.MapNull(types.StringType))
		if diags.HasError() {
			t.Fatal(diags)
		}
		for _, got := range []types.Object{project.Passport, projectData.Passport, team.DefaultPassport} {
			if !got.Equal(passportState(passport)) {
				t.Fatalf("Passport read = %s", got)
			}
		}
	}
	// Exercise full schema/model decoding so resource-only fields cannot break data sources.
	projectResourceSchema := &resource.SchemaResponse{}
	newProjectResource().Schema(ctx, resource.SchemaRequest{}, projectResourceSchema)
	projectDataSchema := &datasource.SchemaResponse{}
	newProjectDataSource().Schema(ctx, datasource.SchemaRequest{}, projectDataSchema)
	teamResourceSchema := &resource.SchemaResponse{}
	newTeamConfigResource().Schema(ctx, resource.SchemaRequest{}, teamResourceSchema)
	teamDataSchema := &datasource.SchemaResponse{}
	newTeamConfigDataSource().Schema(ctx, datasource.SchemaRequest{}, teamDataSchema)
	for _, tc := range []struct {
		state tfsdk.State
		model any
	}{
		{tfsdk.State{Schema: projectResourceSchema.Schema, Raw: tftypes.NewValue(projectResourceSchema.Schema.Type().TerraformType(ctx), nil)}, &Project{}},
		{tfsdk.State{Schema: projectDataSchema.Schema, Raw: tftypes.NewValue(projectDataSchema.Schema.Type().TerraformType(ctx), nil)}, &ProjectDataSource{}},
		{tfsdk.State{Schema: teamResourceSchema.Schema, Raw: tftypes.NewValue(teamResourceSchema.Schema.Type().TerraformType(ctx), nil)}, &TeamConfig{}},
		{tfsdk.State{Schema: teamDataSchema.Schema, Raw: tftypes.NewValue(teamDataSchema.Schema.Type().TerraformType(ctx), nil)}, &TeamConfigData{}},
	} {
		objectType := tc.state.Raw.Type().(tftypes.Object)
		fields := map[string]tftypes.Value{}
		for name, fieldType := range objectType.AttributeTypes {
			fields[name] = tftypes.NewValue(fieldType, nil)
		}
		tc.state.Raw = tftypes.NewValue(objectType, fields)
		if diags := tc.state.Get(ctx, tc.model); diags.HasError() {
			t.Fatal(diags)
		}
	}
	for _, diags := range []diag.Diagnostics{projectResourceSchema.Schema.ValidateImplementation(ctx), teamResourceSchema.Schema.ValidateImplementation(ctx), projectDataSchema.Schema.ValidateImplementation(ctx), teamDataSchema.Schema.ValidateImplementation(ctx)} {
		if diags.HasError() {
			t.Fatal(diags)
		}
	}

}

func TestPassportValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		enabled   types.Bool
		connector types.String
		scope     types.String
		invalid   bool
	}{
		{"default enabled missing connector", types.BoolNull(), types.StringNull(), types.StringNull(), true},
		{"enabled", types.BoolValue(true), types.StringValue("scl_oidc"), types.StringValue("preview"), false},
		{"disabled", types.BoolValue(false), types.StringNull(), types.StringNull(), false},
		{"disabled connector", types.BoolValue(false), types.StringValue("scl_oidc"), types.StringNull(), true},
		{"disabled scope", types.BoolValue(false), types.StringNull(), types.StringValue("preview"), true},
		{"unknown enabled", types.BoolUnknown(), types.StringNull(), types.StringNull(), false},
		{"unknown connector", types.BoolValue(true), types.StringUnknown(), types.StringNull(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := types.ObjectValueMust(passportAttrTypes, map[string]attr.Value{"enabled": tc.enabled, "connector_id": tc.connector, "deployment_type": tc.scope})
			resp := &validator.ObjectResponse{}
			passportValidator{}.ValidateObject(context.Background(), validator.ObjectRequest{Path: path.Root("passport"), ConfigValue: value}, resp)
			if resp.Diagnostics.HasError() != tc.invalid {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
		})
	}
}

func TestPassportTeamConfigV0Upgrade(t *testing.T) {
	ctx := context.Background()
	res := &teamConfigResource{}
	upgrade := res.UpgradeState(ctx)[0]
	priorType := upgrade.PriorSchema.Type().TerraformType(ctx).(tftypes.Object)
	fields := map[string]tftypes.Value{}
	for name, fieldType := range priorType.AttributeTypes {
		fields[name] = tftypes.NewValue(fieldType, nil)
	}
	fields["id"] = tftypes.NewValue(tftypes.String, "team_1")
	priorState := tfsdk.State{Schema: *upgrade.PriorSchema, Raw: tftypes.NewValue(priorType, fields)}
	currentSchema := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, currentSchema)
	resp := &resource.UpgradeStateResponse{State: tfsdk.State{Schema: currentSchema.Schema}}
	upgrade.StateUpgrader(ctx, resource.UpgradeStateRequest{State: &priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var state TeamConfig
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "team_1" || !state.DefaultPassport.IsNull() {
		t.Fatalf("upgraded state = %#v", state)
	}
}

func TestPassportDefaultsPreserveOmittedBlocks(t *testing.T) {
	ctx := context.Background()
	stateValue := passportState(&client.Passport{ConnectorID: "scl_oidc", DeploymentType: "preview"})
	configValue := types.ObjectValueMust(passportAttrTypes, map[string]attr.Value{
		"enabled": types.BoolNull(), "connector_id": types.StringValue("scl_oidc"), "deployment_type": types.StringNull(),
	})
	for _, tc := range []struct {
		name               string
		config, plan, want types.Object
	}{
		{"omitted keeps preview", types.ObjectNull(passportAttrTypes), stateValue, stateValue},
		{"omitted keeps disabled", types.ObjectNull(passportAttrTypes), passportState(nil), passportState(nil)},
		{"configured defaults", configValue, configValue, passportState(&client.Passport{ConnectorID: "scl_oidc", DeploymentType: "all"})},
		{"disabled clears computed connector", types.ObjectValueMust(passportAttrTypes, map[string]attr.Value{"enabled": types.BoolValue(false), "connector_id": types.StringNull(), "deployment_type": types.StringNull()}), stateValue, passportState(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateRaw, err := tc.plan.ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			resp := &planmodifier.ObjectResponse{PlanValue: tc.plan}
			passportPlanModifier{}.PlanModifyObject(ctx, planmodifier.ObjectRequest{ConfigValue: tc.config, PlanValue: tc.plan, StateValue: tc.plan, State: tfsdk.State{Raw: stateRaw}}, resp)
			if !resp.PlanValue.Equal(tc.want) {
				t.Fatalf("Passport plan = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}
