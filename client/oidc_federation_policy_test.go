package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const oidcFederationPolicyResponse = `{
	"policyId":"pol_123",
	"clientId":"cli_123",
	"appId":"cli_123",
	"issuerUrl":"https://token.actions.githubusercontent.com",
	"teamId":"team_123",
	"name":"GitHub Actions",
	"claims":[{"name":"repository","values":[{"value":"vercel/functions","wildcards":false}]}],
	"permissions":["*"],
	"commands":null,
	"resources":{"projectIds":["prj_123"]}
}`

func TestCreateOIDCFederationPolicy(t *testing.T) {
	name := "GitHub Actions"
	var got CreateOIDCFederationPolicyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != oidcFederationPoliciesPath {
			t.Fatalf("request = %s %s, want POST %s", r.Method, r.URL.Path, oidcFederationPoliciesPath)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oidcFederationPolicyResponse))
	}))
	t.Cleanup(server.Close)

	policy, err := New("TOKEN").WithBaseURL(server.URL).CreateOIDCFederationPolicy(context.Background(), CreateOIDCFederationPolicyRequest{
		TeamID:    "team_123",
		ClientID:  "cli_123",
		IssuerURL: "https://token.actions.githubusercontent.com",
		Name:      &name,
		Claims: []OIDCClaim{{
			Name:   "repository",
			Values: []OIDCClaimValue{{Value: "vercel/functions", Wildcards: false}},
		}},
		Permissions: []string{"*"},
		Resources:   &OIDCResources{ProjectIDs: []string{"prj_123"}},
	})
	if err != nil {
		t.Fatalf("CreateOIDCFederationPolicy() error = %v", err)
	}
	if got.TeamID != "" || got.ClientID != "cli_123" || got.IssuerURL != "https://token.actions.githubusercontent.com" {
		t.Fatalf("create request = %#v", got)
	}
	if got.Name == nil || *got.Name != name || len(got.Claims) != 1 || got.Claims[0].Values[0].Value != "vercel/functions" {
		t.Fatalf("create policy fields = %#v", got)
	}
	if len(got.Permissions) != 1 || got.Permissions[0] != "*" || got.Resources == nil || got.Resources.ProjectIDs[0] != "prj_123" {
		t.Fatalf("create policy boundaries = %#v", got)
	}
	assertOIDCFederationPolicyResponse(t, policy)
}

func TestGetOIDCFederationPolicyListsAndFindsPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != oidcFederationPoliciesPath {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, oidcFederationPoliciesPath)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if policyID := r.URL.Query().Get("policyId"); policyID != "" {
			t.Fatalf("policyId = %q, want empty for list endpoint", policyID)
		}
		_, _ = w.Write([]byte(`{"policies":[{"policyId":"pol_other"},` + oidcFederationPolicyResponse + `]}`))
	}))
	t.Cleanup(server.Close)

	policy, err := New("TOKEN").WithBaseURL(server.URL).GetOIDCFederationPolicy(context.Background(), "pol_123", "team_123")
	if err != nil {
		t.Fatalf("GetOIDCFederationPolicy() error = %v", err)
	}
	assertOIDCFederationPolicyResponse(t, policy)
}

func TestGetOIDCFederationPolicyReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"policies":[]}`))
	}))
	t.Cleanup(server.Close)

	_, err := New("TOKEN").WithBaseURL(server.URL).GetOIDCFederationPolicy(context.Background(), "pol_missing", "")
	if !NotFound(err) {
		t.Fatalf("GetOIDCFederationPolicy() error = %v, want NotFound", err)
	}
}

func TestUpdateOIDCFederationPolicy(t *testing.T) {
	name := ""
	permissions := []string{"project:read"}
	var got map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := oidcFederationPoliciesPath + "/pol_123"; r.Method != http.MethodPatch || r.URL.Path != want {
			t.Fatalf("request = %s %s, want PATCH %s", r.Method, r.URL.Path, want)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(oidcFederationPolicyResponse))
	}))
	t.Cleanup(server.Close)

	policy, err := New("TOKEN").WithBaseURL(server.URL).UpdateOIDCFederationPolicy(context.Background(), UpdateOIDCFederationPolicyRequest{
		TeamID:      "team_123",
		PolicyID:    "pol_123",
		Name:        &name,
		Permissions: &permissions,
	})
	if err != nil {
		t.Fatalf("UpdateOIDCFederationPolicy() error = %v", err)
	}
	for _, field := range []string{"teamId", "policyId", "claims", "commands", "resources"} {
		if _, ok := got[field]; ok {
			t.Fatalf("update request unexpectedly contains %q: %s", field, got[field])
		}
	}
	if string(got["name"]) != `""` || string(got["permissions"]) != `["project:read"]` {
		t.Fatalf("update request = %v", got)
	}
	assertOIDCFederationPolicyResponse(t, policy)
}

func TestDeleteOIDCFederationPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := oidcFederationPoliciesPath + "/pol_123"; r.Method != http.MethodDelete || r.URL.Path != want {
			t.Fatalf("request = %s %s, want DELETE %s", r.Method, r.URL.Path, want)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		_, _ = w.Write([]byte(`{"policyId":"pol_123"}`))
	}))
	t.Cleanup(server.Close)

	err := New("TOKEN").WithBaseURL(server.URL).DeleteOIDCFederationPolicy(context.Background(), "pol_123", "team_123")
	if err != nil {
		t.Fatalf("DeleteOIDCFederationPolicy() error = %v", err)
	}
}

func assertOIDCFederationPolicyResponse(t *testing.T, policy OIDCFederationPolicy) {
	t.Helper()
	if policy.PolicyID != "pol_123" || policy.ClientID != "cli_123" {
		t.Fatalf("policy identifiers = %#v", policy)
	}
	if policy.Name == nil || *policy.Name != "GitHub Actions" || policy.TeamID != "team_123" {
		t.Fatalf("policy metadata = %#v", policy)
	}
	if len(policy.Claims) != 1 || policy.Claims[0].Values[0].Value != "vercel/functions" || policy.Claims[0].Values[0].Wildcards {
		t.Fatalf("policy claims = %#v", policy.Claims)
	}
	if policy.Commands != nil || policy.Resources == nil || policy.Resources.ProjectIDs[0] != "prj_123" {
		t.Fatalf("policy boundaries = commands %#v, resources %#v", policy.Commands, policy.Resources)
	}
}
