package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rsschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func shareableTestState() ShareableLinkResourceState {
	return ShareableLinkResourceState{ShareableLink: ShareableLink{
		ID: types.StringValue("alias_123"), Alias: types.StringValue("preview.vercel.app"), TeamID: types.StringValue("team_123"), ProjectID: types.StringValue("prj_123"),
		Secret: types.StringValue("old-secret"), URL: types.StringValue("https://preview.vercel.app/?_vercel_share=old-secret"), CreatedAt: types.Int64Value(1000), CreatedBy: types.StringValue("user_123"), ExpiresAt: types.Int64Null(), Active: types.BoolValue(true),
	}, TTLSeconds: types.Int64Null(), RotationID: types.StringNull()}
}
func shareableResourceState(t *testing.T, r *shareableLinkResource, model ShareableLinkResourceState) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}
func shareableTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return client.New("abcdefghijklmnopqrstuvwx").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"})
}
func writeShareableAlias(w http.ResponseWriter, bypasses any) {
	json.NewEncoder(w).Encode(map[string]any{"uid": "alias_123", "alias": "preview.vercel.app", "projectId": "prj_123", "protectionBypass": bypasses})
}
func TestShareableLinkCreateAndRotation(t *testing.T) {
	for _, rotate := range []bool{false, true} {
		t.Run(fmt.Sprint("rotate=", rotate), func(t *testing.T) {
			gets, patches := 0, 0
			r := &shareableLinkResource{client: shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Query().Get("teamId") != "team_123" {
					t.Error("missing default team")
				}
				if req.Method == http.MethodGet {
					gets++
					writeShareableAlias(w, map[string]any{"*": map[string]any{"scope": "alias-protection-override"}})
					return
				}
				patches++
				if req.Method != "PATCH" || req.URL.Path != "/aliases/alias_123/protection-bypass" {
					t.Errorf("unexpected mutation %s %s", req.Method, req.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["ttl"] != float64(600) {
					t.Errorf("ttl=%v", body["ttl"])
				}
				if rotate {
					revoke, ok := body["revoke"].(map[string]any)
					if !ok || revoke["secret"] != "old-secret" || revoke["regenerate"] != true {
						t.Error("rotation did not revoke the state-owned secret")
					}
				} else if _, ok := body["revoke"]; ok {
					t.Error("create unexpectedly revoked a link")
				}
				fmt.Fprint(w, `{"protectionBypass":{"new-secret":{"scope":"shareable-link","createdAt":1000,"createdBy":"user_123","expires":2000000000},"*":{"scope":"alias-protection-override"}}}`)
			})}
			model := shareableTestState()
			model.TeamID = types.StringUnknown()
			model.TTLSeconds = types.Int64Value(600)
			model.RotationID = types.StringValue("2")
			prior := shareableResourceState(t, r, shareableTestState())
			plan := tfsdk.Plan{Schema: prior.Schema}
			if diags := plan.Set(context.Background(), model); diags.HasError() {
				t.Fatal(diags)
			}
			var result tfsdk.State
			if rotate {
				resp := resource.UpdateResponse{State: tfsdk.State{Schema: prior.Schema}}
				r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: prior}, &resp)
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
				result = resp.State
			} else {
				resp := resource.CreateResponse{State: tfsdk.State{Schema: prior.Schema}}
				r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
				result = resp.State
			}
			var got ShareableLinkResourceState
			if diags := result.Get(context.Background(), &got); diags.HasError() {
				t.Fatal(diags)
			}
			if got.Secret.ValueString() != "new-secret" || got.ID.ValueString() != "alias_123" || got.TeamID.ValueString() != "team_123" || got.TTLSeconds.ValueInt64() != 600 {
				t.Fatalf("unexpected result: %v", got)
			}
			if strings.Contains(got.ID.ValueString(), "secret") {
				t.Fatal("secret in public ID")
			}
			if patches != 1 || (!rotate && gets != 1) {
				t.Errorf("gets=%d patches=%d", gets, patches)
			}
		})
	}
}

func TestShareableLinkReadLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		bypasses                 any
		gone, diagnostic, active bool
	}{
		{name: "revoked", bypasses: map[string]any{}, gone: true},
		{name: "permissions omitted", bypasses: nil, diagnostic: true},
		{name: "externally rotated", bypasses: map[string]any{"replacement": map[string]any{"scope": "shareable-link"}}, diagnostic: true},
		{name: "expired", bypasses: map[string]any{"old-secret": map[string]any{"scope": "shareable-link", "createdAt": 1000, "expires": 1}}},
		{name: "never expires", bypasses: map[string]any{"old-secret": map[string]any{"scope": "shareable-link", "createdAt": 1000}}, active: true},
		{name: "wrong scope", bypasses: map[string]any{"old-secret": map[string]any{"scope": "automation-bypass"}}, gone: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &shareableLinkResource{client: shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" || req.URL.Path != "/v4/aliases/alias_123" {
					t.Errorf("refresh must read owning document: %s %s", req.Method, req.URL.Path)
				}
				writeShareableAlias(w, tt.bypasses)
			})}
			state := shareableResourceState(t, r, shareableTestState())
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() != tt.diagnostic {
				t.Fatalf("diagnostics=%v", resp.Diagnostics)
			}
			if resp.State.Raw.IsNull() != tt.gone {
				t.Fatalf("removed=%v", resp.State.Raw.IsNull())
			}
			if !tt.gone && !tt.diagnostic {
				var got ShareableLinkResourceState
				if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
					t.Fatal(diags)
				}
				if got.Active.ValueBool() != tt.active {
					t.Fatalf("active=%v", got.Active)
				}
			}
		})
	}
}

