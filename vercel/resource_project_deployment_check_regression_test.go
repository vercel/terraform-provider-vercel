package vercel_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

func importedDeploymentCheck(t *testing.T) (resource.Resource, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v2/projects/my-project/checks/check_123" || req.URL.Query().Get("teamId") != "team_123" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(client.ProjectDeploymentCheck{ID: "check_123", ProjectID: "prj_123", OwnerID: "team_123", Name: "Check", Requires: "deployment-url", Targets: []string{"production"}, Source: client.ProjectDeploymentCheckSource{Kind: "webhook", WebhookID: "hook_123"}})
	}))
	t.Cleanup(server.Close)
	for _, factory := range vercel.New().Resources(ctx) {
		r := factory()
		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "vercel"}, &metadata)
		if metadata.TypeName != "vercel_project_deployment_check" {
			continue
		}
		var configured resource.ConfigureResponse
		r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: client.New("test-token").WithBaseURL(server.URL)}, &configured)
		if configured.Diagnostics.HasError() {
			t.Fatalf("configure diagnostics: %v", configured.Diagnostics)
		}
		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "team_123/my-project/check_123"}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("import diagnostics: %v", resp.Diagnostics)
		}
		return r, resp.State
	}
	t.Fatal("project deployment check resource not registered")
	return nil, tfsdk.State{}
}

func TestProjectDeploymentCheckImportPreservesProjectName(t *testing.T) {
	_, state := importedDeploymentCheck(t)
	for name, want := range map[string]string{"project_id": "my-project", "id": "check_123", "team_id": "team_123"} {
		var got string
		if diags := state.GetAttribute(context.Background(), path.Root(name), &got); diags.HasError() {
			t.Fatalf("state diagnostics: %v", diags)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
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
			ctx := context.Background()
			r, state := importedDeploymentCheck(t)
			if diags := state.SetAttribute(ctx, path.Root("requires"), tc.before); diags.HasError() {
				t.Fatalf("state diagnostics: %v", diags)
			}
			plan := tfsdk.Plan(state)
			if diags := plan.SetAttribute(ctx, path.Root("requires"), tc.after); diags.HasError() {
				t.Fatalf("plan diagnostics: %v", diags)
			}
			req := resource.ModifyPlanRequest{State: state, Plan: plan, Config: tfsdk.Config(plan)}
			resp := resource.ModifyPlanResponse{Plan: plan}
			r.(resource.ResourceWithModifyPlan).ModifyPlan(ctx, req, &resp)
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
