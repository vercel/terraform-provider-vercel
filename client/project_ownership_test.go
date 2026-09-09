package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectResponseOwnership(t *testing.T) {
	for _, operation := range []struct {
		name string
		run  func(*Client, string) (ProjectResponse, error)
	}{
		{"get", func(c *Client, team string) (ProjectResponse, error) {
			return c.GetProject(context.Background(), "prj_test", team)
		}},
		{"create", func(c *Client, team string) (ProjectResponse, error) {
			return c.CreateProject(context.Background(), team, CreateProjectRequest{})
		}},
		{"update", func(c *Client, team string) (ProjectResponse, error) {
			return c.UpdateProject(context.Background(), "prj_test", team, UpdateProjectRequest{})
		}},
		{"branch", func(c *Client, team string) (ProjectResponse, error) {
			return c.UpdateProductionBranch(context.Background(), UpdateProductionBranchRequest{ProjectID: "prj_test", TeamID: team})
		}},
		{"link", func(c *Client, team string) (ProjectResponse, error) {
			return c.LinkGitRepoToProject(context.Background(), LinkGitRepoToProjectRequest{ProjectID: "prj_test", TeamID: team})
		}},
		{"unlink", func(c *Client, team string) (ProjectResponse, error) {
			return c.UnlinkGitRepoFromProject(context.Background(), "prj_test", team)
		}},
		{"list", func(c *Client, team string) (ProjectResponse, error) {
			p, err := c.ListProjects(context.Background(), team)
			if err != nil {
				return ProjectResponse{}, err
			}
			if len(p) != 1 {
				return ProjectResponse{}, fmt.Errorf("got %d projects", len(p))
			}
			return p[0], nil
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			for _, tc := range []struct{ name, account, explicit, provider, want string }{
				{name: "unscoped team", account: "team_owner", want: "team_owner"},
				{name: "personal", account: "user_owner"},
				{name: "response overrides scope", account: "team_owner", explicit: "team_scope", provider: "team_default", want: "team_owner"},
				{name: "personal overrides scope", account: "user_owner", provider: "team_default"},
				{name: "missing owner explicit fallback", explicit: "team_scope", provider: "team_default", want: "team_scope"},
				{name: "missing owner provider fallback", provider: "team_default", want: "team_default"},
				{name: "missing owner no scope"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						scope := tc.explicit
						if scope == "" {
							scope = tc.provider
						}
						if got := r.URL.Query().Get("teamId"); got != scope {
							t.Errorf("request team = %q, want %q", got, scope)
						}
						body := fmt.Sprintf(`{"id":"prj_test","accountId":%q}`, tc.account)
						if operation.name == "list" {
							body = `{"projects":[` + body + `]}`
						}
						fmt.Fprintln(w, body)
					}))
					defer h.Close()
					c := New("test").WithBaseURL(h.URL).WithTeam(Team{ID: tc.provider})
					p, err := operation.run(c, tc.explicit)
					if err != nil {
						t.Fatal(err)
					}
					if p.TeamID != tc.want {
						t.Fatalf("team = %q, want %q", p.TeamID, tc.want)
					}
				})
			}
		})
	}
}
