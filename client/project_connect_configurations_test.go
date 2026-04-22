package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const projectTestTeamID = "team_123"

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
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}

		var body CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if len(body.ConnectConfigurations) != 1 {
			t.Fatalf("expected 1 connect configuration, got %d", len(body.ConnectConfigurations))
		}

		entry := body.ConnectConfigurations[0]
		if entry.EnvID != "production" {
			t.Fatalf("expected envId production, got %s", entry.EnvID)
		}
		if entry.ConnectConfigurationID != "cfg_prod" {
			t.Fatalf("expected connect configuration id cfg_prod, got %s", entry.ConnectConfigurationID)
		}
		if entry.Passive {
			t.Fatal("expected passive false")
		}
		if !entry.BuildsEnabled {
			t.Fatal("expected buildsEnabled true")
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

	project, err := client.CreateProject(context.Background(), projectTestTeamID, request)
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 1 {
		t.Fatalf("expected 1 connect configuration in response, got %d", len(project.ConnectConfigurations))
	}

	entry := project.ConnectConfigurations[0]
	if entry.EnvID != "production" {
		t.Fatalf("expected envId production, got %s", entry.EnvID)
	}
	if entry.ConnectConfigurationID != "cfg_prod" {
		t.Fatalf("expected connect configuration id cfg_prod, got %s", entry.ConnectConfigurationID)
	}
	if entry.Passive {
		t.Fatal("expected passive false")
	}
	if !entry.BuildsEnabled {
		t.Fatal("expected buildsEnabled true")
	}
}

func TestCreateProjectOmitsConnectConfigurationsWhenUnset(t *testing.T) {
	t.Parallel()

	request := CreateProjectRequest{Name: "test-project"}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/v8/projects" {
			t.Fatalf("expected path /v8/projects, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if _, ok := body["connectConfigurations"]; ok {
			t.Fatal("did not expect connectConfigurations in request body")
		}

		_, _ = w.Write([]byte(`{"id":"prj_123","name":"test-project"}`))
	})

	project, err := client.CreateProject(context.Background(), projectTestTeamID, request)
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	if project.ConnectConfigurations != nil {
		t.Fatal("expected nil connect configurations in response")
	}
}

