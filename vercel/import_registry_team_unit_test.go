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

func TestRegistryAndMembershipImportOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		make     func(*client.Client) resource.ResourceWithImportState
	}{
		{"repository", "prj_123/example", func(c *client.Client) resource.ResourceWithImportState { return &vcrRepositoryResource{client: c} }},
		{"permission", "prj_123/example/team_grantee", func(c *client.Client) resource.ResourceWithImportState {
			return &vcrRepositoryPermissionResource{client: c}
		}},
		{"microfrontend membership", "mfe_123/prj_123", func(c *client.Client) resource.ResourceWithImportState {
			return &microfrontendGroupMembershipResource{client: c}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, scope := range []string{"discovered", "provider", "explicit", "lookup failure"} {
				t.Run(scope, func(t *testing.T) {
					discovery := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						if r.URL.Path == "/v10/projects/prj_123" && r.URL.Query().Get("teamId") == "" {
							discovery++
							if scope == "lookup failure" {
								w.WriteHeader(403)
								fmt.Fprint(w, `{"error":{"code":"forbidden","message":"Forbidden"}}`)
								return
							}
						} else if r.URL.Path != "/v2/teams/team_owner" && r.URL.Query().Get("teamId") != "team_owner" {
							t.Errorf("unscoped child request: %s", r.URL)
						}
						switch r.URL.Path {
						case "/v10/projects/prj_123":
							fmt.Fprint(w, `{"id":"prj_123","name":"project","accountId":"team_owner"}`)
						case "/v2/teams/team_owner":
							fmt.Fprint(w, `{"id":"team_owner","slug":"owner"}`)
						case "/v1/vcr/repository/example":
							fmt.Fprint(w, `{"repository":{"id":"repo_123","name":"example","projectId":"prj_123"}}`)
						case "/v1/vcr/repository/example/permissions":
							fmt.Fprint(w, `{"permissions":[{"repositoryId":"repo_123","teamId":"team_grantee","teamSlug":"grantee"}]}`)
						case "/v1/microfrontends/groups":
							fmt.Fprint(w, `{"groups":[{"group":{"id":"mfe_123","name":"group","slug":"group"},"projects":[{"id":"prj_123","microfrontends":{"enabled":true,"isDefaultApp":true}}]}]}`)
						default:
							t.Errorf("unexpected URL %s", r.URL)
							http.NotFound(w, r)
						}
					}))
					defer server.Close()
					c := client.New("token").WithBaseURL(server.URL)
					id := tc.id
					if scope == "provider" {
						c.WithTeam(client.Team{ID: "team_owner", Slug: "owner"})
					}
					if scope == "explicit" {
						id = "team_owner/" + id
					}
					res := tc.make(c)
					ctx := context.Background()
					sr := resource.SchemaResponse{}
					res.Schema(ctx, resource.SchemaRequest{}, &sr)
					resp := resource.ImportStateResponse{State: tfsdk.State{Schema: sr.Schema}}
					res.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
					if scope == "lookup failure" {
						if !resp.Diagnostics.HasError() {
							t.Fatal("expected ownership error")
						}
						return
					}
					if resp.Diagnostics.HasError() {
						t.Fatal(resp.Diagnostics)
					}
					var team types.String
					if d := resp.State.GetAttribute(ctx, path.Root("team_id"), &team); d.HasError() {
						t.Fatal(d)
					}
					if team.ValueString() != "team_owner" {
						t.Fatalf("team = %s", team)
					}
					wantDiscovery := 0
					if scope == "discovered" {
						wantDiscovery = 1
					}
					if discovery != wantDiscovery {
						t.Errorf("discovery reads = %d, want %d", discovery, wantDiscovery)
					}
				})
			}
		})
	}
}

func TestMicrofrontendGroupImportRequiresTeam(t *testing.T) {
	ctx := context.Background()
	res := microfrontendGroupResource{client: client.New("token")}
	resp := resource.ImportStateResponse{}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "mfe_123"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected missing team error")
	}
}
