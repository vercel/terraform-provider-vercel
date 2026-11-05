package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectDataSourceLookupValidation(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.DataSourceSchemas["vercel_project"].ValueType()
	for _, tc := range []struct {
		name       string
		project    any
		id         any
		wantErrors bool
	}{
		{name: "name", project: "existing-project"},
		{name: "id", id: "prj_existing"},
		{name: "unknown name", project: tftypes.UnknownValue},
		{name: "unknown id", id: tftypes.UnknownValue},
		{name: "neither", wantErrors: true},
		{name: "both", project: "existing-project", id: "prj_existing", wantErrors: true},
		{name: "empty id", id: "", wantErrors: true},
		{name: "invalid name", project: "invalid_name", wantErrors: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := projectLookupConfig(typ, tc.project, tc.id, nil)
			value, err := tfprotov6.NewDynamicValue(typ, config)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.ValidateDataResourceConfig(ctx, &tfprotov6.ValidateDataResourceConfigRequest{TypeName: "vercel_project", Config: &value})
			if err != nil {
				t.Fatal(err)
			}
			hasErrors := false
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					hasErrors = true
				}
			}
			if hasErrors != tc.wantErrors {
				t.Fatalf("validation errors = %v, want %v: %v", hasErrors, tc.wantErrors, response.Diagnostics)
			}
		})
	}
}

func TestProjectDataSourceLookupRead(t *testing.T) {
	for _, tc := range []struct {
		name       string
		project    any
		id         any
		team       any
		identifier string
		wantTeam   string
	}{
		{name: "name", project: "existing-project", identifier: "existing-project", wantTeam: "team_default"},
		{name: "id after rename", id: "prj_existing", identifier: "prj_existing", wantTeam: "team_default"},
		{name: "explicit team", id: "prj_existing", team: "team_override", identifier: "prj_existing", wantTeam: "team_override"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if req.Method != http.MethodGet || req.URL.Query().Get("teamId") != tc.wantTeam {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				switch req.URL.Path {
				case "/v10/projects/" + tc.identifier:
					name := "existing-project"
					if tc.id != nil {
						name = "renamed-project"
					}
					fmt.Fprintf(w, `{"id":"prj_existing","name":%q,"accountId":%q}`, name, tc.wantTeam)
				case "/v8/projects/prj_existing/env":
					fmt.Fprint(w, `{"envs":[]}`)
				default:
					t.Errorf("unexpected path: %s", req.URL.Path)
					http.NotFound(w, req)
				}
			}))
			defer server.Close()
			d := &projectDataSource{client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_default"})}
			ctx := context.Background()
			var schemaResponse datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
			typ := schemaResponse.Schema.Type().TerraformType(ctx)
			config := tfsdk.Config{Schema: schemaResponse.Schema, Raw: projectLookupConfig(typ, tc.project, tc.id, tc.team)}
			response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			d.Read(ctx, datasource.ReadRequest{Config: config}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var result ProjectDataSource
			if diags := response.State.Get(ctx, &result); diags.HasError() {
				t.Fatal(diags)
			}
			wantName := "existing-project"
			if tc.id != nil {
				wantName = "renamed-project"
			}
			if calls != 2 || result.ID.ValueString() != "prj_existing" || result.Name.ValueString() != wantName || result.TeamID.ValueString() != tc.wantTeam {
				t.Fatalf("read returned id=%s name=%s team=%s after %d calls", result.ID, result.Name, result.TeamID, calls)
			}
		})
	}
}

func projectLookupConfig(typ tftypes.Type, name, id, team any) tftypes.Value {
	fields := make(map[string]tftypes.Value)
	for key, fieldType := range typ.(tftypes.Object).AttributeTypes {
		fields[key] = tftypes.NewValue(fieldType, nil)
	}
	fields["name"] = tftypes.NewValue(tftypes.String, name)
	fields["id"] = tftypes.NewValue(tftypes.String, id)
	fields["team_id"] = tftypes.NewValue(tftypes.String, team)
	return tftypes.NewValue(typ, fields)
}