func TestCreateProjectIncludesMultipleConnectConfigurations(t *testing.T) {
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
			{
				EnvID:                  "preview",
				ConnectConfigurationID: "cfg_preview",
				Passive:                true,
				BuildsEnabled:          false,
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
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}

		var body CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if len(body.ConnectConfigurations) != 2 {
			t.Fatalf("expected 2 connect configurations, got %d", len(body.ConnectConfigurations))
		}

		entry0 := body.ConnectConfigurations[0]
		if entry0.EnvID != "production" || entry0.ConnectConfigurationID != "cfg_prod" || entry0.Passive || !entry0.BuildsEnabled {
			t.Fatalf("unexpected first connect configuration: %+v", entry0)
		}

		entry1 := body.ConnectConfigurations[1]
		if entry1.EnvID != "preview" || entry1.ConnectConfigurationID != "cfg_preview" || !entry1.Passive || entry1.BuildsEnabled {
			t.Fatalf("unexpected second connect configuration: %+v", entry1)
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

	project, err := client.CreateProject(context.Background(), projectTestTeamID, request)
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 2 {
		t.Fatalf("expected 2 connect configurations in response, got %d", len(project.ConnectConfigurations))
	}

	entry0 := project.ConnectConfigurations[0]
	if entry0.EnvID != "production" || entry0.ConnectConfigurationID != "cfg_prod" || entry0.Passive || !entry0.BuildsEnabled {
		t.Fatalf("unexpected first connect configuration: %+v", entry0)
	}

	entry1 := project.ConnectConfigurations[1]
	if entry1.EnvID != "preview" || entry1.ConnectConfigurationID != "cfg_preview" || !entry1.Passive || entry1.BuildsEnabled {
		t.Fatalf("unexpected second connect configuration: %+v", entry1)
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

	request := UpdateProjectRequest{ConnectConfigurations: &connectConfigurations}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v9/projects/prj_123" {
			t.Fatalf("expected path /v9/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
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

		entry := (*body.ConnectConfigurations)[0]
		if entry.EnvID != "preview" {
			t.Fatalf("expected envId preview, got %s", entry.EnvID)
		}
		if entry.ConnectConfigurationID != "cfg_preview" {
			t.Fatalf("expected connect configuration id cfg_preview, got %s", entry.ConnectConfigurationID)
		}
		if !entry.Passive {
			t.Fatal("expected passive true")
		}
		if entry.BuildsEnabled {
			t.Fatal("expected buildsEnabled false")
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

	project, err := client.UpdateProject(context.Background(), "prj_123", projectTestTeamID, request)
	if err != nil {
		t.Fatalf("UpdateProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 1 {
		t.Fatalf("expected 1 connect configuration in response, got %d", len(project.ConnectConfigurations))
	}

	entry := project.ConnectConfigurations[0]
	if entry.EnvID != "preview" {
		t.Fatalf("expected envId preview, got %s", entry.EnvID)
	}
	if entry.ConnectConfigurationID != "cfg_preview" {
		t.Fatalf("expected connect configuration id cfg_preview, got %s", entry.ConnectConfigurationID)
	}
	if !entry.Passive {
		t.Fatal("expected passive true")
	}
	if entry.BuildsEnabled {
		t.Fatal("expected buildsEnabled false")
	}
}

func TestUpdateProjectOmitsConnectConfigurationsWhenNil(t *testing.T) {
	t.Parallel()

	request := UpdateProjectRequest{}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v9/projects/prj_123" {
			t.Fatalf("expected path /v9/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if _, ok := body["connectConfigurations"]; ok {
			t.Fatal("did not expect connectConfigurations in request body")
		}

		_, _ = w.Write([]byte(`{"id":"prj_123","name":"test-project"}`))
	})

	project, err := client.UpdateProject(context.Background(), "prj_123", projectTestTeamID, request)
	if err != nil {
		t.Fatalf("UpdateProject returned error: %v", err)
	}

	if project.ConnectConfigurations != nil {
		t.Fatal("expected nil connect configurations in response")
	}
}

func TestUpdateProjectIncludesEmptyConnectConfigurationsWhenExplicitlyClearing(t *testing.T) {
	t.Parallel()

	empty := []ConnectConfiguration{}
	request := UpdateProjectRequest{ConnectConfigurations: &empty}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v9/projects/prj_123" {
			t.Fatalf("expected path /v9/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		connectConfigurationsRaw, ok := body["connectConfigurations"]
		if !ok {
			t.Fatal("expected connectConfigurations in request body")
		}

		connectConfigurations, ok := connectConfigurationsRaw.([]any)
		if !ok {
			t.Fatalf("expected connectConfigurations to be an array, got %T", connectConfigurationsRaw)
		}

		if len(connectConfigurations) != 0 {
			t.Fatalf("expected empty connectConfigurations array, got %d entries", len(connectConfigurations))
		}

		_, _ = w.Write([]byte(`{"id":"prj_123","name":"test-project","connectConfigurations":[]}`))
	})

	project, err := client.UpdateProject(context.Background(), "prj_123", projectTestTeamID, request)
	if err != nil {
		t.Fatalf("UpdateProject returned error: %v", err)
	}

	if project.ConnectConfigurations == nil {
		t.Fatal("expected non-nil empty connect configurations in response")
	}

	if len(project.ConnectConfigurations) != 0 {
		t.Fatalf("expected 0 connect configurations in response, got %d", len(project.ConnectConfigurations))
	}
}

func TestUpdateProjectIncludesMultipleConnectConfigurations(t *testing.T) {
	t.Parallel()

	connectConfigurations := []ConnectConfiguration{
		{
			EnvID:                  "production",
			ConnectConfigurationID: "cfg_prod",
			Passive:                false,
			BuildsEnabled:          true,
		},
		{
			EnvID:                  "preview",
			ConnectConfigurationID: "cfg_preview",
			Passive:                true,
			BuildsEnabled:          false,
		},
	}

	request := UpdateProjectRequest{ConnectConfigurations: &connectConfigurations}

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v9/projects/prj_123" {
			t.Fatalf("expected path /v9/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}

		var body UpdateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.ConnectConfigurations == nil {
			t.Fatal("expected connectConfigurations in request body")
		}
		if len(*body.ConnectConfigurations) != 2 {
			t.Fatalf("expected 2 connect configurations, got %d", len(*body.ConnectConfigurations))
		}

		entry0 := (*body.ConnectConfigurations)[0]
		if entry0.EnvID != "production" || entry0.ConnectConfigurationID != "cfg_prod" || entry0.Passive || !entry0.BuildsEnabled {
			t.Fatalf("unexpected first connect configuration: %+v", entry0)
		}

		entry1 := (*body.ConnectConfigurations)[1]
		if entry1.EnvID != "preview" || entry1.ConnectConfigurationID != "cfg_preview" || !entry1.Passive || entry1.BuildsEnabled {
			t.Fatalf("unexpected second connect configuration: %+v", entry1)
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

	project, err := client.UpdateProject(context.Background(), "prj_123", projectTestTeamID, request)
	if err != nil {
		t.Fatalf("UpdateProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 2 {
		t.Fatalf("expected 2 connect configurations in response, got %d", len(project.ConnectConfigurations))
	}

	entry0 := project.ConnectConfigurations[0]
	if entry0.EnvID != "production" || entry0.ConnectConfigurationID != "cfg_prod" || entry0.Passive || !entry0.BuildsEnabled {
		t.Fatalf("unexpected first connect configuration: %+v", entry0)
	}

	entry1 := project.ConnectConfigurations[1]
	if entry1.EnvID != "preview" || entry1.ConnectConfigurationID != "cfg_preview" || !entry1.Passive || entry1.BuildsEnabled {
		t.Fatalf("unexpected second connect configuration: %+v", entry1)
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
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
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

	project, err := client.GetProject(context.Background(), "prj_123", projectTestTeamID)
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}

	if len(project.ConnectConfigurations) != 2 {
		t.Fatalf("expected 2 connect configurations, got %d", len(project.ConnectConfigurations))
	}

	entry0 := project.ConnectConfigurations[0]
	if entry0.EnvID != "production" || entry0.ConnectConfigurationID != "cfg_prod" || entry0.Passive || !entry0.BuildsEnabled {
		t.Fatalf("unexpected first connect configuration: %+v", entry0)
	}

	entry1 := project.ConnectConfigurations[1]
	if entry1.EnvID != "preview" || entry1.ConnectConfigurationID != "cfg_preview" || !entry1.Passive || entry1.BuildsEnabled {
		t.Fatalf("unexpected second connect configuration: %+v", entry1)
	}
}

func TestGetProjectReturnsNilConnectConfigurationsWhenFieldOmitted(t *testing.T) {
	t.Parallel()

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/v10/projects/prj_123" {
			t.Fatalf("expected path /v10/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}
		_, _ = w.Write([]byte(`{"id":"prj_123","name":"test-project"}`))
	})

	project, err := client.GetProject(context.Background(), "prj_123", projectTestTeamID)
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}

	if project.ConnectConfigurations != nil {
		t.Fatal("expected nil connect configurations when field is omitted")
	}
}

func TestGetProjectReturnsEmptyConnectConfigurationsWhenFieldIsEmptyArray(t *testing.T) {
	t.Parallel()

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/v10/projects/prj_123" {
			t.Fatalf("expected path /v10/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}
		_, _ = w.Write([]byte(`{"id":"prj_123","name":"test-project","connectConfigurations":[]}`))
	})

	project, err := client.GetProject(context.Background(), "prj_123", projectTestTeamID)
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}

	if project.ConnectConfigurations == nil {
		t.Fatal("expected non-nil empty connect configurations when field is empty array")
	}
	if len(project.ConnectConfigurations) != 0 {
		t.Fatalf("expected 0 connect configurations, got %d", len(project.ConnectConfigurations))
	}
}

func TestGetProjectReturnsNilConnectConfigurationsWhenFieldIsNull(t *testing.T) {
	t.Parallel()

	client := newProjectTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/v10/projects/prj_123" {
			t.Fatalf("expected path /v10/projects/prj_123, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != projectTestTeamID {
			t.Fatalf("expected teamId %s, got %s", projectTestTeamID, r.URL.Query().Get("teamId"))
		}
		_, _ = w.Write([]byte(`{"id":"prj_123","name":"test-project","connectConfigurations":null}`))
	})

	project, err := client.GetProject(context.Background(), "prj_123", projectTestTeamID)
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}

	if project.ConnectConfigurations != nil {
		t.Fatal("expected nil connect configurations when field is null")
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
