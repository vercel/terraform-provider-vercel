package vercel_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func testOrganisationTeamListed(ctx context.Context, c *client.Client, organisationID, teamID string) (bool, error) {
	cursor := ""
	for {
		page, err := c.ListOrganisationTeams(ctx, organisationID, cursor)
		if err != nil {
			return false, err
		}
		for _, team := range page.Teams {
			if team.TeamID == teamID {
				return true, nil
			}
		}
		if page.Cursor == "" {
			return false, nil
		}
		if page.Cursor == cursor {
			return false, fmt.Errorf("organisation teams API returned a repeated cursor")
		}
		cursor = page.Cursor
	}
}

func TestOrganisationTeamDestroyVerification(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		pages     []string
		listed    bool
		wantError bool
	}{
		{"deleted_child_is_inaccessible", 200, []string{`{"teams":[]}`}, false, false},
		{"still_listed", 200, []string{`{"teams":[{"teamId":"team_1"}]}`}, true, false},
		{"listed_on_later_page", 200, []string{`{"teams":[],"cursor":"next+/="}`, `{"teams":[{"teamId":"team_1"}]}`}, true, false},
		{"missing_from_all_pages", 200, []string{`{"teams":[],"cursor":"next+/="}`, `{"teams":[{"teamId":"team_other"}]}`}, false, false},
		{"repeated_cursor", 200, []string{`{"teams":[],"cursor":"next+/="}`, `{"teams":[],"cursor":"next+/="}`}, false, true},
		{"organisation_forbidden", 403, []string{`{"error":{"code":"forbidden","message":"Not authorized"}}`}, false, true},
		{"organisation_error", 500, []string{`{"error":{"code":"failure","message":"failure"}}`}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/v1/organizations/org_1/teams" {
					t.Errorf("destroy verification must stay organisation-scoped: %s %s", req.Method, req.URL)
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, `{"error":{"code":"forbidden","message":"Not authorized"}}`)
					return
				}
				if req.URL.Query().Has("teamId") || req.URL.Query().Has("orgId") {
					t.Errorf("unexpected query scope: %s", req.URL)
				}
				if reads > 0 && req.URL.Query().Get("cursor") != "next+/=" {
					t.Errorf("unexpected cursor: %s", req.URL)
				}
				if reads >= len(tc.pages) {
					t.Error("unexpected extra list request")
					http.Error(w, "unexpected request", 500)
					return
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.pages[reads])
				reads++
			}))
			defer server.Close()
			c := client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "unrelated"})
			listed, err := testOrganisationTeamListed(context.Background(), c, "org_1", "team_1")
			if listed != tc.listed || (err != nil) != tc.wantError {
				t.Fatalf("listed=%v, err=%v", listed, err)
			}
		})
	}
}
