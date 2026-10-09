package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

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
