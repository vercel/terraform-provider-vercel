package vercel

import (
	"context"
	"encoding/json"
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

func TestStandalonePassportLifecycle(t *testing.T) {
	ctx := context.Background()
	passport := &client.Passport{ConnectorID: "scl_existing", DeploymentType: "all"}
	writes := 0
	missing := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("teamId") != "team_test" {
			t.Errorf("team = %s", req.URL.RawQuery)
		}
		if missing {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"error":{"code":"not_found","message":"Missing project"}}`)
			return
		}
		switch req.Method {
		case "GET":
			if req.URL.Path != "/v10/projects/prj_existing" {
				t.Errorf("GET path = %s", req.URL.Path)
			}
		case "PATCH":
			writes++
			if req.URL.Path != "/v9/projects/prj_existing" {
				t.Errorf("PATCH path = %s", req.URL.Path)
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body["passport"] == nil {
				t.Fatalf("PATCH must change Passport only: %v", body)
			}
			if err := json.Unmarshal(body["passport"], &passport); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "prj_existing", "accountId": "team_test", "passport": passport})
	}))
	defer server.Close()
	r := &projectPassportResource{client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_test"})}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	// Import is read-only and adopts settings, not the project lifecycle.
	imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "prj_existing"}, &imported)
	if imported.Diagnostics.HasError() || writes != 0 {
		t.Fatalf("import: %v, writes: %d", imported.Diagnostics, writes)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: imported.State.Raw}
	plan.SetAttribute(ctx, path.Root("deployment_type"), types.StringValue("preview"))
	updated := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, resource.UpdateRequest{State: imported.State, Plan: plan}, &updated)
	if updated.Diagnostics.HasError() || passport.DeploymentType != "preview" {
		t.Fatalf("update: %v, passport: %#v", updated.Diagnostics, passport)
	}
	// Out-of-band removal is drift: the settings attachment disappears, not the project.
	passport = nil
	read := resource.ReadResponse{State: updated.State}
	r.Read(ctx, resource.ReadRequest{State: updated.State}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("read disabled: %v, %s", read.Diagnostics, read.State.Raw)
	}
	created := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() || passport == nil {
		t.Fatalf("create: %v", created.Diagnostics)
	}
	deleted := resource.DeleteResponse{State: created.State}
	r.Delete(ctx, resource.DeleteRequest{State: created.State}, &deleted)
	if deleted.Diagnostics.HasError() || passport != nil {
		t.Fatalf("delete must disable only Passport: %v", deleted.Diagnostics)
	}
	missing = true
	deleted = resource.DeleteResponse{State: created.State}
	r.Delete(ctx, resource.DeleteRequest{State: created.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatal(deleted.Diagnostics)
	}
	read = resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("missing project: %v", read.Diagnostics)
	}
}

func TestProjectPassportOwnership(t *testing.T) {
	ctx := context.Background()
	project := projectForUpdateRequestTests()
	project.Passport = passportState(&client.Passport{ConnectorID: "scl_observed", DeploymentType: "all"})
	project.ManagePassport = types.BoolValue(false)
	request, diags := project.toUpdateProjectRequest(ctx, "before-rename")
	if diags.HasError() {
		t.Fatal(diags)
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(body, &fields)
	if _, ok := fields["passport"]; ok {
		t.Fatalf("unrelated project update wrote delegated Passport: %s", body)
	}
	result, err := convertResponseToProject(ctx, client.ProjectResponse{ID: "prj_existing", Passport: &client.Passport{ConnectorID: "scl_observed", DeploymentType: "all"}}, project, nil)
	if err != nil || !result.ManagePassport.Equal(types.BoolValue(false)) {
		t.Fatalf("ownership lost on refresh: %v", err)
	}
	var schemaResp resource.SchemaResponse
	newProjectResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if d := state.Set(ctx, &project); d.HasError() {
		t.Fatal(d)
	}
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw}
	validation := resource.ValidateConfigResponse{}
	(&projectPassportOwnershipValidator{}).ValidateResource(ctx, resource.ValidateConfigRequest{Config: config}, &validation)
	if !validation.Diagnostics.HasError() {
		t.Fatal("inline passport with manage_passport=false must be rejected")
	}
}
