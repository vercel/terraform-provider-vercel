package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The metadata read must never ask the API to decrypt, must page, must not return
// the ciphertext, and must report a missing variable as NotFound.
func TestGetEnvironmentVariableMetadata(t *testing.T) {
	var single, decrypted int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/projects/prj_1/env" {
			single++
			http.Error(w, `{"error":{"code":"forbidden"}}`, http.StatusForbidden)
			return
		}
		if r.URL.Query().Get("decrypt") != "false" {
			decrypted++
		}
		page := map[string]any{"envs": []map[string]any{{"id": "env_a", "key": "A", "value": "CIPHERTEXT", "type": "encrypted"}}, "pagination": map[string]any{"next": 1}}
		if r.URL.Query().Get("until") == "1" {
			page = map[string]any{"envs": []map[string]any{{"id": "env_b", "key": "B", "value": "CIPHERTEXT", "type": "sensitive"}}, "pagination": map[string]any{"next": nil}}
		}
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()
	c := New("TOKEN").WithBaseURL(server.URL)

	e, err := c.GetEnvironmentVariableMetadata(context.Background(), "prj_1", "team_1", "env_b")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if e.Key != "B" || e.Value != "" || e.TeamID != "team_1" {
		t.Fatalf("got %+v, want key B, empty value, team_1", e)
	}

	_, err = c.GetEnvironmentVariableMetadata(context.Background(), "prj_1", "team_1", "env_missing")
	if !NotFound(err) {
		t.Fatalf("missing variable: err = %v, want NotFound", err)
	}
	if single != 0 || decrypted != 0 {
		t.Fatalf("single-variable calls = %d, decrypting list calls = %d; want 0 and 0", single, decrypted)
	}
}
