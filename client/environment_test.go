package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListEnvironments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/projects/prj_123/custom-environments" {
			t.Fatalf("path = %s, want /v1/projects/prj_123/custom-environments", r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		fmt.Fprint(w, `{
			"environments": [
				{
					"id": "env_staging",
					"slug": "staging",
					"type": "preview",
					"description": "QA staging",
					"branchMatcher": {"type": "startsWith", "pattern": "staging-"},
					"createdAt": 1700000000000,
					"updatedAt": 1700000001000
				}
			]
		}`)
	}))
	defer server.Close()

	c := New("test-token")
	c.baseURL = server.URL

	envs, err := c.ListEnvironments(context.Background(), ListEnvironmentsRequest{
		TeamID:    "team_123",
		ProjectID: "prj_123",
	})
	if err != nil {
		t.Fatalf("ListEnvironments returned error: %v", err)
	}
	if len(envs) != 4 {
		t.Fatalf("len(envs) = %d, want 4", len(envs))
	}

	prod := envs[0]
	if prod.ID != EnvironmentIDProduction || prod.Lifecycle != EnvironmentLifecycleSystem || prod.ManagedBy != EnvironmentManagedByVercel {
		t.Fatalf("unexpected production env: %+v", prod)
	}
	if prod.Capabilities.Manage || !prod.Capabilities.Domains || !prod.Capabilities.Deployments {
		t.Fatalf("unexpected production capabilities: %+v", prod.Capabilities)
	}

	preview := envs[1]
	if preview.ID != EnvironmentIDPreview || preview.BranchMatcher == nil || preview.BranchMatcher.Pattern != "All unassigned git branches" {
		t.Fatalf("unexpected preview env: %+v", preview)
	}

	dev := envs[2]
	if dev.Capabilities.Domains || dev.Capabilities.Deployments || dev.Capabilities.Manage {
		t.Fatalf("unexpected development capabilities: %+v", dev.Capabilities)
	}

	custom := envs[3]
	if custom.ID != "env_staging" || custom.Slug != "staging" || custom.Lifecycle != EnvironmentLifecycleCustom {
		t.Fatalf("unexpected custom env: %+v", custom)
	}
	if custom.CreatedAt == nil || *custom.CreatedAt != 1700000000000 {
		t.Fatalf("unexpected custom createdAt: %v", custom.CreatedAt)
	}
	if !custom.Capabilities.Manage {
		t.Fatalf("custom environments should be manageable")
	}
}

func TestGetEnvironment(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"environments": [
				{"id": "env_staging", "slug": "staging", "type": "preview", "description": "QA staging"}
			]
		}`)
	}))
	defer server.Close()

	c := New("test-token")
	c.baseURL = server.URL

	cases := []struct {
		name     string
		idOrSlug string
		wantID   string
	}{
		{name: "system by id", idOrSlug: EnvironmentIDPreview, wantID: EnvironmentIDPreview},
		{name: "system by name", idOrSlug: "Preview", wantID: EnvironmentIDPreview},
		{name: "custom by id", idOrSlug: "env_staging", wantID: "env_staging"},
		{name: "custom by slug", idOrSlug: "staging", wantID: "env_staging"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, err := c.GetEnvironment(context.Background(), GetEnvironmentRequest{
				TeamID:    "team_123",
				ProjectID: "prj_123",
				IDOrSlug:  tc.idOrSlug,
			})
			if err != nil {
				t.Fatalf("GetEnvironment returned error: %v", err)
			}
			if env.ID != tc.wantID {
				t.Fatalf("ID = %q, want %q", env.ID, tc.wantID)
			}
		})
	}
}

func TestIsSystemEnvironmentID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		id   string
		want bool
	}{
		{id: EnvironmentIDProduction, want: true},
		{id: EnvironmentIDPreview, want: true},
		{id: EnvironmentIDDevelopment, want: true},
		{id: "Production", want: false},
		{id: "staging", want: false},
		{id: "", want: false},
	}
	for _, tc := range cases {
		if got := IsSystemEnvironmentID(tc.id); got != tc.want {
			t.Fatalf("IsSystemEnvironmentID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestGetEnvironmentNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"environments":[]}`)
	}))
	defer server.Close()

	c := New("test-token")
	c.baseURL = server.URL

	_, err := c.GetEnvironment(context.Background(), GetEnvironmentRequest{
		ProjectID: "prj_123",
		IDOrSlug:  "missing",
	})
	if !NotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
