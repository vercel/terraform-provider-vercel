package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestLogDrainSamplingLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		rate     types.Float64
		rules    []client.DrainSamplingRule
		useRules bool
	}{
		{name: "omitted", rate: types.Float64Null()},
		{name: "legacy", rate: types.Float64Value(0.25)},
		{name: "legacy zero", rate: types.Float64Value(0)},
		{name: "empty rules", rate: types.Float64Null(), useRules: true},
		{name: "conditional rule", rate: types.Float64Null(), useRules: true, rules: []client.DrainSamplingRule{{Rate: 0.25, RequestPath: "/api"}}},
		{name: "unconditional rule", rate: types.Float64Null(), useRules: true, rules: []client.DrainSamplingRule{{Rate: 0.25}}},
		{name: "ordered rules", rate: types.Float64Null(), useRules: true, rules: []client.DrainSamplingRule{
			{Rate: 0, Environment: "production", RequestPath: "/health"},
			{Rate: 0.25, Environment: "production"},
			{Rate: 1},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sampling := json.RawMessage(`[]`)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPost {
					var payload struct {
						Sampling json.RawMessage `json:"sampling"`
					}
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Error(err)
						http.Error(w, "invalid request", http.StatusBadRequest)
						return
					}
					if len(payload.Sampling) > 0 {
						sampling = payload.Sampling
					}
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"ld_123","ownerId":"team_123","name":"example","projectIds":["prj_123"],"delivery":{"type":"http","endpoint":"https://example.com/logs","encoding":"ndjson","headers":{},"secret":"generated-secret"},"filterV2":{"version":"v2","filter":{"type":"basic","log":{"sources":["static"]},"deployment":{"environments":["production"]}}},"sampling":%s}`, sampling)
			}))
			defer server.Close()
			apiClient := client.New("test").WithBaseURL(server.URL)
			r := &logDrainResource{client: apiClient}
			var schemaResponse resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			plan, config := logDrainCreatePlanAndConfig(t, schemaResponse.Schema, types.StringNull())
			rules := types.ListNull(drainSamplingRuleAttrType)
			if tc.useRules {
				var diags diag.Diagnostics
				rules, diags = drainSamplingRulesFromAPI(ctx, tc.rules, types.ListValueMust(drainSamplingRuleAttrType, []attr.Value{}))
				if diags.HasError() {
					t.Fatal(diags)
				}
			}
			if diags := plan.SetAttribute(ctx, path.Root("sampling_rate"), tc.rate); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := plan.SetAttribute(ctx, path.Root("sampling_rules"), rules); diags.HasError() {
				t.Fatal(diags)
			}
			created := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &created)
			if created.Diagnostics.HasError() {
				t.Fatal(created.Diagnostics)
			}
			checkState := func(state tfsdk.State) {
				t.Helper()
				var model LogDrain
				if diags := state.Get(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				if !model.SamplingRate.Equal(tc.rate) || !model.SamplingRules.Equal(rules) {
					t.Fatalf("sampling state = (%s, %s), want (%s, %s)", model.SamplingRate, model.SamplingRules, tc.rate, rules)
				}
			}
			checkState(created.State)
			refreshed := resource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Read(ctx, resource.ReadRequest{State: created.State}, &refreshed)
			if refreshed.Diagnostics.HasError() {
				t.Fatal(refreshed.Diagnostics)
			}
			checkState(refreshed.State)

			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "team_123/ld_123"}, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			var importedModel LogDrain
			if diags := imported.State.Get(ctx, &importedModel); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.useRules && (len(tc.rules) > 1 || (len(tc.rules) == 1 && tc.rules[0].RequestPath != "")) && !importedModel.SamplingRules.Equal(rules) {
				t.Fatalf("imported rules = %s, want %s", importedModel.SamplingRules, rules)
			}

			d := &logDrainDataSource{client: apiClient}
			var dsSchema datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &dsSchema)
			configState := tfsdk.State{Schema: dsSchema.Schema}
			if diags := configState.Set(ctx, LogDrainWithoutSecret{
				ID:            types.StringValue("ld_123"),
				TeamID:        types.StringValue("team_123"),
				Endpoint:      types.StringValue("https://example.com/logs"),
				Environments:  types.SetNull(types.StringType),
				Sources:       types.SetNull(types.StringType),
				ProjectIDs:    types.SetNull(types.StringType),
				Headers:       types.MapNull(types.StringType),
				SamplingRules: types.ListNull(drainSamplingRuleAttrType),
			}); diags.HasError() {
				t.Fatal(diags)
			}
			read := datasource.ReadResponse{State: tfsdk.State{Schema: dsSchema.Schema}}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: dsSchema.Schema, Raw: configState.Raw}}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			var data LogDrainWithoutSecret
			if diags := read.State.Get(ctx, &data); diags.HasError() {
				t.Fatal(diags)
			}
			expectedRules := rules
			if !tc.rate.IsNull() {
				var diags diag.Diagnostics
				expectedRules, diags = drainSamplingRulesFromAPI(ctx, []client.DrainSamplingRule{{Rate: tc.rate.ValueFloat64()}}, types.ListNull(drainSamplingRuleAttrType))
				if diags.HasError() {
					t.Fatal(diags)
				}
			} else if len(tc.rules) == 0 {
				expectedRules = types.ListNull(drainSamplingRuleAttrType)
			}
			if !data.SamplingRules.Equal(expectedRules) {
				t.Fatalf("data source rules = %s, want %s", data.SamplingRules, expectedRules)
			}
		})
	}
}
