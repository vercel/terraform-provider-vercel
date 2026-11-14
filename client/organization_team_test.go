package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrganizationTeamPagination(t *testing.T) {
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
	team, err := New("TOKEN").WithBaseURL(server.URL).WithTeam(Team{ID: "unrelated"}).GetOrganizationTeam(context.Background(), "org_1", "team_1")
	if err != nil || team.TeamID != "team_1" || team.BillingPlan != "platform" || calls != 2 {
		t.Fatalf("team = %#v, err = %v, calls = %d", team, err, calls)
	}
}

func TestGetOrganizationTeamMissingAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name               string
		listStatus         int
		organizationStatus int
		organizationBody   string
		notFound           bool
		code               string
	}{
		{"index_lag", 200, 200, `{"organizationId":"org_1"}`, false, "organization_team_not_yet_visible"},
		{"deleted", 200, 404, `{"error":{"code":"not_found","message":"Team not found"}}`, true, "not_found"},
		{"detached", 200, 200, `{"organizationId":null}`, true, "organization_team_not_found"},
		{"moved", 200, 200, `{"organizationId":"org_other"}`, true, "organization_team_not_found"},
		{"organization_forbidden", 200, 403, `{"error":{"code":"forbidden","message":"failure"}}`, false, "forbidden"},
		{"organization_error", 200, 500, `{"error":{"code":"failure","message":"failure"}}`, false, "failure"},
		{"list_forbidden", 403, 0, "", false, "failure"},
		{"list_error", 500, 0, "", false, "failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			organizationReads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				switch r.URL.Path {
				case "/v1/organizations/org_1/teams":
					w.WriteHeader(tc.listStatus)
					if tc.listStatus == 200 {
						fmt.Fprint(w, `{"teams":[]}`)
					} else {
						fmt.Fprint(w, `{"error":{"code":"failure","message":"failure"}}`)
					}
				case "/v1/teams/team_1/organization":
					organizationReads++
					if tc.organizationStatus == 0 {
						t.Error("unexpected organization lookup after a list error")
						http.Error(w, "unexpected request", 500)
						return
					}
					w.WriteHeader(tc.organizationStatus)
					fmt.Fprint(w, tc.organizationBody)
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					http.Error(w, "unexpected request", 400)
				}
			}))
			defer server.Close()
			_, err := New("TOKEN").WithBaseURL(server.URL).WithTeam(Team{ID: "unrelated"}).GetOrganizationTeam(context.Background(), "org_1", "team_1")
			var apiErr APIError
			if !errors.As(err, &apiErr) || NotFound(err) != tc.notFound || apiErr.Code != tc.code {
				t.Fatalf("unexpected error: %v", err)
			}
			if (organizationReads == 1) != (tc.listStatus == 200) {
				t.Fatalf("organization reads = %d", organizationReads)
			}
		})
	}
}
