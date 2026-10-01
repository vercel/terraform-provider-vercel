package client_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	vercelclient "github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestUpdateShareableLink(t *testing.T) {
	for _, tt := range []struct {
		name               string
		ttl                *int64
		revoke, regenerate bool
		body               map[string]any
	}{
		{name: "create", body: map[string]any{}},
		{name: "rotate", ttl: int64Pointer(600), revoke: true, regenerate: true, body: map[string]any{"ttl": float64(600), "revoke": map[string]any{"secret": "old-secret", "regenerate": true}}},
		{name: "delete", revoke: true, body: map[string]any{"revoke": map[string]any{"secret": "old-secret", "regenerate": false}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assertRequest(t, r, "PATCH", "/aliases/alias_123/protection-bypass", "team_123", tt.body)
				fmt.Fprint(w, `{"protectionBypass":{"new-secret":{"scope":"shareable-link","expires":600,"createdAt":123},"*":{"scope":"alias-protection-override"}}}`)
			})
			bypasses, err := c.UpdateShareableLink(context.Background(), vercelclient.UpdateShareableLinkRequest{AliasID: "alias_123", TeamID: "team_123", TTLSeconds: tt.ttl, Secret: "old-secret", Revoke: tt.revoke, Regenerate: tt.regenerate})
			if err != nil {
				t.Fatal(err)
			}
			if bypasses["new-secret"].Expires == nil || *bypasses["new-secret"].Expires != 600 {
				t.Fatal("expiry was not decoded")
			}
			if bypasses["*"].Scope != "alias-protection-override" {
				t.Fatal("unrelated entry was lost")
			}
		})
	}
}
func int64Pointer(value int64) *int64 { return &value }

func TestShareableLinkErrorsDoNotExposeSecrets(t *testing.T) {
	for _, body := range []string{
		`{"error":{"code":"not_found","message":"secret=leaked-secret"}}`,
		`{"protectionBypass":{"leaked-secret":`,
		`{"secret":"leaked-secret"}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(body, "protectionBypass") {
					w.WriteHeader(200)
				} else {
					w.WriteHeader(404)
				}
				fmt.Fprint(w, body)
			})
			_, err := c.UpdateShareableLink(context.Background(), vercelclient.UpdateShareableLinkRequest{AliasID: "alias_123"})
			if err == nil || strings.Contains(err.Error(), "leaked-secret") {
				t.Fatalf("unsafe error: %v", err)
			}
			if strings.Contains(body, `"code":"not_found"`) && !vercelclient.NotFound(err) {
				t.Fatal("not-found semantics were lost")
			}
		})
	}
}
