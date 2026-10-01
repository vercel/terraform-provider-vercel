package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPassportAPIRequests(t *testing.T) {
	configured := &Passport{ConnectorID: "scl_oidc", DeploymentType: "preview"}
	var disabled *Passport
	for _, tc := range []struct {
		name     string
		passport **Passport
	}{
		{"omitted", nil}, {"enabled", &configured}, {"disabled", &disabled},
	} {
		for _, team := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/project", true: "/team"}[team], func(t *testing.T) {
				key, path := "passport", "/v9/projects/prj_1"
				if team {
					key, path = "defaultPassport", "/v2/teams/team_1"
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPatch || r.URL.Path != path {
						t.Errorf("request = %s %s", r.Method, r.URL.Path)
					}
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					got, present := body[key]
					if present != (tc.passport != nil) {
						t.Errorf("Passport field present = %v", present)
					}
					if tc.passport != nil {
						want, _ := json.Marshal(*tc.passport)
						if string(got) != string(want) {
							t.Errorf("Passport = %s, want %s", got, want)
						}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "id_1", key: configured})
				}))
				defer server.Close()
				cl := New("test").WithBaseURL(server.URL)
				var got *Passport
				if team {
					response, err := cl.UpdateTeam(context.Background(), UpdateTeamRequest{TeamID: "team_1", DefaultPassport: tc.passport})
					if err != nil {
						t.Fatal(err)
					}
					got = response.DefaultPassport
				} else {
					response, err := cl.UpdateProject(context.Background(), "prj_1", "team_1", UpdateProjectRequest{Passport: tc.passport})
					if err != nil {
						t.Fatal(err)
					}
					got = response.Passport
				}
				if !reflect.DeepEqual(got, configured) {
					t.Fatalf("response Passport = %#v", got)
				}
			})
		}
	}
}
