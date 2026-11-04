package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	vercelclient "github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProtectionInventoryPagesAndAccessTypes(t *testing.T) {
	pages := 0
	c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" || req.URL.Query().Get("teamId") != "team_123" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
		}
		switch req.URL.Path {
		case "/v9/projects/prj_123":
			fmt.Fprint(w, `{"id":"prj_123","accountId":"team_123","protectionBypass":{
				"automation-secret":{"scope":"automation-bypass","createdAt":1000,"createdBy":"user_1","note":"CI"},
				"integration-secret":{"scope":"integration-automation-bypass","integrationId":"int_1","configurationId":"icfg_1"}
			},"optionsAllowlist":{"paths":[{"value":"/api"}]},"trustedSources":{
				"projects":{"prj_other":{}},
				"oidcProviders":{"https://issuer.example":[{"claims":{"aud":["audience"],"sub":["subject"]},"to":{"slugs":["preview"]}}]}
			}}`)
		case "/v4/aliases":
			if req.URL.Query().Get("projectId") != "prj_123" || req.URL.Query().Get("limit") != "100" {
				t.Errorf("missing project filter/limit: %s", req.URL)
			}
			pages++
			if pages == 1 {
				if req.URL.Query().Has("until") {
					t.Error("unexpected initial cursor")
				}
				fmt.Fprint(w, `{"aliases":[{"uid":"alias_1","alias":"preview.example","projectId":"prj_123","protectionBypass":{
					"*":{"scope":"alias-protection-override"},
					"link-secret":{"scope":"shareable-link","createdAt":1000,"expires":2000000000},
					"user_2":{"scope":"user","access":"granted","lastUpdatedAt":2000,"lastUpdatedBy":"user_1"},
					"user_3":{"scope":"user","access":"requested"},
					"invite@example.com":{"scope":"email_invite"}
				}}],"pagination":{"next":100}}`)
			} else {
				if req.URL.Query().Get("until") != "100" {
					t.Error("missing next cursor")
				}
				fmt.Fprint(w, `{"aliases":[{"uid":"alias_2","alias":"old.example","projectId":"prj_123","protectionBypass":{
					"expired-secret":{"scope":"shareable-link","expires":1}
				}}],"pagination":{"next":null}}`)
			}
		default:
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
	})
	got, err := c.GetProjectProtectionInventory(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 || len(got.Entries) != 13 || len(got.IncompleteReasons) != 2 {
		t.Fatalf("pages=%d entries=%d reasons=%v", pages, len(got.Entries), got.IncompleteReasons)
	}
	counts := map[string]int{}
	for _, entry := range got.Entries {
		counts[entry.Scope]++
		if !strings.HasPrefix(entry.ApprovalKey, "v1:") || len(entry.ApprovalKey) != 67 {
			t.Errorf("invalid approval key: %q", entry.ApprovalKey)
		}
		if entry.Scope == "automation-bypass" && (entry.IsEnvVar == nil || !*entry.IsEnvVar) {
			t.Error("legacy default was not normalized")
		}
		if entry.Alias != nil && *entry.Alias == "old.example" && entry.Status != "expired" {
			t.Error("expired shareable link was not retained as expired")
		}
		if entry.Subject != nil && *entry.Subject == "user_2" && (entry.Status != "granted" || entry.LastUpdatedAt == nil || *entry.LastUpdatedAt != 2000) {
			t.Error("user grant metadata was lost")
		}
	}
	for _, scope := range []string{"automation-bypass", "integration-automation-bypass", "alias-protection-override", "shareable-link", "user", "email_invite", "options-allowlist", "trusted-project", "trusted-oidc", "vercel-ci-same-repository"} {
		if counts[scope] == 0 {
			t.Errorf("missing scope %s", scope)
		}
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"automation-secret", "integration-secret", "link-secret", "expired-secret"} {
		if strings.Contains(string(encoded), secret) {
			t.Error("inventory retained a bearer secret")
		}
	}
}

