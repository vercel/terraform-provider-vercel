package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeBuildMachineType(t *testing.T) {
	type testCase struct {
		name              string
		resourceConfig    *ResourceConfigResponse
		expectedType      string
		expectedSelection string
	}

	for _, tc := range []testCase{
		{
			name:           "nil resource config is a no-op",
			resourceConfig: nil,
		},
		{
			name: "selection=elastic overrides concrete type to elastic",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "enhanced",
				BuildMachineSelection: "elastic",
			},
			expectedType:      "elastic",
			expectedSelection: "elastic",
		},
		{
			name: "selection=elastic overrides standard to elastic",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "standard",
				BuildMachineSelection: "elastic",
			},
			expectedType:      "elastic",
			expectedSelection: "elastic",
		},
		{
			name: "selection=fixed leaves buildMachineType untouched",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "enhanced",
				BuildMachineSelection: "fixed",
			},
			expectedType:      "enhanced",
			expectedSelection: "fixed",
		},
		{
			name: "empty selection leaves buildMachineType untouched",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "turbo",
				BuildMachineSelection: "",
			},
			expectedType:      "turbo",
			expectedSelection: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &ProjectResponse{ResourceConfig: tc.resourceConfig}
			r.normalizeBuildMachineType()

			if tc.resourceConfig == nil {
				if r.ResourceConfig != nil {
					t.Fatalf("expected nil resource config to remain nil")
				}
				return
			}
			if got := r.ResourceConfig.BuildMachineType; got != tc.expectedType {
				t.Errorf("BuildMachineType: got %q, want %q", got, tc.expectedType)
			}
			if got := r.ResourceConfig.BuildMachineSelection; got != tc.expectedSelection {
				t.Errorf("BuildMachineSelection: got %q, want %q", got, tc.expectedSelection)
			}
		})
	}
}

func TestGetProjectNormalizesElasticBuildMachine(t *testing.T) {
	type testCase struct {
		name         string
		responseJSON string
		expectedType string
	}

	for _, tc := range []testCase{
		{
			name: "API returns selection=elastic with concrete type -> provider returns elastic",
			responseJSON: `{
				"id": "proj_1",
				"name": "test",
				"resourceConfig": {
					"buildMachineType": "enhanced",
					"buildMachineSelection": "elastic"
				}
			}`,
			expectedType: "elastic",
		},
		{
			name: "API returns selection=fixed -> provider returns concrete type",
			responseJSON: `{
				"id": "proj_1",
				"name": "test",
				"resourceConfig": {
					"buildMachineType": "turbo",
					"buildMachineSelection": "fixed"
				}
			}`,
			expectedType: "turbo",
		},
		{
			name: "API omits selection -> provider returns concrete type",
			responseJSON: `{
				"id": "proj_1",
				"name": "test",
				"resourceConfig": {
					"buildMachineType": "enhanced"
				}
			}`,
			expectedType: "enhanced",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintln(w, tc.responseJSON)
			}))
			defer h.Close()

			cl := New("INVALID")
			cl.baseURL = fmt.Sprintf("http://%s", h.Listener.Addr().String())

			r, err := cl.GetProject(context.Background(), "proj_1", "")
			if err != nil {
				t.Fatalf("GetProject: %v", err)
			}
			if r.ResourceConfig == nil {
				t.Fatalf("expected resourceConfig to be set")
			}
			if got := r.ResourceConfig.BuildMachineType; got != tc.expectedType {
				t.Errorf("BuildMachineType: got %q, want %q", got, tc.expectedType)
			}
		})
	}
}

func TestGetProjectTeamOwnership(t *testing.T) {
	for _, tc := range []struct {
		name         string
		accountID    string
		teamID       string
		providerTeam string
		requestTeam  string
		wantTeam     string
	}{
		{name: "bare project discovers team", accountID: "team_123", wantTeam: "team_123"},
		{name: "explicit team", accountID: "team_123", teamID: "team_123", requestTeam: "team_123", wantTeam: "team_123"},
		{name: "provider team", accountID: "team_123", providerTeam: "team_123", requestTeam: "team_123", wantTeam: "team_123"},
		{name: "explicit team overrides provider", accountID: "team_123", teamID: "team_123", providerTeam: "team_other", requestTeam: "team_123", wantTeam: "team_123"},
		{name: "personal project", accountID: "user_123"},
		{name: "missing owner preserves explicit team", teamID: "team_123", requestTeam: "team_123", wantTeam: "team_123"},
		{name: "missing owner preserves provider team", providerTeam: "team_123", requestTeam: "team_123", wantTeam: "team_123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/v10/projects/prj_123" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				if got := req.URL.Query().Get("teamId"); got != tc.requestTeam {
					t.Errorf("request teamId = %q, want %q", got, tc.requestTeam)
				}
				fmt.Fprintf(w, `{"id":"prj_123","accountId":%q}`, tc.accountID)
			}))
			defer server.Close()

			c := New("test").WithBaseURL(server.URL).WithTeam(Team{ID: tc.providerTeam})
			project, err := c.GetProject(context.Background(), "prj_123", tc.teamID)
			if err != nil {
				t.Fatal(err)
			}
			if project.TeamID != tc.wantTeam {
				t.Fatalf("TeamID = %q, want %q", project.TeamID, tc.wantTeam)
			}
		})
	}
}
