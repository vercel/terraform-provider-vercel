package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectTracingImportTeamOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, importID, providerTeam, owner, wantTeam string
		wantLookups                                   int
	}{
		{"bare team project", "prj_123", "", "team_owner", "team_owner", 1},
		{"personal project", "prj_123", "", "user_owner", "", 1},
		{"explicit team", "team_explicit/prj_123", "team_default", "", "team_explicit", 0},
		{"provider team", "prj_123", "team_default", "", "team_default", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lookups, settingsReads := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.URL.Path == "/v10/projects/prj_123" {
					lookups++
					if req.URL.Query().Get("teamId") != "" {
						t.Error("ownership lookup should be unscoped")
					}
					_, _ = fmt.Fprintf(w, `{"id":"prj_123","accountId":%q}`, tc.owner)
					return
				}
				settingsReads++
				if req.URL.Path != "/v1/drains/tracing/config" || req.URL.Query().Get("teamId") != tc.wantTeam {
					t.Errorf("unexpected settings request: %s", req.URL)
				}
				_, _ = fmt.Fprint(w, `{"enabled":true,"sampling":[]}`)
			}))
			t.Cleanup(server.Close)
			r := &projectTracingResource{client: client.New("token").WithBaseURL(server.URL).WithTeam(client.Team{ID: tc.providerTeam})}
			ctx := context.Background()
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.importID}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var team types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("team_id"), &team); diags.HasError() {
				t.Fatal(diags)
			}
			if !team.Equal(toTeamID(tc.wantTeam)) {
				t.Fatalf("team = %s, want %s", team, toTeamID(tc.wantTeam))
			}
			read := resource.ReadResponse{State: resp.State}
			r.Read(ctx, resource.ReadRequest{State: resp.State}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			if !resp.State.Raw.Equal(read.State.Raw) {
				t.Fatal("refresh changed imported state")
			}
			if lookups != tc.wantLookups || settingsReads != 2 {
				t.Fatalf("lookups=%d reads=%d", lookups, settingsReads)
			}
		})
	}
}

func TestProjectChildImportsRejectOwnershipLookupFailure(t *testing.T) {
	c := client.New("token")
	for _, tc := range []struct {
		name, importID string
		r              resource.ResourceWithImportState
	}{
		{"bulk redirects", "prj_123", &bulkRedirectsResource{client: c}},
		{"custom environment", "prj_123/staging", &customEnvironmentResource{client: c}},
		{"deployment protection exception", "prj_123/example.com", &deploymentProtectionExceptionResource{client: c}},
		{"flag definition", "prj_123/flag", &featureFlagDefinitionResource{client: c}},
		{"flag config", "prj_123/flag", &featureFlagConfigResource{client: c}},
		{"flag segment", "prj_123/segment", &featureFlagSegmentResource{client: c}},
		{"flag SDK key", "prj_123/key", &featureFlagSDKKeyResource{client: c}},
		{"firewall config", "prj_123", &firewallConfigResource{client: c}},
		{"protection bypass", "prj_123/secret", &projectProtectionBypassResource{client: c}},
		{"environment variable", "prj_123/env", &projectEnvironmentVariableResource{client: c}},
		{"route", "prj_123/route", &projectRouteResource{client: c}},
		{"rolling release", "prj_123", &projectRollingReleaseResource{client: c}},
		{"domain", "prj_123/example.com", &projectDomainResource{client: c}},
		{"tracing", "prj_123", &projectTracingResource{client: c}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests++
				if req.URL.Path != "/v10/projects/prj_123" {
					t.Errorf("unexpected request: %s", req.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = fmt.Fprint(w, `{"error":{"code":"forbidden","message":"Forbidden"}}`)
			}))
			t.Cleanup(server.Close)
			c.WithBaseURL(server.URL)
			ctx := context.Background()
			var schema resource.SchemaResponse
			tc.r.Schema(ctx, resource.SchemaRequest{}, &schema)
			resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema}}
			tc.r.ImportState(ctx, resource.ImportStateRequest{ID: tc.importID}, &resp)
			if !resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() || requests != 1 {
				t.Fatalf("failed lookup must stop import: requests=%d, diagnostics=%v, state=%v", requests, resp.Diagnostics, resp.State.Raw)
			}
		})
	}
}

func TestProjectImportRejectsMissingOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"prj_123"}`)
	}))
	t.Cleanup(server.Close)
	resp := &resource.ImportStateResponse{}
	_, ok := importProjectTeam(context.Background(), client.New("token").WithBaseURL(server.URL), "prj_123", "", resp)
	if ok || !resp.Diagnostics.HasError() {
		t.Fatalf("missing owner must fail: ok=%v diagnostics=%v", ok, resp.Diagnostics)
	}
}

func TestSplitBypassIDRejectsEmptyComponents(t *testing.T) {
	for _, id := range []string{"/prj#example.com#1.2.3.4", "team/#example.com#1.2.3.4", "team/prj##1.2.3.4", "team/prj#example.com#", "team/prj#"} {
		if _, _, _, _, ok := splitBypassID(id); ok {
			t.Errorf("accepted malformed ID %q", id)
		}
	}
}
