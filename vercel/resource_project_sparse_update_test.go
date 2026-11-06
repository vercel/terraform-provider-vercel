package vercel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectSparseUpdate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		changes map[string]attr.Value
		want    string
	}{
		{name: "rename", changes: map[string]attr.Value{"name": types.StringValue("renamed")}, want: `{"name":"renamed"}`},
		{name: "set command", changes: map[string]attr.Value{"build_command": types.StringValue("npm run build:new")}, want: `{"buildCommand":"npm run build:new"}`},
		{name: "clear command", changes: map[string]attr.Value{"build_command": types.StringNull()}, want: `{"buildCommand":null}`},
		{name: "false is a write", changes: map[string]attr.Value{"git_lfs": types.BoolValue(false)}, want: `{"gitLFS":false}`},
		{name: "disable skew protection", changes: map[string]attr.Value{"skew_protection": types.StringNull()}, want: `{"skewProtectionMaxAge":0}`},
		{name: "resource config child only", changes: map[string]attr.Value{"resource_config.fluid": types.BoolValue(false)}, want: `{"resourceConfig":{"fluid":false}}`},
		{name: "function timeout only", changes: map[string]attr.Value{"resource_config.function_default_timeout": types.Int64Value(120)}, want: `{"resourceConfig":{"functionDefaultTimeout":120}}`},
		{name: "build machine only", changes: map[string]attr.Value{"build_machine_type": types.StringValue("enhanced")}, want: `{"resourceConfig":{"buildMachineType":"enhanced"}}`},
		{name: "Git provider child only", changes: map[string]attr.Value{"git_provider_options.git_commit_status": types.BoolValue(false)}, want: `{"gitProviderOptions":{"gitCommitStatus":false}}`},
		{name: "inverse Git provider child", changes: map[string]attr.Value{"git_provider_options.repository_dispatch_events": types.BoolValue(false)}, want: `{"gitProviderOptions":{"disableRepositoryDispatchEvents":true}}`},
		{name: "unknown settings omitted", changes: map[string]attr.Value{"name": types.StringValue("renamed"), "build_command": types.StringUnknown(), "git_lfs": types.BoolUnknown()}, want: `{"name":"renamed"}`},
		{name: "region order is unchanged", changes: map[string]attr.Value{"name": types.StringValue("renamed"), "resource_config.function_default_regions": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("sfo1"), types.StringValue("iad1")})}, want: `{"name":"renamed"}`},
		{name: "computed region alias is unknown", changes: map[string]attr.Value{"name": types.StringValue("renamed"), "serverless_function_region": types.StringUnknown()}, want: `{"name":"renamed"}`},
		{name: "computed preview alias is unknown", changes: map[string]attr.Value{"name": types.StringValue("renamed"), "preview_comments": types.BoolUnknown()}, want: `{"name":"renamed"}`},
		{name: "computed feedback alias is unknown", changes: map[string]attr.Value{"name": types.StringValue("renamed"), "enable_preview_feedback": types.BoolUnknown()}, want: `{"name":"renamed"}`},
		{name: "explicit feedback update", changes: map[string]attr.Value{"enable_preview_feedback": types.BoolValue(false), "preview_comments": types.BoolUnknown()}, want: `{"enablePreviewFeedback":false}`},
		{name: "explicit deprecated feedback update", changes: map[string]attr.Value{"enable_preview_feedback": types.BoolUnknown(), "preview_comments": types.BoolValue(false)}, want: `{"enablePreviewFeedback":false}`},
		{name: "unchanged uses GET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := sparseProjectAPIResponse()
			var patches []map[string]any
			gets := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/prj_123":
					gets++
				case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123":
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					patches = append(patches, body)
					mergeSparseProjectSettings(project, body)
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				if err := json.NewEncoder(w).Encode(project); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
			ctx := context.Background()
			var schemaResponse resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "prj_123"}, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			// Git provider options are only read when this block is managed.
			gitOptions := types.ObjectValueMust(gitProviderOptionsAttrType.AttrTypes, map[string]attr.Value{
				"require_verified_commits":       types.BoolValue(true),
				"create_deployments":             types.BoolValue(true),
				"repository_dispatch_events":     types.BoolValue(true),
				"git_commit_status":              types.BoolValue(true),
				"consolidated_git_commit_status": types.ObjectNull(consolidatedGitCommitStatusAttrType.AttrTypes),
			})
			if diags := imported.State.SetAttribute(ctx, path.Root("git_provider_options"), gitOptions); diags.HasError() {
				t.Fatal(diags)
			}
			plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: imported.State.Raw}
			for name, value := range tc.changes {
				parts := strings.Split(name, ".")
				p := path.Root(parts[0])
				for _, part := range parts[1:] {
					p = p.AtName(part)
				}
				if diags := plan.SetAttribute(ctx, p, value); diags.HasError() {
					t.Fatal(diags)
				}
			}
			response := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Update(ctx, resource.UpdateRequest{State: imported.State, Plan: plan}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if tc.want == "" {
				if len(patches) != 0 || gets != 2 {
					t.Fatalf("no-op sent %d PATCH and %d GET requests, want 0 PATCH and import/update GETs", len(patches), gets)
				}
				return
			}
			var want map[string]any
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			if len(patches) != 1 || !reflect.DeepEqual(patches[0], want) {
				t.Fatalf("PATCHes = %#v, want only %#v", patches, want)
			}
		})
	}
}

