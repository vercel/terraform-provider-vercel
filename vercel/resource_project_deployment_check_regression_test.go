package vercel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectDeploymentCheckImportPreservesProjectName(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v2/projects/my-project/checks/check_123" || req.URL.Query().Get("teamId") != "team_123" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(client.ProjectDeploymentCheck{ID: "check_123", ProjectID: "prj_123", OwnerID: "team_123", Name: "Check", Requires: "deployment-url", Targets: []string{"production"}, Source: client.ProjectDeploymentCheckSource{Kind: "webhook"}})
	}))
	defer server.Close()
	r := &projectDeploymentCheckResource{client: client.New("test-token").WithBaseURL(server.URL)}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "team_123/my-project/check_123"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", resp.Diagnostics)
	}
	var imported ProjectDeploymentCheck
	if diags := resp.State.Get(ctx, &imported); diags.HasError() {
		t.Fatalf("state diagnostics: %v", diags)
	}
	if imported.ProjectID.ValueString() != "my-project" || imported.ID.ValueString() != "check_123" || imported.TeamID.ValueString() != "team_123" {
		t.Fatalf("unexpected imported identifiers: %+v", imported)
	}
}

func deploymentCheckRegressionPlan(t *testing.T, requires string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	(&projectDeploymentCheckResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)
	model, diags := projectDeploymentCheckFromClient(ctx, client.ProjectDeploymentCheck{ID: "check_123", Name: "Check", Requires: requires, Targets: []string{"production"}, Source: client.ProjectDeploymentCheckSource{Kind: "webhook", WebhookID: "hook_123"}}, types.StringValue("my-project"), types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatalf("model diagnostics: %v", diags)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatalf("plan diagnostics: %v", diags)
	}
	return plan
}

func TestProjectDeploymentCheckRequiresReplacement(t *testing.T) {
	for _, tc := range []struct {
		before, after string
		replace       bool
	}{
		{"deployment-url", "none", true},
		{"none", "none", false},
		{"none", "deployment-url", false},
	} {
		t.Run(tc.before+"_to_"+tc.after, func(t *testing.T) {
			before := deploymentCheckRegressionPlan(t, tc.before)
			after := deploymentCheckRegressionPlan(t, tc.after)
			req := resource.ModifyPlanRequest{State: tfsdk.State(before), Plan: after, Config: tfsdk.Config(after)}
			resp := &resource.ModifyPlanResponse{Plan: after}
			(&projectDeploymentCheckResource{}).ModifyPlan(context.Background(), req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("plan diagnostics: %v", resp.Diagnostics)
			}
			if tc.replace {
				if len(resp.RequiresReplace) != 1 || !resp.RequiresReplace[0].Equal(path.Root("requires")) {
					t.Fatalf("replacement paths = %v, want requires", resp.RequiresReplace)
				}
			} else if len(resp.RequiresReplace) != 0 {
				t.Fatalf("unexpected replacement paths: %v", resp.RequiresReplace)
			}
		})
	}
}

