package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectImportGitCommentsUpdate(t *testing.T) {
	configured := types.ObjectValueMust(gitCommentsAttrTypes, map[string]attr.Value{
		"on_pull_request": types.BoolValue(true),
		"on_commit":       types.BoolValue(false),
	})
	null := types.ObjectNull(gitCommentsAttrTypes)
	unknown := types.ObjectUnknown(gitCommentsAttrTypes)
	for _, tc := range []struct {
		name     string
		apiValue string
		plan     *types.Object
		wantJSON string
	}{
		{name: "preserve imported settings", apiValue: `{"onPullRequest":false,"onCommit":true}`},
		{name: "preserve API default", apiValue: `null`},
		{name: "omit null from old state", apiValue: `{"onPullRequest":false,"onCommit":true}`, plan: &null},
		{name: "omit unknown", apiValue: `{"onPullRequest":false,"onCommit":true}`, plan: &unknown},
		{name: "explicit settings", apiValue: `{"onPullRequest":false,"onCommit":true}`, plan: &configured, wantJSON: `{"onCommit":false,"onPullRequest":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updates := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				comments := tc.apiValue
				name := "example"
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/prj_123":
				case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123":
					updates++
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						http.Error(w, "invalid JSON", http.StatusBadRequest)
						return
					}
					if got := string(body["gitComments"]); got != tc.wantJSON {
						t.Errorf("gitComments = %q, want %q", got, tc.wantJSON)
					}
					if got := string(body["name"]); got != `"renamed"` {
						t.Errorf("name = %s, want renamed", got)
					}
					if tc.wantJSON != "" {
						comments = tc.wantJSON
					}
					name = "renamed"
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				fmt.Fprintf(w, `{"id":"prj_123","name":%q,"accountId":"team_123","nodeVersion":"22.x","gitComments":%s}`, name, comments)
			}))
			defer server.Close()

			ctx := context.Background()
			r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "prj_123"}, &importResp)
			if importResp.Diagnostics.HasError() {
				t.Fatal(importResp.Diagnostics)
			}
			var imported types.Object
			if diags := importResp.State.GetAttribute(ctx, path.Root("git_comments"), &imported); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.apiValue == "null" {
				if !imported.IsNull() {
					t.Fatalf("imported git_comments = %s, want null", imported)
				}
			} else if imported.IsNull() || !imported.Attributes()["on_pull_request"].Equal(types.BoolValue(false)) || !imported.Attributes()["on_commit"].Equal(types.BoolValue(true)) {
				t.Fatalf("imported git_comments = %s, want live settings", imported)
			}

			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: importResp.State.Raw}
			if diags := plan.SetAttribute(ctx, path.Root("name"), types.StringValue("renamed")); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.plan != nil {
				if diags := plan.SetAttribute(ctx, path.Root("git_comments"), *tc.plan); diags.HasError() {
					t.Fatal(diags)
				}
			}
			updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.Update(ctx, resource.UpdateRequest{State: importResp.State, Plan: plan}, &updateResp)
			if updateResp.Diagnostics.HasError() {
				t.Fatal(updateResp.Diagnostics)
			}
			if updates != 1 {
				t.Fatalf("updates = %d, want 1", updates)
			}
		})
	}
}

func TestProjectGitCommentsLifecycle(t *testing.T) {
	for _, comments := range []string{`null`, `{"onPullRequest":false,"onCommit":true}`} {
		t.Run(comments, func(t *testing.T) {
			name := "example"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				if req.Method == http.MethodPatch {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					if body["gitComments"] != nil {
						comments = string(body["gitComments"])
					}
					if body["name"] != nil {
						_ = json.Unmarshal(body["name"], &name)
					}
				}
				fmt.Fprintf(w, `{"id":"prj_123","name":%q,"accountId":"team_123","nodeVersion":"22.x","autoAssignCustomDomains":true,"gitForkProtection":true,"enableAffectedProjectsDeployments":false,"gitComments":%s}`, name, comments)
			}))
			defer server.Close()
			factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{Provider: New(), client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"})})}
			config := func(projectName, extra string) string {
				return fmt.Sprintf(`resource "vercel_project" "test" {
name = %q
node_version = "22.x"
%s
}`, projectName, extra)
			}
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
				{Config: config("example", "")},
				{Config: config("renamed", "")},
				{Config: config("renamed", "git_comments = { on_pull_request = true, on_commit = false }")},
				{Config: config("renamed", ""), Check: tfresource.ComposeTestCheckFunc(tfresource.TestCheckResourceAttr("vercel_project.test", "git_comments.on_pull_request", "true"), tfresource.TestCheckResourceAttr("vercel_project.test", "git_comments.on_commit", "false"))},
				{Config: config("renamed", "git_comments = {}"), ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
				{Config: config("renamed", "git_comments = { on_pull_request = true }"), ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
				{Config: config("renamed", "")},
			}})
		})
	}
}
