package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newProtectionBypassTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := New("test-token")
	client.baseURL = server.URL
	return client
}

// When the project has another bypass scope (e.g. shareable-link) the response
// includes that entry alongside the automation-bypass entry. The client must
// filter by scope rather than erroring on the total count.
func TestUpdateProtectionBypassForAutomation_FiltersNonAutomationScopes(t *testing.T) {
	t.Parallel()

	client := newProtectionBypassTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v10/projects/prj_123/protection-bypass", "team_123", map[string]any{})
		_, _ = w.Write([]byte(`{
			"protectionBypass": {
				"generated-automation-secret": {"scope": "automation-bypass"},
				"shareable-link-secret": {"scope": "shareable-link"}
			}
		}`))
	})

	secret, err := client.UpdateProtectionBypassForAutomation(context.Background(), UpdateProtectionBypassForAutomationRequest{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		NewValue:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "generated-automation-secret" {
		t.Fatalf("expected generated-automation-secret, got %q", secret)
	}
}

// During a rotation we send generate+revoke and the API can transiently return
// both old and new automation-bypass entries. When the caller provided the new
// secret we should return it directly without inspecting the response map.
func TestUpdateProtectionBypassForAutomation_RotationReturnsCallerSecret(t *testing.T) {
	t.Parallel()

	client := newProtectionBypassTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"protectionBypass": {
				"old-automation-secret": {"scope": "automation-bypass"},
				"new-automation-secret": {"scope": "automation-bypass"}
			}
		}`))
	})

	secret, err := client.UpdateProtectionBypassForAutomation(context.Background(), UpdateProtectionBypassForAutomationRequest{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		NewValue:  true,
		NewSecret: "new-automation-secret",
		OldSecret: "old-automation-secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "new-automation-secret" {
		t.Fatalf("expected new-automation-secret, got %q", secret)
	}
}

// If the response has no automation-bypass entry at all the client should
// surface a clear error rather than silently returning an empty secret.
func TestUpdateProtectionBypassForAutomation_NoAutomationEntryErrors(t *testing.T) {
	t.Parallel()

	client := newProtectionBypassTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"protectionBypass": {
				"shareable-link-secret": {"scope": "shareable-link"}
			}
		}`))
	})

	_, err := client.UpdateProtectionBypassForAutomation(context.Background(), UpdateProtectionBypassForAutomationRequest{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		NewValue:  true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected number of automation-bypass items (0)") {
		t.Fatalf("error did not mention automation-bypass count: %v", err)
	}
}

// Disabling (NewValue=false) should never inspect the response map, so even a
// response with multiple entries must not produce an error.
func TestUpdateProtectionBypassForAutomation_DisableIgnoresResponseShape(t *testing.T) {
	t.Parallel()

	client := newProtectionBypassTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"protectionBypass": {
				"shareable-link-a": {"scope": "shareable-link"},
				"shareable-link-b": {"scope": "shareable-link"}
			}
		}`))
	})

	secret, err := client.UpdateProtectionBypassForAutomation(context.Background(), UpdateProtectionBypassForAutomationRequest{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		NewValue:  false,
		OldSecret: "old-automation-secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "" {
		t.Fatalf("expected empty secret on disable, got %q", secret)
	}
}