func TestShareableLinkDeleteOnlyRevokesOwnedSecret(t *testing.T) {
	r := &shareableLinkResource{client: shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Revoke struct {
				Secret     string
				Regenerate bool
			}
		}
		json.NewDecoder(req.Body).Decode(&body)
		if req.URL.Path != "/aliases/alias_123/protection-bypass" || body.Revoke.Secret != "old-secret" || body.Revoke.Regenerate {
			t.Error("unsafe revoke")
		}
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":{"code":"not_found","message":"already revoked"}}`)
	})}
	state := shareableResourceState(t, r, shareableTestState())
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

func TestShareableLinkImportAndDataSource(t *testing.T) {
	c := shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			t.Error("read/import mutated alias")
		}
		writeShareableAlias(w, map[string]any{"old-secret": map[string]any{"scope": "shareable-link", "createdAt": 1000999, "expires": 1600}, "*": map[string]any{"scope": "alias-protection-override"}})
	})
	r := &shareableLinkResource{client: c}
	state := shareableResourceState(t, r, shareableTestState())
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: state.Schema}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "team_123/preview.vercel.app"}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got ShareableLinkResourceState
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.TTLSeconds.ValueInt64() != 600 || got.Active.ValueBool() || got.Secret.ValueString() != "old-secret" {
		t.Fatalf("bad import: %v", got)
	}
	d := &shareableLinkDataSource{client: c}
	var schemaResp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: resp.State.Raw}
	// Resource-only attributes must never be decoded by the data source.
	dsState := tfsdk.State{Schema: schemaResp.Schema}
	if diags := dsState.Set(context.Background(), got.ShareableLink); diags.HasError() {
		t.Fatal(diags)
	}
	config.Raw = dsState.Raw
	dsResp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &dsResp)
	if dsResp.Diagnostics.HasError() {
		t.Fatal(dsResp.Diagnostics)
	}
	var dsGot ShareableLink
	if diags := dsResp.State.Get(context.Background(), &dsGot); diags.HasError() {
		t.Fatal(diags)
	}
	if !dsGot.Secret.Equal(got.Secret) || !dsGot.ExpiresAt.Equal(got.ExpiresAt) {
		t.Fatal("data source did not read existing expired link")
	}
}

func TestShareableLinkSensitiveSchemas(t *testing.T) {
	var rs resource.SchemaResponse
	(&shareableLinkResource{}).Schema(context.Background(), resource.SchemaRequest{}, &rs)
	var ds datasource.SchemaResponse
	(&shareableLinkDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &ds)
	for _, key := range []string{"secret", "url"} {
		if !rs.Schema.Attributes[key].(rsschema.StringAttribute).Sensitive || !ds.Schema.Attributes[key].(dsschema.StringAttribute).Sensitive {
			t.Errorf("%s is not sensitive", key)
		}
	}
	if _, ok := ds.Schema.Attributes["ttl_seconds"]; ok {
		t.Fatal("resource-only attribute in data source")
	}
	expires := time.Now().Unix() - 1
	link := shareableLinkState(client.AliasResponse{UID: "alias_123", Alias: "preview.vercel.app"}, "secret", client.ProtectionBypass{Expires: &expires})
	if link.Active.ValueBool() {
		t.Fatal("expired link reported active")
	}
}

func TestShareableLinkCreateDoesNotAdoptExistingLink(t *testing.T) {
	mutations := 0
	r := &shareableLinkResource{client: shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			mutations++
			t.Error("existing link was mutated")
		}
		writeShareableAlias(w, map[string]any{"external-secret": map[string]any{"scope": "shareable-link"}})
	})}
	state := shareableResourceState(t, r, shareableTestState())
	plan := tfsdk.Plan(state)
	resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if !resp.Diagnostics.HasError() || mutations != 0 {
		t.Fatalf("expected import diagnostic without mutation: %v", resp.Diagnostics)
	}
}

func TestShareableLinkOwnershipGuard(t *testing.T) {
	mutations := 0
	r := &shareableLinkResource{client: shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			mutations++
			t.Error("transferred alias was mutated")
		}
		fmt.Fprint(w, `{"uid":"alias_123","alias":"preview.vercel.app","projectId":"other-project","protectionBypass":{"old-secret":{"scope":"shareable-link"}}}`)
	})}
	state := shareableResourceState(t, r, shareableTestState())
	readResp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("read accepted a transferred alias")
	}
	model := shareableTestState()
	model.RotationID = types.StringValue("2")
	plan := tfsdk.Plan{Schema: state.Schema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &updateResp)
	if !updateResp.Diagnostics.HasError() || mutations != 0 {
		t.Fatal("rotation accepted a transferred alias")
	}
}

func TestShareableLinkReadHTTPFailures(t *testing.T) {
	for _, status := range []int{403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := &shareableLinkResource{client: shareableTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"error":{"code":%q,"message":"private response"}}`, map[int]string{403: "forbidden", 404: "not_found"}[status])
			})}
			state := shareableResourceState(t, r, shareableTestState())
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.State.Raw.IsNull() != (status == 404) || resp.Diagnostics.HasError() != (status == 403) {
				t.Fatalf("status %d: state=%v diagnostics=%v", status, resp.State.Raw, resp.Diagnostics)
			}
		})
	}
}
