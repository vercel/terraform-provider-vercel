package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

const (
	testEdgeConfigTokenSecret = "s3cret-token"
	testEdgeConfigTokenCS     = "https://edge-config.vercel.com/ecfg_1?token=" + testEdgeConfigTokenSecret
)

func edgeConfigTokenTestResource(t *testing.T, handler http.HandlerFunc) (*edgeConfigTokenResource, resource.SchemaResponse) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	r := &edgeConfigTokenResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	return r, schemaResponse
}

func edgeConfigTokenTestState(t *testing.T, schemaResponse resource.SchemaResponse, id string) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schemaResponse.Schema}
	diags := state.Set(context.Background(), EdgeConfigToken{
		Label:            types.StringValue("test token"),
		Token:            types.StringValue(testEdgeConfigTokenSecret),
		ID:               types.StringValue(id),
		TeamID:           types.StringValue("team_1"),
		EdgeConfigID:     types.StringValue("ecfg_1"),
		ConnectionString: types.StringValue(testEdgeConfigTokenCS),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func TestEdgeConfigTokenReadByIDKeepsTokenFromState(t *testing.T) {
	ctx := context.Background()
	r, schemaResponse := edgeConfigTokenTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/edge-config/ecfg_1/tokens" {
			t.Errorf("unexpected request to %s", req.URL.Path)
			http.NotFound(w, req)
			return
		}
		if req.URL.Query().Get("teamId") != "team_1" {
			t.Errorf("unexpected team query: %s", req.URL.RawQuery)
		}
		fmt.Fprint(w, `[{"id":"tok_1","label":"updated label","edgeConfigId":"ecfg_1","partialToken":"s3c********","token":"ignored-legacy-token"}]`)
	})

	resp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Read(ctx, resource.ReadRequest{State: edgeConfigTokenTestState(t, schemaResponse, "tok_1")}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	var got EdgeConfigToken
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.Token.ValueString() != testEdgeConfigTokenSecret {
		t.Errorf("token = %q, want the token from state", got.Token.ValueString())
	}
	if got.ConnectionString.ValueString() != testEdgeConfigTokenCS {
		t.Errorf("connection_string = %q, want %q", got.ConnectionString.ValueString(), testEdgeConfigTokenCS)
	}
	if got.ID.ValueString() != "tok_1" {
		t.Errorf("id = %q, want tok_1", got.ID.ValueString())
	}
	if got.Label.ValueString() != "updated label" {
		t.Errorf("label = %q, want updated label", got.Label.ValueString())
	}
}

func TestEdgeConfigTokenReadErrorsKeepState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "forbidden", status: http.StatusForbidden, body: `{"error":{"code":"forbidden","message":"Forbidden"}}`},
		{name: "server error", status: http.StatusInternalServerError, body: `{"error":{"code":"internal","message":"Unavailable"}}`},
		{name: "invalid list", status: http.StatusOK, body: `{"tokens":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, schemaResponse := edgeConfigTokenTestResource(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			})
			state := edgeConfigTokenTestState(t, schemaResponse, "tok_1")
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error")
			}
			if !resp.State.Raw.Equal(state.Raw) {
				t.Error("state changed after a failed read")
			}
		})
	}
}

func TestEdgeConfigTokenDeleteByID(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			r, schemaResponse := edgeConfigTokenTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodDelete || req.URL.Path != "/v1/edge-config/ecfg_1/tokens" || req.URL.Query().Get("teamId") != "team_1" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				var body map[string][]string
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if len(body) != 1 || len(body["ids"]) != 1 || body["ids"][0] != "tok_1" {
					t.Errorf("expected only the token ID in the delete body, got %v", body)
				}
				w.WriteHeader(status)
			})
			var resp resource.DeleteResponse
			r.Delete(context.Background(), resource.DeleteRequest{State: edgeConfigTokenTestState(t, schemaResponse, "tok_1")}, &resp)
			if resp.Diagnostics.HasError() != (status == http.StatusForbidden) {
				t.Errorf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

func TestEdgeConfigTokenReadRemovesDeletedToken(t *testing.T) {
	r, schemaResponse := edgeConfigTokenTestResource(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"id":"tok_other","label":"other","edgeConfigId":"ecfg_1"}]`)
	})

	resp := resource.ReadResponse{State: edgeConfigTokenTestState(t, schemaResponse, "tok_1")}
	r.Read(context.Background(), resource.ReadRequest{State: edgeConfigTokenTestState(t, schemaResponse, "tok_1")}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

func TestEdgeConfigTokenReadWithoutIDLooksUpByToken(t *testing.T) {
	ctx := context.Background()
	r, schemaResponse := edgeConfigTokenTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/global-config/ecfg_1/token" || req.Header.Get("x-vercel-global-config-token") != testEdgeConfigTokenSecret {
			t.Errorf("unexpected request to %s", req.URL.Path)
			http.NotFound(w, req)
			return
		}
		fmt.Fprint(w, `{"id":"tok_1","label":"test token","edgeConfigId":"ecfg_1"}`)
	})

	resp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Read(ctx, resource.ReadRequest{State: edgeConfigTokenTestState(t, schemaResponse, "")}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	var got EdgeConfigToken
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.ID.ValueString() != "tok_1" || got.ConnectionString.ValueString() != testEdgeConfigTokenCS {
		t.Errorf("unexpected state: id=%q connection_string=%q", got.ID.ValueString(), got.ConnectionString.ValueString())
	}
}

func TestEdgeConfigTokenImportErrorsOmitToken(t *testing.T) {
	for _, tc := range []struct {
		name     string
		handler  http.HandlerFunc
		closed   bool
		importID string
	}{
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":{"code":"internal","message":"failed for %s"}}`, testEdgeConfigTokenSecret)
			},
		},
		{
			name:    "network error",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
		},
		{
			name:     "invalid URL escape",
			handler:  func(http.ResponseWriter, *http.Request) {},
			importID: "team_1/ecfg_1/" + testEdgeConfigTokenSecret + "%",
		},
		{
			name:     "invalid import format",
			handler:  func(http.ResponseWriter, *http.Request) {},
			importID: "team_1/ecfg_1/" + testEdgeConfigTokenSecret + "/extra",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			if tc.closed {
				server.Close()
			} else {
				t.Cleanup(server.Close)
			}
			r := &edgeConfigTokenResource{client: client.New("test").WithBaseURL(server.URL)}
			var schemaResponse resource.SchemaResponse
			r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)

			resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			importID := tc.importID
			if importID == "" {
				importID = "team_1/ecfg_1/" + testEdgeConfigTokenSecret
			}
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: importID}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error")
			}
			for _, d := range resp.Diagnostics {
				if strings.Contains(d.Summary()+d.Detail(), testEdgeConfigTokenSecret) {
					t.Errorf("diagnostic contains the token: %s: %s", d.Summary(), d.Detail())
				}
			}
		})
	}
}
