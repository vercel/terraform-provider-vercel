package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectsDataSourceRead(t *testing.T) {
	for _, tc := range []struct {
		name, team string
		config     projectSearchConfig
		empty      bool
	}{
		{name: "default team", team: "team_default"},
		{name: "team override", team: "team_override", config: projectSearchConfig{TeamID: types.StringValue("team_override")}},
		{name: "framework and protection", team: "team_default", config: projectSearchConfig{Framework: types.StringValue("nextjs"), PassportEnabled: types.BoolValue(true), PasswordProtectionEnabled: types.BoolValue(false)}},
		{name: "no matches", team: "team_default", config: projectSearchConfig{Framework: types.StringValue("astro")}, empty: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/v10/projects" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				if req.URL.Query().Get("teamId") != tc.team {
					t.Errorf("teamId = %q, want %q", req.URL.Query().Get("teamId"), tc.team)
				}
				if !req.URL.Query().Has("until") {
					fmt.Fprintf(w, `{"projects":[{"id":"prj_disabled","name":"plain","accountId":%q}],"pagination":{"next":123}}`, tc.team)
				} else {
					fmt.Fprintf(w, `{"projects":[{"id":"prj_enabled","name":"web","accountId":%q,"framework":"nextjs","link":{"type":"github","org":"example","repo":"web"},"passport":{"connectorId":"connector_1","deploymentType":"all"},"ssoProtection":{"deploymentType":"preview"},"trustedIps":{"deploymentType":"all","addresses":[{"value":"192.0.2.1"}]},"protectionBypass":{"do-not-store-this":{"scope":"automation-bypass"}}},{"id":"prj_enabled","name":"web","accountId":%q,"framework":"nextjs","link":{"type":"github","org":"example","repo":"web"},"passport":{"connectorId":"connector_1","deploymentType":"all"},"ssoProtection":{"deploymentType":"preview"},"trustedIps":{"deploymentType":"all"}}],"pagination":{"next":null}}`, tc.team, tc.team)
				}
			}))
			defer server.Close()
			d := &projectsDataSource{client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_default"})}
			ctx := context.Background()
			var schemaResp datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
			config := tfsdk.State{Schema: schemaResp.Schema}
			if diags := config.Set(ctx, projectsDataSourceModel{projectSearchConfig: tc.config}); diags.HasError() {
				t.Fatal(diags)
			}
			response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: config.Raw}}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var result projectsDataSourceModel
			if diags := response.State.Get(ctx, &result); diags.HasError() {
				t.Fatal(diags)
			}
			if result.TeamID.ValueString() != tc.team || calls != 2 {
				t.Fatalf("team = %s, calls = %d", result.TeamID, calls)
			}
			expected := 2
			if tc.name == "framework and protection" {
				expected = 1
			}
			if tc.empty {
				expected = 0
			}
			if result.Projects == nil || len(result.Projects) != expected {
				t.Fatalf("projects = %#v, want %d", result.Projects, expected)
			}
			if strings.Contains(response.State.Raw.String(), "do-not-store-this") {
				t.Fatal("bypass secret was stored")
			}
			if disabled, ok := result.Projects["prj_disabled"]; ok {
				if disabled.Passport.Enabled.ValueBool() || !disabled.Passport.DeploymentType.IsNull() || !disabled.GitRepository.IsNull() || !disabled.Framework.IsNull() {
					t.Fatalf("incorrect disabled summary: %#v", disabled)
				}
			}
			if enabled, ok := result.Projects["prj_enabled"]; ok {
				if !enabled.Passport.Enabled.ValueBool() || enabled.Passport.DeploymentType.ValueString() != "all" || !enabled.VercelAuthentication.Enabled.ValueBool() || !enabled.TrustedIPs.Enabled.ValueBool() {
					t.Fatalf("incorrect enabled summary: %#v", enabled)
				}
			}
		})
	}
}

func TestProjectSearchProtectionFilters(t *testing.T) {
	summary := summarizeProject(client.ProjectResponse{
		Passport:             &client.Passport{DeploymentType: "all"},
		VercelAuthentication: &client.VercelAuthentication{DeploymentType: "none"},
		PasswordProtection:   &client.PasswordProtection{DeploymentType: "preview"},
	})
	for _, tc := range []struct {
		config projectSearchConfig
		want   bool
	}{
		{config: projectSearchConfig{}, want: true},
		{config: projectSearchConfig{PassportEnabled: types.BoolValue(true), PasswordProtectionEnabled: types.BoolValue(true), VercelAuthenticationEnabled: types.BoolValue(false), TrustedIPsEnabled: types.BoolValue(false)}, want: true},
		{config: projectSearchConfig{PassportEnabled: types.BoolValue(false)}},
		{config: projectSearchConfig{PasswordProtectionEnabled: types.BoolValue(false)}},
		{config: projectSearchConfig{VercelAuthenticationEnabled: types.BoolValue(true)}},
		{config: projectSearchConfig{TrustedIPsEnabled: types.BoolValue(true)}},
		{config: projectSearchConfig{Framework: types.StringValue("nextjs")}},
	} {
		if got := tc.config.matches(summary); got != tc.want {
			t.Errorf("matches(%#v) = %t, want %t", tc.config, got, tc.want)
		}
	}
	if !summary.VercelAuthentication.DeploymentType.IsNull() {
		t.Fatal("disabled scope should be null")
	}
}
