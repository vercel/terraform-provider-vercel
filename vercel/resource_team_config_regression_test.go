package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestTeamConfigNullableSettingsPlanning(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["vercel_team_config"].ValueType().(tftypes.Object)
	for _, tc := range []struct {
		name       string
		changeName bool
		prior      any
		configured any
		want       any
	}{
		{name: "null settings no-op"},
		{name: "null settings unrelated update", changeName: true},
		{name: "known settings unrelated update", changeName: true, prior: "off", want: "off"},
		{name: "explicit settings override state", changeName: true, prior: "off", configured: "on", want: "on"},
		{name: "unknown configuration remains unknown", changeName: true, prior: "off", configured: tftypes.UnknownValue, want: tftypes.UnknownValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := teamConfigNullFields(typ)
			config := teamConfigNullFields(typ)
			proposed := teamConfigNullFields(typ)
			for _, fields := range []map[string]tftypes.Value{prior, config, proposed} {
				fields["id"] = tftypes.NewValue(tftypes.String, "team_test")
				fields["name"] = tftypes.NewValue(tftypes.String, "Original")
			}
			if tc.changeName {
				config["name"] = tftypes.NewValue(tftypes.String, "Renamed")
				proposed["name"] = config["name"]
			}
			for _, name := range []string{"description", "enable_preview_feedback", "enable_production_feedback"} {
				prior[name] = tftypes.NewValue(tftypes.String, tc.prior)
				config[name] = tftypes.NewValue(tftypes.String, tc.configured)
				proposed[name] = prior[name]
				if tc.configured != nil {
					proposed[name] = config[name]
				}
			}
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName:         "vercel_team_config",
				Config:           teamConfigDynamicValue(t, typ, config),
				PriorState:       teamConfigDynamicValue(t, typ, prior),
				ProposedNewState: teamConfigDynamicValue(t, typ, proposed),
			})
			if err != nil {
				t.Fatal(err)
			}
			teamConfigCheckProtocolDiagnostics(t, response.Diagnostics)
			planned, err := response.PlannedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]tftypes.Value
			if err := planned.As(&fields); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"description", "enable_preview_feedback", "enable_production_feedback"} {
				want := tftypes.NewValue(tftypes.String, tc.want)
				if !fields[name].Equal(want) {
					t.Errorf("planned %s = %s, want %s", name, fields[name], want)
				}
			}
			if got := fields["default_deployment_protection"]; !got.IsNull() || !got.IsKnown() {
				t.Errorf("omitted null default_deployment_protection = %s, want null", got)
			}
			if !tc.changeName && !planned.Equal(tftypes.NewValue(typ, prior)) {
				t.Errorf("omitted settings caused a no-op plan to change: %s", planned)
			}
		})
	}
}

func TestTeamConfigDescriptionValidation(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["vercel_team_config"].ValueType().(tftypes.Object)
	for _, tc := range []struct {
		name    string
		value   any
		invalid bool
	}{
		{name: "omitted"},
		{name: "unknown", value: tftypes.UnknownValue},
		{name: "empty", value: ""},
		{name: "140 characters", value: strings.Repeat("a", 140)},
		{name: "141 characters", value: strings.Repeat("a", 141), invalid: true},
		{name: "140 Unicode characters", value: strings.Repeat("é", 140)},
		{name: "141 Unicode characters", value: strings.Repeat("é", 141), invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := teamConfigNullFields(typ)
			fields["id"] = tftypes.NewValue(tftypes.String, "team_test")
			fields["description"] = tftypes.NewValue(tftypes.String, tc.value)
			response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "vercel_team_config", Config: teamConfigDynamicValue(t, typ, fields),
			})
			if err != nil {
				t.Fatal(err)
			}
			invalid := false
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					invalid = true
				}
			}
			if invalid != tc.invalid {
				t.Fatalf("validation error = %v, want %v: %v", invalid, tc.invalid, response.Diagnostics)
			}
		})
	}
}

func TestTeamConfigDescriptionPatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value types.String
		want  string
	}{
		{name: "omitted", value: types.StringNull()},
		{name: "unknown", value: types.StringUnknown()},
		{name: "clear", value: types.StringValue(""), want: `""`},
		{name: "replace", value: types.StringValue("Updated"), want: `"Updated"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, diags := convertResponseToTeamConfig(context.Background(), client.Team{ID: "team_test"}, types.MapNull(types.StringType))
			if diags.HasError() {
				t.Fatal(diags)
			}
			config.Description = tc.value
			request, diags := config.toUpdateTeamRequest(context.Background(), "", types.StringNull())
			if diags.HasError() {
				t.Fatal(diags)
			}
			body, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			if got := string(fields["description"]); got != tc.want {
				t.Fatalf("PATCH description = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestTeamConfigClearDescriptionUpdate(t *testing.T) {
	ctx := context.Background()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.Method != "PATCH" || req.URL.Path != "/v2/teams/team_test" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["description"]) != `""` {
			t.Errorf("clear description request = %s, want empty string", body["description"])
		}
		fmt.Fprint(w, `{"id":"team_test","name":"Example","slug":"example","description":""}`)
	}))
	defer server.Close()
	r := &teamConfigResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	description := "Original description"
	model, diags := convertResponseToTeamConfig(ctx, client.Team{ID: "team_test", Name: "Example", Slug: "example", Description: &description}, types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatal(diags)
	}
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: state.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("description"), types.StringValue("")); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var got TeamConfig
	if diags := response.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if !got.Description.Equal(types.StringValue("")) || requests != 1 {
		t.Fatalf("updated description = %s, requests = %d", got.Description, requests)
	}
}

func teamConfigNullFields(typ tftypes.Object) map[string]tftypes.Value {
	fields := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for name, fieldType := range typ.AttributeTypes {
		fields[name] = tftypes.NewValue(fieldType, nil)
	}
	return fields
}

func teamConfigDynamicValue(t *testing.T, typ tftypes.Type, fields map[string]tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	value, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, fields))
	if err != nil {
		t.Fatal(err)
	}
	return &value
}

func teamConfigCheckProtocolDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	if t.Failed() {
		t.FailNow()
	}
}
