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
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func projectTestIdentity(t *testing.T, r *projectResource) *tfsdk.ResourceIdentity {
	t.Helper()
	var schema resource.IdentitySchemaResponse
	r.IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &schema)
	return &tfsdk.ResourceIdentity{Schema: schema.IdentitySchema, Raw: tftypes.NewValue(schema.IdentitySchema.Type().TerraformType(context.Background()), nil)}
}

func assertProjectIdentity(t *testing.T, value *tfsdk.ResourceIdentity, team types.String) {
	t.Helper()
	var identity projectIdentity
	if diags := value.Get(context.Background(), &identity); diags.HasError() {
		t.Fatal(diags)
	}
	if identity.ProjectID.ValueString() != "prj_123" || !identity.TeamID.Equal(team) {
		t.Fatalf("identity = %#v, want team %s", identity, team)
	}
}

func TestProjectIdentityImportAndLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, team, account string
		identity            bool
	}{
		{name: "identity with team", team: "team_123", account: "team_123", identity: true},
		{name: "identity provider default", account: "team_default", identity: true},
		{name: "legacy import null identity", account: "team_default"},
		{name: "personal identity", account: "user_123", identity: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectedRequestTeam := tc.team
			if expectedRequestTeam == "" && tc.account != "user_123" {
				expectedRequestTeam = "team_default"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Query().Get("teamId") != expectedRequestTeam {
					t.Errorf("teamId = %q, want %q", req.URL.Query().Get("teamId"), expectedRequestTeam)
				}
				name := "web"
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/prj_123":
				case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123":
					name = "renamed"
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				fmt.Fprintf(w, `{"id":"prj_123","name":%q,"accountId":%q,"nodeVersion":"22.x"}`, name, tc.account)
			}))
			defer server.Close()
			c := client.New("test").WithBaseURL(server.URL)
			if tc.account != "user_123" {
				c.WithTeam(client.Team{ID: "team_default"})
			}
			r := &projectResource{client: c}
			ctx := context.Background()
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			req := resource.ImportStateRequest{ID: "prj_123", Identity: projectTestIdentity(t, r)}
			if tc.identity {
				req.ID = ""
				if diags := req.Identity.Set(ctx, projectIdentity{ProjectID: types.StringValue("prj_123"), TeamID: toTeamID(tc.team)}); diags.HasError() {
					t.Fatal(diags)
				}
			}
			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema}, Identity: projectTestIdentity(t, r)}
			r.ImportState(ctx, req, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			expectedTeam := types.StringValue(tc.account)
			if tc.account == "user_123" {
				expectedTeam = types.StringNull()
			}
			assertProjectIdentity(t, imported.Identity, expectedTeam)
			read := resource.ReadResponse{State: tfsdk.State{Schema: schema.Schema}, Identity: projectTestIdentity(t, r)}
			r.Read(ctx, resource.ReadRequest{State: imported.State}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			assertProjectIdentity(t, read.Identity, expectedTeam)
			plan := tfsdk.Plan{Schema: schema.Schema, Raw: read.State.Raw}
			if diags := plan.SetAttribute(ctx, path.Root("name"), types.StringValue("renamed")); diags.HasError() {
				t.Fatal(diags)
			}
			updated := resource.UpdateResponse{State: tfsdk.State{Schema: schema.Schema}, Identity: projectTestIdentity(t, r)}
			r.Update(ctx, resource.UpdateRequest{Plan: plan, State: read.State}, &updated)
			if updated.Diagnostics.HasError() {
				t.Fatal(updated.Diagnostics)
			}
			assertProjectIdentity(t, updated.Identity, expectedTeam)
		})
	}
}

func TestProjectCreateSetsIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/v8/projects" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
			return
		}
		fmt.Fprint(w, `{"id":"prj_123","name":"web","accountId":"team_123","nodeVersion":"22.x"}`)
	}))
	defer server.Close()
	r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
	ctx := context.Background()
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)

	plan := tfsdk.Plan{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)}
	for name, value := range map[string]types.String{"id": types.StringUnknown(), "name": types.StringValue("web"), "team_id": types.StringValue("team_123")} {
		if diags := plan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
			t.Fatal(diags)
		}
	}

	resp := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}, Identity: projectTestIdentity(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	assertProjectIdentity(t, resp.Identity, types.StringValue("team_123"))
}
