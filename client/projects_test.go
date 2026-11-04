package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProjectsPagination(t *testing.T) {
	calls := 0
	repository := "https://github.com/example/web?ref=main&other=value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.Method != http.MethodGet || req.URL.Path != "/v10/projects" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
		}
		if req.URL.Query().Get("teamId") != "team_default" || req.URL.Query().Get("repoUrl") != repository || req.URL.Query().Get("limit") != "100" {
			t.Errorf("unexpected query: %s", req.URL.RawQuery)
		}
		switch calls {
		case 1:
			if req.URL.Query().Has("until") {
				t.Error("first page has until")
			}
			projects := make([]ProjectResponse, 100)
			for i := range projects {
				projects[i] = ProjectResponse{ID: fmt.Sprintf("prj_%d", i), AccountID: "team_default"}
			}
			next := int64(123)
			_ = json.NewEncoder(w).Encode(ListProjectsResponse{Projects: projects, Pagination: PageInfo{Next: &next}})
		case 2:
			if req.URL.Query().Get("until") != "123" {
				t.Error("second page did not use cursor")
			}
			fmt.Fprint(w, `{"projects":[{"id":"prj_100","accountId":"team_default","resourceConfig":{"buildMachineType":"enhanced","buildMachineSelection":"elastic"}}],"pagination":{"next":null}}`)
		default:
			t.Error("unexpected additional page")
		}
	}))
	defer server.Close()
	c := New("test").WithBaseURL(server.URL).WithTeam(Team{ID: "team_default"})
	projects, err := c.ListProjectsWithOptions(context.Background(), ListProjectsRequest{RepositoryURL: repository})
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 101 || calls != 2 {
		t.Fatalf("projects = %d, calls = %d", len(projects), calls)
	}
	if projects[100].TeamID != "team_default" || projects[100].ResourceConfig.BuildMachineType != "elastic" {
		t.Fatalf("project not normalized: %#v", projects[100])
	}
}

func TestListProjectsDoesNotReturnPartialResults(t *testing.T) {
	for _, fail := range []bool{true, false} {
		t.Run(fmt.Sprintf("page_error_%t", fail), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if calls > 1 && fail {
					http.Error(w, `{"error":{"code":"bad_request","message":"page failed"}}`, http.StatusBadRequest)
					return
				}
				fmt.Fprint(w, `{"projects":[{"id":"prj_1"}],"pagination":{"next":123}}`)
			}))
			defer server.Close()
			projects, err := New("test").WithBaseURL(server.URL).ListProjects(context.Background(), "")
			if err == nil || projects != nil {
				t.Fatalf("projects = %#v, err = %v; expected no partial results", projects, err)
			}
			if calls != 2 {
				t.Fatalf("calls = %d, want 2", calls)
			}
		})
	}
}

func TestListProjectsTeamOverrideAndPersonalOwnership(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("teamId") != "team_override" {
			t.Errorf("teamId = %q", req.URL.Query().Get("teamId"))
		}
		fmt.Fprint(w, `{"projects":[{"id":"prj_team","accountId":"team_override"},{"id":"prj_personal","accountId":"user_123"}],"pagination":{"next":null}}`)
	}))
	defer server.Close()
	projects, err := New("test").WithBaseURL(server.URL).WithTeam(Team{ID: "team_default"}).ListProjects(context.Background(), "team_override")
	if err != nil {
		t.Fatal(err)
	}
	if projects[0].TeamID != "team_override" || projects[1].TeamID != "" {
		t.Fatalf("unexpected ownership: %#v", projects)
	}
}

func TestListProjectsRejectsCursorCycle(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		next := 123
		if req.URL.Query().Get("until") == "123" {
			next = 456
		}
		fmt.Fprintf(w, `{"projects":[],"pagination":{"next":%d}}`, next)
	}))
	defer server.Close()
	projects, err := New("test").WithBaseURL(server.URL).ListProjects(context.Background(), "")
	if err == nil || projects != nil || calls != 3 {
		t.Fatalf("projects = %#v, err = %v, calls = %d", projects, err, calls)
	}
}