func TestProjectSparseUpdateRetainsPlanUntilRead(t *testing.T) {
	ctx := context.Background()
	project := sparseProjectAPIResponse()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/prj_123":
		case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123":
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(body, map[string]any{"name": "renamed"}) {
				t.Errorf("rename resent unrelated settings: %#v", body)
			}
			// A dashboard edit after refresh must survive the unrelated update.
			project["buildCommand"] = "dashboard build"
			project["resourceConfig"].(map[string]any)["functionDefaultTimeout"] = float64(180)
			mergeSparseProjectSettings(project, body)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
			return
		}
		if err := json.NewEncoder(w).Encode(project); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "prj_123"}, &imported)
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: imported.State.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("name"), types.StringValue("renamed")); diags.HasError() {
		t.Fatal(diags)
	}
	updated := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Update(ctx, resource.UpdateRequest{State: imported.State, Plan: plan}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	for _, p := range []path.Path{path.Root("build_command"), path.Root("resource_config").AtName("function_default_timeout")} {
		var got, want attr.Value
		if diags := updated.State.GetAttribute(ctx, p, &got); diags.HasError() {
			t.Fatal(diags)
		}
		if diags := plan.GetAttribute(ctx, p, &want); diags.HasError() {
			t.Fatal(diags)
		}
		if !got.Equal(want) {
			t.Errorf("state at %s = %s, want planned %s until refresh", p, got, want)
		}
	}
	refreshed := resource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Read(ctx, resource.ReadRequest{State: updated.State}, &refreshed)
	if refreshed.Diagnostics.HasError() {
		t.Fatal(refreshed.Diagnostics)
	}
	var command types.String
	if diags := refreshed.State.GetAttribute(ctx, path.Root("build_command"), &command); diags.HasError() {
		t.Fatal(diags)
	}
	if command.ValueString() != "dashboard build" {
		t.Fatalf("refresh build_command = %s, want dashboard edit", command)
	}
	var timeout types.Int64
	if diags := refreshed.State.GetAttribute(ctx, path.Root("resource_config").AtName("function_default_timeout"), &timeout); diags.HasError() {
		t.Fatal(diags)
	}
	if timeout.ValueInt64() != 180 {
		t.Fatalf("refresh function_default_timeout = %s, want dashboard edit", timeout)
	}
}

func sparseProjectAPIResponse() map[string]any {
	return map[string]any{
		"id": "prj_123", "name": "example", "accountId": "team_123", "nodeVersion": "22.x",
		"buildCommand": "npm run build", "gitLFS": true, "autoAssignCustomDomains": true, "gitForkProtection": true,
		"skewProtectionMaxAge": 86400, "enablePreviewFeedback": true,
		"resourceConfig":     map[string]any{"functionDefaultMemoryType": "standard", "functionDefaultTimeout": 60, "functionDefaultRegions": []string{"iad1", "sfo1"}, "fluid": true, "elasticConcurrencyEnabled": true, "buildMachineType": "standard"},
		"gitProviderOptions": map[string]any{"requireVerifiedCommits": true, "createDeployments": "enabled", "disableRepositoryDispatchEvents": false, "gitCommitStatus": true},
	}
}

func mergeSparseProjectSettings(project, changes map[string]any) {
	for name, value := range changes {
		if name == "resourceConfig" || name == "gitProviderOptions" {
			for key, setting := range value.(map[string]any) {
				project[name].(map[string]any)[key] = setting
			}
		} else {
			project[name] = value
		}
	}
}
