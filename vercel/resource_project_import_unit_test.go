package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectImportPasswordProtectionUpdate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		disable  bool
		wantJSON string
	}{
		{name: "preserve imported password", wantJSON: `{"deploymentType":"all"}`},
		{name: "replace password", password: "replacement-password", wantJSON: `{"deploymentType":"all","password":"replacement-password"}`},
		{name: "disable protection", disable: true, wantJSON: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updates := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				protection := `{"deploymentType":"all"}`
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
					if got := string(body["passwordProtection"]); got != tc.wantJSON {
						t.Errorf("passwordProtection = %s, want %s", got, tc.wantJSON)
					}
					if got := string(body["name"]); got != `"renamed"` {
						t.Errorf("name = %s, want renamed", got)
					}
					name = "renamed"
					if tc.disable {
						protection = "null"
					}
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				fmt.Fprintf(w, `{"id":"prj_123","name":%q,"accountId":"team_123","nodeVersion":"22.x","passwordProtection":%s}`, name, protection)
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
			var password types.String
			passwordPath := path.Root("password_protection").AtName("password")
			if diags := importResp.State.GetAttribute(ctx, passwordPath, &password); diags.HasError() {
				t.Fatal(diags)
			}
			if !password.Equal(types.StringValue("")) {
				t.Fatalf("imported password = %s, want empty placeholder", password)
			}

			// ignore_changes carries the imported password into the update plan.
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: importResp.State.Raw}
			if diags := plan.SetAttribute(ctx, path.Root("name"), types.StringValue("renamed")); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.disable {
				if diags := plan.SetAttribute(ctx, path.Root("password_protection"), types.ObjectNull(passwordProtectionWithPasswordAttrType.AttrTypes)); diags.HasError() {
					t.Fatal(diags)
				}
			} else if tc.password != "" {
				if diags := plan.SetAttribute(ctx, passwordPath, types.StringValue(tc.password)); diags.HasError() {
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
			var protection types.Object
			if diags := updateResp.State.GetAttribute(ctx, path.Root("password_protection"), &protection); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.disable {
				if !protection.IsNull() {
					t.Fatalf("password protection = %s, want null", protection)
				}
			} else if !protection.Attributes()["password"].Equal(types.StringValue(tc.password)) {
				t.Fatalf("password = %s, want %q", protection.Attributes()["password"], tc.password)
			}
		})
	}
}

func TestProjectImportGitProviderOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v10/projects/prj_123" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
			return
		}
		fmt.Fprint(w, `{"id":"prj_123","name":"example","accountId":"team_123","nodeVersion":"22.x","gitProviderOptions":{"requireVerifiedCommits":true,"createDeployments":"enabled","disableRepositoryDispatchEvents":false,"gitCommitStatus":true,"consolidatedGitCommitStatus":{"enabled":true,"propagateFailures":true}}}`)
	}))
	defer server.Close()

	ctx := context.Background()
	r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "prj_123"}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	for _, tc := range []struct {
		attribute string
		want      types.Bool
	}{
		{attribute: "require_verified_commits", want: types.BoolValue(true)},
		{attribute: "create_deployments", want: types.BoolValue(true)},
		{attribute: "repository_dispatch_events", want: types.BoolValue(true)},
		{attribute: "git_commit_status", want: types.BoolValue(true)},
	} {
		var got types.Bool
		attributePath := path.Root("git_provider_options").AtName(tc.attribute)
		if diags := resp.State.GetAttribute(ctx, attributePath, &got); diags.HasError() {
			t.Fatal(diags)
		}
		if !got.Equal(tc.want) {
			t.Errorf("imported git_provider_options.%s = %s, want %s", tc.attribute, got, tc.want)
		}
	}

	for _, attribute := range []string{"enabled", "propagate_failures"} {
		var got types.Bool
		attributePath := path.Root("git_provider_options").AtName("consolidated_git_commit_status").AtName(attribute)
		if diags := resp.State.GetAttribute(ctx, attributePath, &got); diags.HasError() {
			t.Fatal(diags)
		}
		if !got.Equal(types.BoolValue(true)) {
			t.Errorf("imported git_provider_options.consolidated_git_commit_status.%s = %s, want true", attribute, got)
		}
	}
}

