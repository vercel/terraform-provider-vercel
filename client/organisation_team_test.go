package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrganisationTeamPagination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/organizations/org_1/teams" || r.URL.Query().Has("teamId") || r.URL.Query().Has("orgId") {
			t.Errorf("unexpected scope: %s", r.URL)
		}
		switch calls {
		case 1:
			fmt.Fprint(w, `{"teams":[],"cursor":"next+/="}`)
		case 2:
			if r.URL.Query().Get("cursor") != "next+/=" {
				t.Errorf("cursor = %q", r.URL.Query().Get("cursor"))
			}
			fmt.Fprint(w, `{"teams":[{"organizationId":"org_1","teamId":"team_1","name":"Product","slug":"product"}]}`)
		default:
			t.Errorf("unexpected call %d", calls)
		}
	}))
	defer server.Close()
	team, err := New("TOKEN").WithBaseURL(server.URL).WithTeam(Team{ID: "unrelated"}).GetOrganisationTeam(context.Background(), "org_1", "team_1")
	if err != nil || team.TeamID != "team_1" || team.BillingPlan != "platform" || calls != 2 {
		t.Fatalf("team = %#v, err = %v, calls = %d", team, err, calls)
	}
}

func TestGetOrganisationTeamMissingAndErrors(t *testing.T) {
	for _, status := range []int{200, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				if status == 200 {
					fmt.Fprint(w, `{"teams":[]}`)
				} else {
					fmt.Fprint(w, `{"error":{"code":"failure","message":"failure"}}`)
				}
			}))
			defer server.Close()
			_, err := New("TOKEN").WithBaseURL(server.URL).GetOrganisationTeam(context.Background(), "org_1", "team_1")
			if err == nil || NotFound(err) != (status == 200) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
