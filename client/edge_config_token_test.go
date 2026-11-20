package client_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestGetEdgeConfigTokenByID(t *testing.T) {
	var gotPath string
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprintln(w, `[
			{"id":"a","label":"first","edgeConfigId":"ecfg_xxx","createdAt":1,"partialToken":"tkn********"},
			{"id":"b","label":"second","edgeConfigId":"ecfg_xxx","createdAt":2,"partialToken":"abc********"}
		]`)
	}))
	t.Cleanup(h.Close)
	cl := client.New("INVALID").WithBaseURL(fmt.Sprintf("http://%s", h.Listener.Addr().String()))

	t.Run("Found", func(t *testing.T) {
		got, err := cl.GetEdgeConfigTokenByID(context.Background(), client.EdgeConfigTokenRequest{
			ID:           "b",
			TeamID:       "team_123",
			EdgeConfigID: "ecfg_xxx",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotPath != "/v1/edge-config/ecfg_xxx/tokens" {
			t.Errorf("path = %q, want the token list", gotPath)
		}
		if got.ID != "b" || got.Label != "second" || got.EdgeConfigID != "ecfg_xxx" || got.TeamID != "team_123" {
			t.Errorf("unexpected token: %+v", got)
		}
		if got.Token != "" {
			t.Errorf("Token = %q, want empty", got.Token)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := cl.GetEdgeConfigTokenByID(context.Background(), client.EdgeConfigTokenRequest{
			ID:           "missing",
			EdgeConfigID: "ecfg_xxx",
		})
		if !client.NotFound(err) {
			t.Fatalf("expected a not found error, got %v", err)
		}
	})
}

func TestDeleteEdgeConfigToken(t *testing.T) {
	for _, tc := range []struct {
		Name     string
		Request  client.EdgeConfigTokenRequest
		WantBody string
	}{
		{
			Name:     "ByID",
			Request:  client.EdgeConfigTokenRequest{ID: "a", Token: "tkn_xxx", EdgeConfigID: "ecfg_xxx"},
			WantBody: `{"ids":["a"]}`,
		},
		{
			Name:     "ByTokenWithoutID",
			Request:  client.EdgeConfigTokenRequest{Token: "tkn_xxx", EdgeConfigID: "ecfg_xxx"},
			WantBody: `{"tokens":["tkn_xxx"]}`,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			var gotBody string
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				gotBody = strings.TrimSpace(string(b))
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(h.Close)
			cl := client.New("INVALID").WithBaseURL(fmt.Sprintf("http://%s", h.Listener.Addr().String()))

			if err := cl.DeleteEdgeConfigToken(context.Background(), tc.Request); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotBody != tc.WantBody {
				t.Errorf("body = %s, want %s", gotBody, tc.WantBody)
			}
		})
	}
}

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