func TestProjectDeploymentCheckSourcePlanModifiers(t *testing.T) {
	ctx := context.Background()
	plan := deploymentCheckRegressionPlan(t, "deployment-url")
	state := tfsdk.State(plan)
	source := plan.Schema.(schema.Schema).Attributes["source"].(schema.SingleNestedAttribute)
	for name, attr := range source.Attributes {
		attribute := attr.(schema.StringAttribute)
		if !attribute.Computed {
			continue
		}
		t.Run("preserve_"+name, func(t *testing.T) {
			req := planmodifier.StringRequest{Path: path.Root("source").AtName(name), State: state, Plan: plan, StateValue: types.StringValue("known"), PlanValue: types.StringUnknown(), ConfigValue: types.StringNull()}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			for _, modifier := range attribute.PlanModifiers {
				req.PlanValue = resp.PlanValue
				modifier.PlanModifyString(ctx, req, resp)
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("modifier diagnostics: %v", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(req.StateValue) || resp.RequiresReplace {
				t.Fatalf("plan value = %v, replacement = %t; want known value without replacement", resp.PlanValue, resp.RequiresReplace)
			}
		})
	}
	for _, name := range []string{"kind", "external_check_name", "provider", "webhook_id", "external_resource_id"} {
		t.Run("replace_"+name, func(t *testing.T) {
			req := planmodifier.StringRequest{Path: path.Root("source").AtName(name), State: state, Plan: plan, StateValue: types.StringValue("before"), PlanValue: types.StringValue("after"), ConfigValue: types.StringValue("after")}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			for _, modifier := range source.Attributes[name].(schema.StringAttribute).PlanModifiers {
				req.PlanValue = resp.PlanValue
				modifier.PlanModifyString(ctx, req, resp)
			}
			if resp.Diagnostics.HasError() || !resp.RequiresReplace {
				t.Fatalf("expected source change replacement; response = %+v", resp)
			}
		})
	}
	var model ProjectDeploymentCheck
	if diags := state.Get(ctx, &model); diags.HasError() {
		t.Fatalf("state diagnostics: %v", diags)
	}
	// An omitted computed child must not make the parent require replacement.
	attrs := model.Source.Attributes()
	attrs["webhook_id"] = types.StringUnknown()
	plannedSource := types.ObjectValueMust(projectDeploymentCheckSourceAttrTypes, attrs)
	req := planmodifier.ObjectRequest{State: state, Plan: plan, StateValue: model.Source, PlanValue: plannedSource, ConfigValue: model.Source}
	resp := &planmodifier.ObjectResponse{PlanValue: plannedSource}
	for _, modifier := range source.PlanModifiers {
		req.PlanValue = resp.PlanValue
		modifier.PlanModifyObject(ctx, req, resp)
	}
	if resp.Diagnostics.HasError() || resp.RequiresReplace {
		t.Fatalf("parent source must not force replacement for a computed child: %+v", resp)
	}
}

func TestProjectDeploymentCheckValidateConfig(t *testing.T) {
	ctx := context.Background()
	gitSource := ProjectDeploymentCheckSource{Kind: types.StringValue("git-provider"), Provider: types.StringValue("github"), ExternalCheckName: types.StringValue("e2e")}
	for _, tc := range []struct {
		name           string
		source         ProjectDeploymentCheckSource
		omitSource     bool
		unknownSource  bool
		rerequestable  types.Bool
		targets        []string
		unknownTargets bool
		wantError      string
	}{
		{name: "git provider", source: gitSource},
		{name: "git provider reruns", source: gitSource, rerequestable: types.BoolValue(true), wantError: "Unsupported rerun setting"},
		{name: "webhook", source: ProjectDeploymentCheckSource{Kind: types.StringValue("webhook"), WebhookID: types.StringValue("hook_123")}, rerequestable: types.BoolValue(true)},
		{name: "integration", source: ProjectDeploymentCheckSource{Kind: types.StringValue("integration"), ExternalResourceID: types.StringValue("resource_123")}},
		{name: "wrong webhook field", source: ProjectDeploymentCheckSource{Kind: types.StringValue("integration"), WebhookID: types.StringValue("hook_123")}, wantError: "Invalid source attribute"},
		{name: "wrong integration field", source: ProjectDeploymentCheckSource{Kind: types.StringValue("webhook"), ExternalResourceID: types.StringValue("resource_123")}, wantError: "Invalid source attribute"},
		{name: "wrong git fields", source: ProjectDeploymentCheckSource{Kind: types.StringValue("webhook"), Provider: types.StringValue("github"), ExternalCheckName: types.StringValue("e2e")}, wantError: "Invalid source attribute"},
		{name: "all plus production", source: gitSource, targets: []string{"all", "production"}, wantError: "Invalid deployment targets"},
		{name: "all alone", source: gitSource, targets: []string{"all"}},
		{name: "unknown kind", source: ProjectDeploymentCheckSource{Kind: types.StringUnknown(), Provider: types.StringUnknown(), WebhookID: types.StringUnknown()}, rerequestable: types.BoolValue(true)},
		{name: "unknown git values", source: ProjectDeploymentCheckSource{Kind: types.StringValue("git-provider"), Provider: types.StringUnknown(), ExternalCheckName: types.StringUnknown()}, rerequestable: types.BoolUnknown()},
		{name: "unknown source", unknownSource: true},
		{name: "unknown targets", source: gitSource, unknownTargets: true},
		{name: "imported source omitted", omitSource: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := deploymentCheckRegressionPlan(t, "deployment-url")
			var model ProjectDeploymentCheck
			if diags := plan.Get(ctx, &model); diags.HasError() {
				t.Fatalf("decode model: %v", diags)
			}
			model.IsRerequestable = tc.rerequestable
			source, diags := types.ObjectValueFrom(ctx, projectDeploymentCheckSourceAttrTypes, tc.source)
			if diags.HasError() {
				t.Fatalf("source diagnostics: %v", diags)
			}
			model.Source = source
			if tc.omitSource {
				model.Source = types.ObjectNull(projectDeploymentCheckSourceAttrTypes)
			}
			if tc.unknownSource {
				model.Source = types.ObjectUnknown(projectDeploymentCheckSourceAttrTypes)
			}
			if tc.unknownTargets {
				model.Targets = types.SetValueMust(types.StringType, []attr.Value{types.StringUnknown()})
			}
			if tc.targets != nil {
				model.Targets, diags = types.SetValueFrom(ctx, types.StringType, tc.targets)
				if diags.HasError() {
					t.Fatalf("target diagnostics: %v", diags)
				}
			}
			if diags := plan.Set(ctx, model); diags.HasError() {
				t.Fatalf("config diagnostics: %v", diags)
			}
			resp := &resource.ValidateConfigResponse{}
			(&projectDeploymentCheckResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config(plan)}, resp)
			if tc.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
				}
				return
			}
			for _, diagnostic := range resp.Diagnostics {
				if diagnostic.Summary() == tc.wantError {
					return
				}
			}
			t.Fatalf("expected %q diagnostic, got %v", tc.wantError, resp.Diagnostics)
		})
	}
}

func TestProjectDeploymentCheckSourceStringValidation(t *testing.T) {
	ctx := context.Background()
	plan := deploymentCheckRegressionPlan(t, "deployment-url")
	source := plan.Schema.(schema.Schema).Attributes["source"].(schema.SingleNestedAttribute)
	for _, name := range []string{"external_check_name", "provider", "webhook_id", "external_resource_id"} {
		for _, value := range []string{"", " ", " padded "} {
			t.Run(name+"/"+value, func(t *testing.T) {
				req := validator.StringRequest{Path: path.Root("source").AtName(name), ConfigValue: types.StringValue(value)}
				resp := &validator.StringResponse{}
				for _, rule := range source.Attributes[name].(schema.StringAttribute).Validators {
					rule.ValidateString(ctx, req, resp)
				}
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected %s to reject %q", name, value)
				}
			})
		}
	}
}
