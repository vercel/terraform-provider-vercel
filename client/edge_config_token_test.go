package client_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"

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

			var logs bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &logs)
			if err := cl.DeleteEdgeConfigToken(ctx, tc.Request); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotBody != tc.WantBody {
				t.Errorf("body = %s, want %s", gotBody, tc.WantBody)
			}
			if strings.Contains(logs.String(), tc.Request.Token) {
				t.Error("delete logs contain the token")
			}
		})
	}
}

func TestGetEdgeConfigTokenErrorsOmitToken(t *testing.T) {
	const token = "secret-token"
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "not found", status: http.StatusNotFound, body: `{"error":{"code":"not_found","message":"` + token + `"}}`},
		{name: "forbidden", status: http.StatusForbidden, body: `{"error":{"code":"` + token + `","message":"` + token + `"}}`},
		{name: "rate limited", status: http.StatusTooManyRequests, body: `{"error":{"code":"rate_limited","message":"` + token + `"}}`},
		{name: "invalid error body", status: http.StatusBadGateway, body: token},
		{name: "invalid success body", status: http.StatusOK, body: token},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			t.Cleanup(server.Close)
			cl := client.New("test").WithBaseURL(server.URL)
			var logs bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &logs)
			_, err := cl.GetEdgeConfigToken(ctx, client.EdgeConfigTokenRequest{EdgeConfigID: "ecfg_1", Token: token})
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error()+logs.String(), token) {
				t.Error("error or logs contain the token")
			}
			if client.NotFound(err) != (tc.status == http.StatusNotFound) {
				t.Errorf("NotFound(%v) = %t", err, client.NotFound(err))
			}
		})
	}
}

func TestGetEdgeConfigToken(t *testing.T) {
	var gotPath, gotQuery, gotHeader string
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotHeader = r.Header.Get("x-vercel-global-config-token")
		fmt.Fprintln(w, `{"id":"a","label":"my token","edgeConfigId":"ecfg_xxx","createdAt":1,"partialToken":"tkn********"}`)
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

	if gotPath != "/v1/global-config/ecfg_xxx/token" {
		t.Errorf("path = %q, want the lookup route", gotPath)
	}
	if strings.Contains(gotPath+gotQuery, req.Token) {
		t.Errorf("request URL contains the token: %s?%s", gotPath, gotQuery)
	}
	if gotHeader != req.Token {
		t.Errorf("x-vercel-global-config-token = %q, want %q", gotHeader, req.Token)
	}
	if got.Token != req.Token {
		t.Errorf("Token = %q, want %q", got.Token, req.Token)
	}
	if got.ID != "a" || got.Label != "my token" || got.EdgeConfigID != req.EdgeConfigID || got.TeamID != req.TeamID {
		t.Errorf("unexpected token: %+v", got)
	}

	wantCS := "https://edge-config.vercel.com/ecfg_xxx?token=tkn_xxx"
	if cs := got.ConnectionString(); cs != wantCS {
		t.Errorf("ConnectionString() = %q, want %q", cs, wantCS)
	}
}
