package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateVCRRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/vcr/repository/repo_123":
			if r.Method != http.MethodPatch {
				t.Fatalf("method = %s, want PATCH", r.Method)
			}
			if projectID := r.URL.Query().Get("projectId"); projectID != "prj_123" {
				t.Fatalf("projectId = %q, want prj_123", projectID)
			}
			if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
				t.Fatalf("teamId = %q, want team_123", teamID)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if len(body) != 1 || body["public"] != true {
				t.Fatalf("body = %#v, want only public=true", body)
			}
			_, _ = w.Write([]byte(`{"repository":{"id":"repo_123","projectId":"prj_123","name":"example","public":true}}`))
		case "/v2/teams/team_123":
			_, _ = w.Write([]byte(`{"id":"team_123","slug":"acme"}`))
		case "/v10/projects/prj_123":
			_, _ = w.Write([]byte(`{"id":"prj_123","name":"storefront"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	repository, err := New("TOKEN").WithBaseURL(server.URL).UpdateVCRRepository(context.Background(), UpdateVCRRepositoryRequest{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		IDOrName:  "repo_123",
		Public:    true,
	})
	if err != nil {
		t.Fatalf("UpdateVCRRepository() error = %v", err)
	}
	if !repository.Public {
		t.Fatal("repository.Public = false, want true")
	}
	if repository.URL != "vcr.vercel.com/acme/storefront/example" {
		t.Fatalf("repository.URL = %q, want vcr.vercel.com/acme/storefront/example", repository.URL)
	}
	if repository.TeamID != "team_123" {
		t.Fatalf("repository.TeamID = %q, want team_123", repository.TeamID)
	}
}
