package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectCronsTeamOwnership(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		for _, tc := range []struct {
			name, response, providerTeam, wantTeam string
		}{
			{"bare ID", `{"accountId":"team_owner","crons":{"disabledAt":123}}`, "", "team_owner"},
			{"provider fallback", `{"crons":{"disabledAt":123}}`, "team_default", "team_default"},
			{"personal owner", `{"accountId":"user_owner","crons":{"disabledAt":123}}`, "", ""},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != method || r.URL.Query().Get("teamId") != tc.providerTeam {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.response))
				}))
				t.Cleanup(server.Close)
				c := New("TOKEN").WithBaseURL(server.URL).WithTeam(Team{ID: tc.providerTeam})
				var out ProjectCrons
				var err error
				if method == http.MethodGet {
					out, err = c.GetProjectCrons(context.Background(), "prj_123", "")
				} else {
					out, err = c.UpdateProjectCrons(context.Background(), ProjectCrons{ProjectID: "prj_123", Enabled: false})
				}
				if err != nil {
					t.Fatal(err)
				}
				if out.TeamID != tc.wantTeam || out.ProjectID != "prj_123" || out.Enabled {
					t.Fatalf("unexpected crons: %#v; want team %q and disabled", out, tc.wantTeam)
				}
			})
		}
	}
}