func TestProtectionInventoryRejectsIncompleteResponsesAndSanitizesErrors(t *testing.T) {
	for _, test := range []struct {
		name, project, aliases string
		status                 int
	}{
		{name: "missing pagination", aliases: `{"aliases":[]}`},
		{name: "missing aliases", aliases: `{"pagination":{"next":null}}`},
		{name: "wrong alias owner", aliases: `{"aliases":[{"uid":"alias_1","alias":"preview.example","projectId":"prj_other","protectionBypass":{}}],"pagination":{}}`},
		{name: "cursor cycle", aliases: `{"aliases":[],"pagination":{"next":100}}`},
		{name: "forbidden", status: 403, aliases: `{"error":{"code":"forbidden","message":"hidden-secret"}}`},
		{name: "malformed response", aliases: `{"protectionBypass":{"hidden-secret":`},
		{name: "wrong team", project: `{"id":"prj_123","accountId":"team_other"}`},
		{name: "malformed bypass", project: `{"id":"prj_123","accountId":"team_123","protectionBypass":{"hidden-secret":{"scope":123}}}`},
		{name: "new trusted setting", project: `{"id":"prj_123","accountId":"team_123","optionsAllowlist":null,"trustedSources":{"unknownAccessSetting":true}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				requests++
				if requests > 5 {
					t.Fatal("pagination did not terminate")
				}
				if strings.Contains(req.URL.Path, "/projects/") {
					if test.project != "" {
						fmt.Fprint(w, test.project)
					} else {
						fmt.Fprint(w, `{"id":"prj_123","accountId":"team_123","protectionBypass":{},"optionsAllowlist":null,"trustedSources":null}`)
					}
					return
				}
				if test.status != 0 {
					w.WriteHeader(test.status)
				}
				fmt.Fprint(w, test.aliases)
			})
			_, err := c.GetProjectProtectionInventory(context.Background(), "prj_123", "team_123")
			if err == nil {
				t.Fatal("expected inventory read error")
			}
			if strings.Contains(err.Error(), "hidden-secret") {
				t.Fatal("error exposed a bearer secret")
			}
		})
	}
}

func TestProtectionInventoryMissingMapsAndUnknownScopes(t *testing.T) {
	c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "/projects/") {
			fmt.Fprint(w, `{"id":"prj_123","accountId":"team_123","protectionBypass":{"secret":{"scope":"future-scope"}}}`)
			return
		}
		fmt.Fprint(w, `{"aliases":[{"uid":"alias_1","alias":"preview.example","projectId":"prj_123"}],"pagination":{}}`)
	})
	got, err := c.GetProjectProtectionInventory(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Status != "unknown" || len(got.IncompleteReasons) != 6 {
		t.Fatalf("unexpected inventory: %+v", got)
	}
}

func readInventoryFixture(t *testing.T, bypasses, trusted string) vercelclient.ProjectProtectionInventory {
	t.Helper()
	c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "/projects/") {
			fmt.Fprintf(w, `{"id":"prj_123","accountId":"team_123","protectionBypass":%s,"optionsAllowlist":null,"trustedSources":%s}`, bypasses, trusted)
			return
		}
		fmt.Fprint(w, `{"aliases":[],"pagination":{}}`)
	})
	got, err := c.GetProjectProtectionInventory(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestProtectionInventoryApprovalKeys(t *testing.T) {
	// Select the automation entry because entries are sorted by approval key.
	automationKey := func(bypass string) string {
		for _, e := range readInventoryFixture(t, bypass, `null`).Entries {
			if e.Scope == "automation-bypass" {
				return e.ApprovalKey
			}
		}
		t.Fatal("missing automation entry")
		return ""
	}
	first := automationKey(`{"secret":{"scope":"automation-bypass","note":"first","createdAt":1}}`)
	if first != automationKey(`{"secret":{"createdAt":2,"note":"second","scope":"automation-bypass"}}`) {
		t.Error("metadata changed approval key")
	}
	if first != automationKey(`{"secret":{"scope":"automation-bypass","isEnvVar":true}}`) {
		t.Error("explicit legacy default changed approval key")
	}
	for _, bypass := range []string{
		`{"rotated-secret":{"scope":"automation-bypass"}}`,
		`{"secret":{"scope":"automation-bypass","isEnvVar":false}}`,
		`{"secret":{"scope":"automation-bypass","expires":2000000000}}`,
		`{"secret":{"scope":"automation-bypass","futureAccessSetting":true}}`,
	} {
		if first == automationKey(bypass) {
			t.Error("access change did not change approval key")
		}
	}
	trustedKey := func(trusted string) string {
		for _, e := range readInventoryFixture(t, `{}`, trusted).Entries {
			if e.Scope == "trusted-oidc" {
				return e.ApprovalKey
			}
		}
		t.Fatal("missing trusted-oidc entry")
		return ""
	}
	first = trustedKey(`{"oidcProviders":{"https://issuer.example":[{"claims":{"aud":["a","b"],"note":["required"]},"to":{"slugs":["preview","production"]}}]}}`)
	if first != trustedKey(`{"oidcProviders":{"https://issuer.example":[{"label":"cosmetic","claims":{"note":["required"],"aud":["b","a"]},"to":{"slugs":["production","preview"]}}]}}`) {
		t.Error("reordering changed approval key")
	}
	if first == trustedKey(`{"oidcProviders":{"https://issuer.example":[{"claims":{"aud":["a","b"],"note":["changed"]},"to":{"slugs":["preview","production"]}}]}}`) {
		t.Error("claim named note was excluded from approval")
	}
	defaults := readInventoryFixture(t, `{}`, `null`)
	denied := readInventoryFixture(t, `{}`, `{"projects":{"prj_123":{"customAllow":[]}},"enableVercelCiSameRepository":false}`)
	if len(defaults.Entries) != 2 || len(denied.Entries) != 1 || denied.Entries[0].CustomAllow == nil || denied.Entries[0].Status != "disabled" {
		t.Fatal("default and explicit deny semantics were lost")
	}
}
