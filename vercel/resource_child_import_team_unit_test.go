package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestChildImportTeamOwnership(t *testing.T) {
	for _, op := range []struct {
		name, importID, parentPath, parentJSON, childPath, childJSON string
		resource                                                     func(*client.Client) resource.ResourceWithImportState
	}{
		{"edge config item", "ec_1/key", "/v1/edge-config/ec_1", `{"ownerId":"team_owner"}`, "/v1/edge-config/ec_1/item/key", `{"edgeConfigId":"ec_1","key":"key","value":"hello"}`, func(c *client.Client) resource.ResourceWithImportState { return &edgeConfigItemResource{client: c} }},
		{"blob connection", "store_1/conn_1", "/v1/storage/stores/store_1", `{"store":{"ownerId":"team_owner"}}`, "/v1/storage/stores/store_1/connections", `{"connections":[{"id":"conn_1","projectId":"prj_1","envVarEnvironments":["production"]}]}`, func(c *client.Client) resource.ResourceWithImportState {
			return &blobProjectConnectionResource{client: c}
		}},
	} {
		t.Run(op.name, func(t *testing.T) {
			for _, fail := range []bool{false, true} {
				t.Run(fmt.Sprintf("owner_missing_%t", fail), func(t *testing.T) {
					childCalled := false
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case op.parentPath:
							if fail {
								fmt.Fprint(w, `{}`)
							} else {
								fmt.Fprint(w, op.parentJSON)
							}
						case op.childPath:
							childCalled = true
							if r.URL.Query().Get("teamId") != "team_owner" {
								t.Errorf("child request lacks recovered team: %s", r.URL)
							}
							fmt.Fprint(w, op.childJSON)
						default:
							t.Errorf("unexpected path %s", r.URL.Path)
							w.WriteHeader(404)
						}
					}))
					defer server.Close()
					res := op.resource(client.New("test").WithBaseURL(server.URL))
					schema := resource.SchemaResponse{}
					res.Schema(context.Background(), resource.SchemaRequest{}, &schema)
					resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema}}
					res.ImportState(context.Background(), resource.ImportStateRequest{ID: op.importID}, &resp)
					if fail {
						if !resp.Diagnostics.HasError() {
							t.Fatal("expected ownership diagnostic")
						}
						if childCalled {
							t.Fatal("child queried after ownership could not be established")
						}
						return
					}
					if resp.Diagnostics.HasError() {
						t.Fatal(resp.Diagnostics)
					}
					var team types.String
					diags := resp.State.GetAttribute(context.Background(), path.Root("team_id"), &team)
					if diags.HasError() {
						t.Fatal(diags)
					}
					if team.ValueString() != "team_owner" {
						t.Errorf("imported team=%s", team)
					}
				})
			}
		})
	}
}
