package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTeam(t *testing.T) {
	type TestCase struct {
		Name         string
		ResponseJSON string
	}

	for _, tc := range []TestCase{
		{
			Name:         "SAML",
			ResponseJSON: `{ "saml": { "roles": { "A": "OWNER", "B": { "accessGroupId": "foo" } } } }`,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintln(w, tc.ResponseJSON)
			}))
			cl := New("INVALID")
			cl.baseURL = fmt.Sprintf("http://%s", h.Listener.Addr().String())
			_, err := cl.GetTeam(context.Background(), "INVALID")
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestUpdateTeamBuildMachineDefault(t *testing.T) {
	basic := "basic"
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		resourceConfig := body["resourceConfig"].(map[string]any)
		buildMachine := resourceConfig["buildMachine"].(map[string]any)
		if got := buildMachine["default"]; got != "basic" {
			t.Fatalf("buildMachine.default = %v, want basic", got)
		}
		fmt.Fprintln(w, `{ "id": "team_1", "resourceConfig": { "buildMachine": { "default": "basic" } } }`)
	}))
	defer h.Close()

	cl := New("INVALID")
	cl.baseURL = fmt.Sprintf("http://%s", h.Listener.Addr().String())
	team, err := cl.UpdateTeam(context.Background(), UpdateTeamRequest{
		TeamID: "team_1",
		ResourceConfig: &TeamResourceConfig{
			BuildMachine: &TeamBuildMachine{Default: &basic},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if team.ResourceConfig == nil || team.ResourceConfig.BuildMachine == nil || team.ResourceConfig.BuildMachine.Default == nil {
		t.Fatal("expected build machine default in response")
	}
	if got := *team.ResourceConfig.BuildMachine.Default; got != "basic" {
		t.Fatalf("buildMachine.default = %q, want basic", got)
	}
}
