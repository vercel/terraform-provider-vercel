package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseTeamOwnership(t *testing.T) {
	operations := []struct {
		name       string
		method     string
		path       string
		ownerField string
		list       bool
		call       func(*Client, string) (string, error)
	}{
		{"get attack challenge mode", "GET", "/v10/projects/prj_1", "accountId", false, func(c *Client, team string) (string, error) {
			r, err := c.GetAttackChallengeMode(context.Background(), "prj_1", team)
			return r.TeamID, err
		}},
		{"get shared environment variable", "GET", "/v1/env/env_1", "ownerId", false, func(c *Client, team string) (string, error) {
			r, err := c.GetSharedEnvironmentVariable(context.Background(), team, "env_1")
			return r.TeamID, err
		}},
		{"list shared environment variables", "GET", "/v1/env/all", "ownerId", true, func(c *Client, team string) (string, error) {
			r, err := c.ListSharedEnvironmentVariablesPage(context.Background(), ListSharedEnvironmentVariablesRequest{TeamID: team})
			if err != nil {
				return "", err
			}
			if len(r.EnvironmentVariables) != 1 {
				return "", fmt.Errorf("expected one environment variable, got %d", len(r.EnvironmentVariables))
			}
			return r.EnvironmentVariables[0].TeamID, nil
		}},
		{"get access group", "GET", "/v1/access-groups/ag_1", "teamId", false, func(c *Client, team string) (string, error) {
			r, err := c.GetAccessGroup(context.Background(), GetAccessGroupRequest{TeamID: team, AccessGroupID: "ag_1"})
			return r.TeamID, err
		}},
		{"create access group", "POST", "/v1/access-groups", "teamId", false, func(c *Client, team string) (string, error) {
			r, err := c.CreateAccessGroup(context.Background(), CreateAccessGroupRequest{TeamID: team, Name: "test"})
			return r.TeamID, err
		}},
		{"update access group", "POST", "/v1/access-groups/ag_1", "teamId", false, func(c *Client, team string) (string, error) {
			r, err := c.UpdateAccessGroup(context.Background(), UpdateAccessGroupRequest{TeamID: team, AccessGroupID: "ag_1", Name: "test"})
			return r.TeamID, err
		}},
		{"create access group project", "POST", "/v1/access-groups/ag_1/projects", "teamId", false, func(c *Client, team string) (string, error) {
			r, err := c.CreateAccessGroupProject(context.Background(), CreateAccessGroupProjectRequest{TeamID: team, AccessGroupID: "ag_1", ProjectID: "prj_1", Role: "ADMIN"})
			return r.TeamID, err
		}},
		{"update access group project", "PATCH", "/v1/access-groups/ag_1/projects/prj_1", "teamId", false, func(c *Client, team string) (string, error) {
			r, err := c.UpdateAccessGroupProject(context.Background(), UpdateAccessGroupProjectRequest{TeamID: team, AccessGroupID: "ag_1", ProjectID: "prj_1", Role: "ADMIN"})
			return r.TeamID, err
		}},
	}
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			for _, tc := range []struct{ name, owner, requestTeam, providerTeam, want string }{
				{"bare ID discovers owner", "team_owner", "", "", "team_owner"},
				{"response owner preserved", "team_owner", "team_request", "team_provider", "team_owner"},
				{"missing owner uses request", "", "team_request", "team_provider", "team_request"},
				{"missing owner uses provider", "", "", "team_provider", "team_provider"},
				{"missing owner and fallback", "", "", "", ""},
			} {
				t.Run(tc.name, func(t *testing.T) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != op.method || r.URL.Path != op.path {
							t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, op.method, op.path)
						}
						wantQuery := tc.requestTeam
						if wantQuery == "" {
							wantQuery = tc.providerTeam
						}
						if r.URL.Query().Get("teamId") != wantQuery {
							t.Errorf("teamId query = %q, want %q", r.URL.Query().Get("teamId"), wantQuery)
						}
						response := map[string]any{op.ownerField: tc.owner}
						if op.list {
							response = map[string]any{"data": []any{response}}
						}
						_ = json.NewEncoder(w).Encode(response)
					}))
					defer server.Close()
					c := New("test").WithBaseURL(server.URL).WithTeam(Team{ID: tc.providerTeam})
					got, err := op.call(c, tc.requestTeam)
					if err != nil {
						t.Fatal(err)
					}
					if got != tc.want {
						t.Errorf("TeamID = %q, want %q", got, tc.want)
					}
				})
			}
		})
	}
}
