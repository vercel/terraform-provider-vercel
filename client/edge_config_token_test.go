package client_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestGetEdgeConfigToken(t *testing.T) {
	type TestCase struct {
		Name         string
		ResponseJSON string
	}

	// Both shapes must round-trip to a struct where Token matches the value
	// passed in the request. "WithToken" is what GET /v1/edge-config/:id/token/:token
	// emits today (the `token` field is currently deprecated); "WithoutToken"
	// is the forthcoming shape once the deprecated field is removed.
	for _, tc := range []TestCase{
		{
			Name:         "WithToken",
			ResponseJSON: `{"token":"tkn_xxx","id":"a","label":"my token","edgeConfigId":"ecfg_xxx","createdAt":1,"partialToken":"tkn_********"}`,
		},
		{
			Name:         "WithoutToken",
			ResponseJSON: `{"id":"a","label":"my token","edgeConfigId":"ecfg_xxx","createdAt":1,"partialToken":"tkn_********"}`,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintln(w, tc.ResponseJSON)
			}))
			t.Cleanup(h.Close)

			cl := client.New("INVALID").WithBaseURL(fmt.Sprintf("http://%s", h.Listener.Addr().String()))
			req := client.EdgeConfigTokenRequest{
				Token:        "tkn_xxx",
				TeamID:       "team_123",
				EdgeConfigID: "ecfg_xxx",
			}
			got, err := cl.GetEdgeConfigToken(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Token != req.Token {
				t.Errorf("Token = %q, want %q", got.Token, req.Token)
			}
			if got.EdgeConfigID != req.EdgeConfigID {
				t.Errorf("EdgeConfigID = %q, want %q", got.EdgeConfigID, req.EdgeConfigID)
			}
			if got.TeamID != req.TeamID {
				t.Errorf("TeamID = %q, want %q", got.TeamID, req.TeamID)
			}
			if got.ID != "a" {
				t.Errorf("ID = %q, want %q", got.ID, "a")
			}
			if got.Label != "my token" {
				t.Errorf("Label = %q, want %q", got.Label, "my token")
			}

			wantCS := "https://edge-config.vercel.com/ecfg_xxx?token=tkn_xxx"
			if cs := got.ConnectionString(); cs != wantCS {
				t.Errorf("ConnectionString() = %q, want %q", cs, wantCS)
			}
		})
	}
}
