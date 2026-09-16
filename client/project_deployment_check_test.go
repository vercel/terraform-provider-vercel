package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectDeploymentCheckCRUD(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q", got)
		}
		if r.Method == http.MethodPost {
			if r.URL.Path != "/v2/projects/prj_123/checks" {
				t.Fatalf("path = %q", r.URL.Path)
			}
			var body CreateProjectDeploymentCheckRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.ProjectID != "" || body.TeamID != "" || body.Source == nil || body.Source.ExternalCheckName != "e2e" {
				t.Fatalf("body = %#v", body)
			}
		} else if r.URL.Path != "/v2/projects/prj_123/checks/check_123" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body["name"] != "E2E tests" {
				t.Fatalf("PATCH body = %#v", body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodDelete {
			_, _ = w.Write([]byte(`{"id":"check_123","name":"E2E","ownerId":"team_123","projectId":"prj_123","isRerequestable":false,"requires":"deployment-url","source":{"kind":"git-provider","provider":"github","externalCheckName":"e2e"},"sourceKind":"git-provider","blocks":"deployment-promotion","targets":["production"],"timeout":300,"createdAt":1,"updatedAt":2}`))
		} else {
			_, _ = w.Write([]byte(`{"success":true}`))
		}
	}))
	t.Cleanup(server.Close)
	c := New("TOKEN").WithBaseURL(server.URL)
	ctx := context.Background()
	source := ProjectDeploymentCheckSource{Kind: "git-provider", Provider: "github", ExternalCheckName: "e2e"}
	created, err := c.CreateProjectDeploymentCheck(ctx, CreateProjectDeploymentCheckRequest{ProjectID: "prj_123", TeamID: "team_123", Name: "E2E", Requires: "deployment-url", Source: &source})
	if err != nil || created.ID != "check_123" {
		t.Fatalf("create = %#v, %v", created, err)
	}
	if _, err = c.GetProjectDeploymentCheck(ctx, "prj_123", "check_123", "team_123"); err != nil {
		t.Fatal(err)
	}
	name := "E2E tests"
	if _, err = c.UpdateProjectDeploymentCheck(ctx, UpdateProjectDeploymentCheckRequest{ProjectID: "prj_123", TeamID: "team_123", ID: "check_123", Name: &name}); err != nil {
		t.Fatal(err)
	}
	if err = c.DeleteProjectDeploymentCheck(ctx, "prj_123", "check_123", "team_123"); err != nil {
		t.Fatal(err)
	}
	want := []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodDelete}
	if fmt.Sprint(methods) != fmt.Sprint(want) {
		t.Fatalf("methods = %v, want %v", methods, want)
	}
}
