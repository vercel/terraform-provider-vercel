package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestFirewallImportStateLifecycle(t *testing.T) {
	apiDefaults := defaultCRSMap()
	for _, category := range []string{"rce", "gen", "xss", "sqli"} {
		apiDefaults[category] = client.CoreRuleSet{Active: true, Action: "log"}
	}
	full := client.FirewallConfig{
		Enabled: false,
		ManagedRulesets: map[string]client.ManagedRule{
			"owasp": {Active: true}, "bot_protection": {Active: true, Action: "challenge"}, "ai_bots": {Active: false, Action: "deny"},
		},
		CRS: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}, "xss": {Active: true, Action: "log"}},
		Rules: []client.FirewallRule{
			{ID: "rule_redirect", Name: "redirect", Active: true, Description: "keep description", Action: client.Action{Mitigate: client.Mitigate{Action: "redirect", Redirect: &client.Redirect{Location: "https://example.com", Permanent: true}, ActionDuration: "1h"}}, ConditionGroup: []client.ConditionGroup{{Conditions: []client.Condition{
				{Type: "query", Op: "eq", Key: "campaign", Value: "spring", Neg: true},
				{Type: "path", Op: "inc", Value: []any{"/a", "/b"}},
				{Type: "header", Op: "ex", Key: "x-test", Value: ""},
			}}}},
			{ID: "rule_rate", Name: "rate", Active: false, Action: client.Action{Mitigate: client.Mitigate{Action: "rate_limit", RateLimit: &client.RateLimit{Algo: "fixed_window", Window: 60, Limit: 100, Keys: []string{"ip"}, Action: "deny"}}}, ConditionGroup: []client.ConditionGroup{{Conditions: []client.Condition{
				{Type: "query", Op: "eq", Key: "empty", Value: ""},
				{Type: "query", Op: "ninc", Key: "campaign", Value: []any{"summer", "winter"}},
				{Type: "path", Op: "eq", Value: "/api"},
			}}}},
		},
		IPRules: []client.IPRule{{ID: "ip_one", Hostname: "*", IP: "192.0.2.1", Action: "deny"}, {ID: "ip_two", Hostname: "example.com", IP: "192.0.2.2", Action: "log", Notes: "keep notes"}},
	}
	for _, tc := range []struct {
		name      string
		config    client.FirewallConfig
		wantOWASP bool
	}{
		{name: "full", config: full, wantOWASP: true},
		{name: "partial CRS", config: client.FirewallConfig{Enabled: true, ManagedRulesets: map[string]client.ManagedRule{"owasp": {Active: true}}, CRS: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}}}, wantOWASP: true},
		{name: "CRS only", config: client.FirewallConfig{Enabled: true, CRS: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}, "xss": {Active: true, Action: "log"}}}},
		{name: "disabled OWASP with active category", config: client.FirewallConfig{Enabled: true, ManagedRulesets: map[string]client.ManagedRule{"owasp": {Active: false}}, CRS: map[string]client.CoreRuleSet{"xss": {Active: true, Action: "log"}}}},
		{name: "OWASP marker without CRS", config: client.FirewallConfig{Enabled: true, ManagedRulesets: map[string]client.ManagedRule{"owasp": {Active: true}}}, wantOWASP: true},
		{name: "unknown managed rule", config: client.FirewallConfig{Enabled: true, ManagedRulesets: map[string]client.ManagedRule{"future": {Active: true, Action: "deny"}}}},
		{
			name:      "API default CRS with active marker",
			config:    client.FirewallConfig{Enabled: true, CRS: apiDefaults, ManagedRulesets: map[string]client.ManagedRule{"owasp": {Active: true}}},
			wantOWASP: true,
		},
		{
			name:   "API default CRS without marker",
			config: client.FirewallConfig{Enabled: true, CRS: apiDefaults},
		},
		{
			name:   "API default CRS with inactive marker",
			config: client.FirewallConfig{Enabled: true, CRS: apiDefaults, ManagedRulesets: map[string]client.ManagedRule{"owasp": {Active: false}}},
		},
	} {
		for _, importID := range []string{"team_123/prj_123", "prj_123"} {
			t.Run(tc.name+"/"+importID, func(t *testing.T) {
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					if r.Method != http.MethodGet || r.URL.Path != "/v1/security/firewall/config/active" || r.URL.Query().Get("teamId") != "team_123" || r.URL.Query().Get("projectId") != "prj_123" {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					_ = json.NewEncoder(w).Encode(tc.config)
				}))
				defer server.Close()
				team := "team_123"
				if importID != "prj_123" {
					team = "team_ignored"
				}
				r, imported := firewallImportResponse(t, server.URL, team)
				r.ImportState(context.Background(), resource.ImportStateRequest{ID: importID}, imported)
				if imported.Diagnostics.HasError() {
					t.Fatalf("import: %v", imported.Diagnostics)
				}
				var model FirewallConfig
				if diags := imported.State.Get(context.Background(), &model); diags.HasError() {
					t.Fatalf("decode imported state: %v", diags)
				}
				if model.ID.ValueString() != "team_123/prj_123" || model.TeamID.ValueString() != "team_123" || model.ProjectID.ValueString() != "prj_123" || model.Enabled.IsNull() || model.Enabled.ValueBool() != tc.config.Enabled {
					t.Fatalf("incorrect imported identifiers or enabled: %+v", model)
				}
				hasOWASP := model.ManagedRulesets != nil && model.ManagedRulesets.OWASP != nil
				if hasOWASP != tc.wantOWASP {
					t.Fatalf("OWASP present = %t, want %t", hasOWASP, tc.wantOWASP)
				}
				roundTrip, err := model.toClient()
				if err != nil {
					t.Fatal(err)
				}
				if roundTrip.ManagedRulesets["owasp"].Active != tc.wantOWASP {
					t.Fatalf("round trip changed OWASP activation: %+v", roundTrip.ManagedRulesets)
				}
				if tc.wantOWASP {
					for category, want := range tc.config.CRS {
						got, ok := roundTrip.CRS[category]
						// Omitted categories are disabled with action log on a full update.
						if !ok {
							got = client.CoreRuleSet{Active: false, Action: "log"}
						}
						if got != want {
							t.Fatalf("round trip changed CRS %s: got %+v, want %+v", category, got, want)
						}
					}
				} else if len(roundTrip.CRS) != 0 {
					t.Fatalf("disabled OWASP emitted CRS: %+v", roundTrip.CRS)
				}
				if tc.name == "full" {
					if !model.IPRules.Rules[0].Notes.IsNull() || !model.Rules.Rules[1].Description.IsNull() || !model.Rules.Rules[1].Action.ActionDuration.IsNull() {
						t.Fatal("empty optional strings must import as null")
					}
					got, err := model.toClient()
					if err != nil {
						t.Fatal(err)
					}
					wantJSON, _ := json.Marshal(full)
					gotJSON, _ := json.Marshal(got)
					if string(gotJSON) != string(wantJSON) {
						t.Fatalf("import changed API semantics:\ngot  %s\nwant %s", gotJSON, wantJSON)
					}
				}
				state := imported.State
				for refresh := 0; refresh < 2; refresh++ {
					read := &resource.ReadResponse{State: state}
					r.Read(context.Background(), resource.ReadRequest{State: state}, read)
					if read.Diagnostics.HasError() {
						t.Fatalf("refresh %d: %v", refresh, read.Diagnostics)
					}
					if !read.State.Raw.Equal(imported.State.Raw) {
						t.Fatalf("refresh %d changed imported state", refresh)
					}
					state = read.State
				}
				if requests != 3 {
					t.Fatalf("requests = %d, want import and two refreshes", requests)
				}
			})
		}
	}
}

func TestFirewallImportStateAPIErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, `{"error":{"code":"test_error","message":"test error"}}`)
			}))
			defer server.Close()
			r, resp := firewallImportResponse(t, server.URL, "team_123")
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: "prj_123"}, resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected import error diagnostic")
			}
			if !resp.State.Raw.IsNull() {
				t.Fatalf("failed import populated state: %s", resp.State.Raw)
			}
		})
	}
}

func firewallImportResponse(t *testing.T, baseURL, teamID string) (*firewallConfigResource, *resource.ImportStateResponse) {
	t.Helper()
	r := &firewallConfigResource{client: client.New("token").WithBaseURL(baseURL).WithTeam(client.Team{ID: teamID})}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	return r, &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
}
