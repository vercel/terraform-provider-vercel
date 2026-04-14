package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateProjectIncludesConnectConfigurations(t *testing.T) {
	t.Parallel()

	request := CreateProjectRequest{
		Name: "test-project",
		ConnectConfigurations: []ConnectConfiguration{
			{
				EnvID:                  "production",
				ConnectConfigurationID: "cfg_prod",
				Passive:                false,
				BuildsEnabled:          true,
			},
		},
	}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/v8/projects" {
			t.Fatalf("expected path /v8/projects, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != "team_123" {
			t.Fatalf("expected teamId team_123, got %s", r.URL.Query().Get("teamId"))
		}

		var body CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if len(body.ConnectConfigurations) != 1 {
			t.Fatalf("expected 1 connect configuration, got %d", len(body.ConnectConfigurations))
		}

		if body.ConnectConfigurations[0].ConnectConfigurationID != "cfg_prod" {
			t.Fatalf(
				"expected connect configuration id cfg_prod, got %s",
				body.ConnectConfigurations[0].ConnectConfigurationID,
			)
		}

		_, _ = w.Write([]byte(`{
			"id":"prj_123",
			"name":"test-project",
			"connectConfigurations":[
				{
					"envId":"production",
					"connectConfigurationId":"cfg_prod",
					"passive":false,
					"buildsEnabled":true
				}
			]
		}`))
	})

	project, err := client.CreateProject(context.Background(), "team_123", request)
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 1 {
		t.Fatalf("expected 1 connect configuration in response, got %d", len(project.ConnectConfigurations))
	}

	if project.ConnectConfigurations[0].EnvID != "production" {
		t.Fatalf("expected envId production, got %s", project.ConnectConfigurations[0].EnvID)
	}
}

func TestUpdateProjectIncludesConnectConfigurations(t *testing.T) {
	t.Parallel()

	connectConfigurations := []ConnectConfiguration{
		{
			EnvID:                  "preview",
			ConnectConfigurationID: "cfg_preview",
			Passive:                true,
			BuildsEnabled:          false,
		},
	}

	request := UpdateProjectRequest{
		ConnectConfigurations: &connectConfigurations,
	}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v9/projects/prj_123" {
			t.Fatalf("expected path /v9/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != "team_123" {
			t.Fatalf("expected teamId team_123, got %s", r.URL.Query().Get("teamId"))
		}

		var body UpdateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.ConnectConfigurations == nil {
			t.Fatal("expected connectConfigurations in request body")
		}

		if len(*body.ConnectConfigurations) != 1 {
			t.Fatalf("expected 1 connect configuration, got %d", len(*body.ConnectConfigurations))
		}

		if (*body.ConnectConfigurations)[0].ConnectConfigurationID != "cfg_preview" {
			t.Fatalf(
				"expected connect configuration id cfg_preview, got %s",
				(*body.ConnectConfigurations)[0].ConnectConfigurationID,
			)
		}

		_, _ = w.Write([]byte(`{
			"id":"prj_123",
			"name":"test-project",
			"connectConfigurations":[
				{
					"envId":"preview",
					"connectConfigurationId":"cfg_preview",
					"passive":true,
					"buildsEnabled":false
				}
			]
		}`))
	})

	project, err := client.UpdateProject(
		context.Background(),
		"prj_123",
		"team_123",
		request,
	)
	if err != nil {
		t.Fatalf("UpdateProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 1 {
		t.Fatalf("expected 1 connect configuration in response, got %d", len(project.ConnectConfigurations))
	}

	if !project.ConnectConfigurations[0].Passive {
		t.Fatal("expected response connect configuration to be passive")
	}
}

func TestGetProjectReadsConnectConfigurations(t *testing.T) {
	t.Parallel()

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/v10/projects/prj_123" {
			t.Fatalf("expected path /v10/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != "team_123" {
			t.Fatalf("expected teamId team_123, got %s", r.URL.Query().Get("teamId"))
		}

		_, _ = w.Write([]byte(`{
			"id":"prj_123",
			"name":"test-project",
			"connectConfigurations":[
				{
					"envId":"production",
					"connectConfigurationId":"cfg_prod",
					"passive":false,
					"buildsEnabled":true
				},
				{
					"envId":"preview",
					"connectConfigurationId":"cfg_preview",
					"passive":true,
					"buildsEnabled":false
				}
			]
		}`))
	})

	project, err := client.GetProject(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 2 {
		t.Fatalf("expected 2 connect configurations, got %d", len(project.ConnectConfigurations))
	}

	if project.ConnectConfigurations[1].ConnectConfigurationID != "cfg_preview" {
		t.Fatalf(
			"expected second configuration id cfg_preview, got %s",
			project.ConnectConfigurations[1].ConnectConfigurationID,
		)
	}
}

func newProjectTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := New("test-token")
	client.baseURL = server.URL
	return client
}
