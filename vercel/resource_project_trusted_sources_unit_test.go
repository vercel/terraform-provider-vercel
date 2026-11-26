package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestStandaloneTrustedSourcesTerraformLifecycle(t *testing.T) {
	var mu sync.Mutex
	// Existing protection must never be written by this resource.
	project := client.ProjectResponse{ID: "prj_existing", TeamID: "team_123", AccountID: "team_123", VercelAuthentication: &client.VercelAuthentication{DeploymentType: "all"}}
	var patches []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if req.URL.Query().Get("teamId") != "team_123" {
			t.Errorf("team = %s", req.URL.RawQuery)
		}
		switch req.Method {
		case "GET":
			if req.URL.Path != "/v10/projects/prj_existing" {
				t.Errorf("GET path = %s", req.URL.Path)
			}
		case "PATCH":
			if req.URL.Path != "/v9/projects/prj_existing" {
				t.Errorf("PATCH path = %s", req.URL.Path)
			}
			var fields map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&fields); err != nil {
				t.Error(err)
				return
			}
			for key := range fields {
				if key != "trustedSources" {
					t.Errorf("unrelated setting written: %s", key)
				}
			}
			raw := fields["trustedSources"]
			patches = append(patches, string(raw))
			// Like the API, omitted or null updates preserve the stored CI flag.
			var stored *bool
			if project.TrustedSources != nil {
				stored = project.TrustedSources.EnableVercelCiSameRepository
			}
			var next *client.ProjectTrustedSources
			if err := json.Unmarshal(raw, &next); err != nil {
				t.Error(err)
			}
			if next == nil {
				project.TrustedSources = nil
				if stored != nil {
					project.TrustedSources = &client.TrustedSources{EnableVercelCiSameRepository: stored}
				}
				break
			}
			if next.EnableVercelCiSameRepository == nil {
				next.EnableVercelCiSameRepository = stored
			}
			project.TrustedSources = &client.TrustedSources{Projects: next.Projects, OIDCProviders: next.OIDCProviders, EnableVercelCiSameRepository: next.EnableVercelCiSameRepository}
		default:
			t.Errorf("must never create/delete the project: %s", req.Method)
		}
		json.NewEncoder(w).Encode(project)
	}))
	defer server.Close()
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{Provider: New(), client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"})})}
	const projects = `projects = [{ project_id = "prj_frontend", label = "Frontend" }]`
	const externalSources = `external_sources = [{
  issuer = "https://token.actions.githubusercontent.com"
  label  = "GitHub Actions"
  to     = { slugs = ["preview"] }
  claims = { aud = ["https://vercel.com/acme"], repository = ["acme/app"] }
 }]`
	config := func(fields ...string) string {
		body := ""
		for _, field := range fields {
			body += field + "\n"
		}
		return `resource "vercel_project_trusted_sources" "settings" {
 project_id = "prj_existing"
 ` + body + "}"
	}
	checkProtectionUntouched := func(_ *terraform.State) error {
		mu.Lock()
		defer mu.Unlock()
		if project.VercelAuthentication == nil || project.VercelAuthentication.DeploymentType != "all" {
			return fmt.Errorf("unrelated protection was changed")
		}
		return nil
	}
	name := "vercel_project_trusted_sources.settings"
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: config(projects, externalSources), Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckResourceAttr(name, "team_id", "team_123"),
			tfresource.TestCheckResourceAttr(name, "projects.#", "1"),
			tfresource.TestCheckResourceAttr(name, "external_sources.#", "1"),
			tfresource.TestCheckResourceAttr(name, "enable_vercel_ci_same_repository", "true"),
			checkProtectionUntouched,
		)},
		{Config: config(projects, `enable_vercel_ci_same_repository = false`), Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckNoResourceAttr(name, "external_sources"),
			tfresource.TestCheckResourceAttr(name, "enable_vercel_ci_same_repository", "false"),
		)},
		// Omitting the flag keeps the stored value and plans no change.
		{Config: config(projects), PlanOnly: true},
		{ResourceName: name, ImportState: true, ImportStateVerify: true, ImportStateId: "team_123/prj_existing"},
		{Config: config(externalSources), Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckNoResourceAttr(name, "projects"),
			tfresource.TestCheckResourceAttr(name, "external_sources.#", "1"),
			tfresource.TestCheckResourceAttr(name, "enable_vercel_ci_same_repository", "false"),
			checkProtectionUntouched,
		)},
	}})
	mu.Lock()
	defer mu.Unlock()
	oidc := `"oidcProviders":{"https://token.actions.githubusercontent.com":[{"to":{"slugs":["preview"]},"label":"GitHub Actions","claims":{"aud":["https://vercel.com/acme"],"repository":["acme/app"]}}]}`
	expected := []string{
		`{"projects":{"prj_frontend":{"label":"Frontend"}},` + oidc + `}`,
		`{"projects":{"prj_frontend":{"label":"Frontend"}},"oidcProviders":{},"enableVercelCiSameRepository":false}`,
		`{"projects":{},` + oidc + `}`,
		`null`,
	}
	if !reflect.DeepEqual(patches, expected) {
		t.Fatalf("PATCH bodies:\n got %v\nwant %v", patches, expected)
	}
	if project.TrustedSources == nil || len(project.TrustedSources.Projects) != 0 || len(project.TrustedSources.OIDCProviders) != 0 {
		t.Fatalf("destroy did not clear Trusted Sources: %#v", project.TrustedSources)
	}
	if project.VercelAuthentication == nil {
		t.Fatal("destroy changed unrelated protection")
	}
}

func TestStandaloneTrustedSourcesImportIsReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			t.Errorf("import must be read-only: %s", req.Method)
		}
		fmt.Fprint(w, `{"id":"prj_existing","accountId":"team_123","trustedSources":{"projects":{"prj_a":{}},"enableVercelCiSameRepository":false}}`)
	}))
	defer server.Close()
	ctx := context.Background()
	r := &projectTrustedSourcesResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "team_123/prj_existing"}, &imported)
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	var model projectTrustedSourcesModel
	if d := imported.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if len(model.Projects.Elements()) != 1 || !model.ExternalSources.IsNull() || !model.EnableVercelCiSameRepository.Equal(types.BoolValue(false)) {
		t.Fatalf("import did not adopt the live configuration: %#v", model)
	}
}

func TestProjectTrustedSourcesOwnership(t *testing.T) {
	ctx := context.Background()
	observed := client.TrustedSources{Projects: map[string]client.TrustedSourcesProject{"prj_a": {}}}
	project := projectForUpdateRequestTests()
	project.ManageTrustedSources = types.BoolValue(false)
	project.Passport = types.ObjectNull(passportAttrTypes)
	project.TrustedSources = trustedSourcesObjectFromResponse(observed)
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
	if _, ok := fields["trustedSources"]; ok {
		t.Fatalf("project update wrote delegated Trusted Sources: %s", body)
	}
	result, err := convertResponseToProject(ctx, client.ProjectResponse{ID: "prj_existing", TrustedSources: &observed}, project, nil)
	if err != nil || !result.ManageTrustedSources.Equal(types.BoolValue(false)) {
		t.Fatalf("ownership lost on refresh: %v", err)
	}
	if !result.TrustedSources.IsNull() {
		t.Fatal("refresh adopted delegated Trusted Sources, which would plan to clear them")
	}
	var schemaResp resource.SchemaResponse
	newProjectResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if d := state.Set(ctx, &project); d.HasError() {
		t.Fatal(d)
	}
	validation := resource.ValidateConfigResponse{}
	(&projectTrustedSourcesOwnershipValidator{}).ValidateResource(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw}}, &validation)
	if !validation.Diagnostics.HasError() {
		t.Fatal("inline trusted_sources with manage_trusted_sources=false must be rejected")
	}

	project.ManageTrustedSources = types.BoolValue(true)
	request, diags = project.toUpdateProjectRequest(ctx, "before-rename")
	if diags.HasError() {
		t.Fatal(diags)
	}
	body, _ = json.Marshal(request)
	fields = nil
	json.Unmarshal(body, &fields)
	if _, ok := fields["trustedSources"]; !ok {
		t.Fatalf("managed project update omitted Trusted Sources: %s", body)
	}
}