// A project that never configured git_provider_options still gets the live options in state
// on import, so check that Terraform does not then plan a change for an absent configuration.
func TestProjectImportGitProviderOptionsPlansNoChange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v10/projects/prj_123" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
			return
		}
		// gitProviderOptions holds the defaults the API reports for a project that never set them.
		fmt.Fprint(w, `{"id":"prj_123","name":"example","accountId":"team_123","nodeVersion":"22.x","autoAssignCustomDomains":true,"gitForkProtection":true,"gitProviderOptions":{"createDeployments":"disabled","disableRepositoryDispatchEvents":true}}`)
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
	var importedOptions types.Object
	if diags := importResp.State.GetAttribute(ctx, path.Root("git_provider_options"), &importedOptions); diags.HasError() {
		t.Fatal(diags)
	}
	if importedOptions.IsNull() {
		t.Fatal("import left git_provider_options out of state")
	}

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("unexpected schema type %T", schemaResp.Schema.Type().TerraformType(ctx))
	}
	configAttributes := map[string]tftypes.Value{}
	for attribute, attributeType := range objectType.AttributeTypes {
		configAttributes[attribute] = tftypes.NewValue(attributeType, nil)
	}
	configAttributes["name"] = tftypes.NewValue(tftypes.String, "example")
	config, err := tfprotov6.NewDynamicValue(objectType, tftypes.NewValue(objectType, configAttributes))
	if err != nil {
		t.Fatal(err)
	}
	state, err := tfprotov6.NewDynamicValue(objectType, importResp.State.Raw)
	if err != nil {
		t.Fatal(err)
	}

	providerServer, err := providerserver.NewProtocol6WithError(New())()
	if err != nil {
		t.Fatal(err)
	}
	// Terraform keeps the prior state for an optional and computed attribute that the
	// configuration omits, so the proposed new state is the imported state.
	planResp, err := providerServer.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "vercel_project",
		PriorState:       &state,
		ProposedNewState: &state,
		Config:           &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range planResp.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("plan error: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	planned, err := planResp.PlannedState.Unmarshal(objectType)
	if err != nil {
		t.Fatal(err)
	}
	var plannedOptions types.Object
	plannedState := tfsdk.State{Schema: schemaResp.Schema, Raw: planned}
	if diags := plannedState.GetAttribute(ctx, path.Root("git_provider_options"), &plannedOptions); diags.HasError() {
		t.Fatal(diags)
	}
	if !plannedOptions.Equal(importedOptions) {
		t.Errorf("planned git_provider_options = %s, want the imported %s", plannedOptions, importedOptions)
	}
	if !planned.Equal(importResp.State.Raw) {
		t.Error("planning an unchanged configuration after import produced a diff")
	}
}

// Changing another attribute of an imported project whose configuration omits
// git_provider_options leaves the block unknown in the plan, so the update must not send any
// git provider options and overwrite settings the configuration says nothing about.
func TestProjectUpdateOmitsUnknownGitProviderOptions(t *testing.T) {
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
			if options, ok := body["gitProviderOptions"]; ok {
				t.Errorf("update sent gitProviderOptions = %s, want it omitted", options)
			}
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
			return
		}
		fmt.Fprint(w, `{"id":"prj_123","name":"example","accountId":"team_123","nodeVersion":"22.x","gitProviderOptions":{"createDeployments":"disabled","disableRepositoryDispatchEvents":true}}`)
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

	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: importResp.State.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("name"), types.StringValue("renamed")); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := plan.SetAttribute(ctx, path.Root("git_provider_options"), types.ObjectUnknown(gitProviderOptionsAttrType.AttrTypes)); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, resource.UpdateRequest{State: importResp.State, Plan: plan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if updates != 1 {
		t.Fatalf("updates = %d, want 1", updates)
	}
}

func TestProjectImportTeamOwnership(t *testing.T) {
	for _, tc := range []struct {
		name         string
		importID     string
		accountID    string
		providerTeam string
		requestTeam  string
		wantTeam     types.String
	}{
		{name: "bare ID discovers team", importID: "prj_123", accountID: "team_123", wantTeam: types.StringValue("team_123")},
		{name: "qualified ID", importID: "team_123/prj_123", accountID: "team_123", requestTeam: "team_123", wantTeam: types.StringValue("team_123")},
		{name: "provider default", importID: "prj_123", accountID: "team_123", providerTeam: "team_123", requestTeam: "team_123", wantTeam: types.StringValue("team_123")},
		{name: "personal project", importID: "prj_123", accountID: "user_123", wantTeam: types.StringNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/v10/projects/prj_123" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				if got := req.URL.Query().Get("teamId"); got != tc.requestTeam {
					t.Errorf("request teamId = %q, want %q", got, tc.requestTeam)
				}
				fmt.Fprintf(w, `{"id":"prj_123","name":"example","accountId":%q,"nodeVersion":"22.x"}`, tc.accountID)
			}))
			defer server.Close()

			ctx := context.Background()
			r := &projectResource{client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: tc.providerTeam})}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.importID}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var teamID types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("team_id"), &teamID); diags.HasError() {
				t.Fatal(diags)
			}
			if !teamID.Equal(tc.wantTeam) {
				t.Fatalf("imported team_id = %s, want %s", teamID, tc.wantTeam)
			}
		})
	}
}

func TestProjectReadRecoversTeamOwnership(t *testing.T) {
	envRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", req.Method)
			http.Error(w, "read only", http.StatusMethodNotAllowed)
			return
		}
		switch req.URL.Path {
		case "/v10/projects/prj_123":
			if got := req.URL.Query().Get("teamId"); got != "" {
				t.Errorf("initial read teamId = %q, want empty", got)
			}
			fmt.Fprint(w, `{"id":"prj_123","name":"example","accountId":"team_123","nodeVersion":"22.x"}`)
		case "/v8/projects/prj_123/env":
			envRequests++
			if got := req.URL.Query().Get("teamId"); got != "team_123" {
				t.Errorf("environment read teamId = %q, want team_123", got)
			}
			fmt.Fprint(w, `{"envs":[]}`)
		default:
			t.Errorf("unexpected endpoint: %s", req.URL)
			http.NotFound(w, req)
		}
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
	state := importResp.State
	if diags := state.SetAttribute(ctx, path.Root("team_id"), types.StringNull()); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := state.SetAttribute(ctx, path.Root("environment"), types.SetValueMust(envVariableElemType, []attr.Value{})); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var teamID types.String
	if diags := resp.State.GetAttribute(ctx, path.Root("team_id"), &teamID); diags.HasError() {
		t.Fatal(diags)
	}
	if !teamID.Equal(types.StringValue("team_123")) {
		t.Fatalf("refreshed team_id = %s, want team_123", teamID)
	}
	if envRequests != 1 {
		t.Fatalf("environment requests = %d, want 1", envRequests)
	}
}
