package vercel_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

func TestProjectEnvironmentVariableWriteOnlyImport(t *testing.T) {
	for _, tc := range []struct {
		name, id  string
		writeOnly bool
	}{
		{"legacy", "team_test/prj_test/env_test", false},
		{"team_write_only", "write-only:team_test/prj_test/env_test", true},
		{"default_team_write_only", "write-only:prj_test/env_test", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			stored := "fake-secret-before-import"
			var patches []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/v10/projects/prj_test/env/env_test" {
					t.Errorf("unexpected path: %s", req.URL.Path)
					w.WriteHeader(400)
					return
				}
				switch req.Method {
				case http.MethodGet:
				case http.MethodPatch:
					var patch map[string]any
					if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					patches = append(patches, patch)
					if v, ok := patch["value"].(string); ok {
						stored = v
					}
				default:
					t.Errorf("unexpected mutation: %s", req.Method)
					w.WriteHeader(400)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": "env_test", "key": "TEST_SECRET", "value": stored, "type": "encrypted", "target": []string{"development", "preview"}, "comment": ""})
			}))
			defer server.Close()
			var r resource.Resource
			for _, factory := range vercel.New().Resources(ctx) {
				candidate := factory()
				var metadata resource.MetadataResponse
				candidate.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "vercel"}, &metadata)
				if metadata.TypeName == "vercel_project_environment_variable" {
					r = candidate
					break
				}
			}
			if r == nil {
				t.Fatal("resource not found")
			}
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			var configured resource.ConfigureResponse
			r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: client.New("fake-token").WithTeam(client.Team{ID: "team_test"}).WithBaseURL(server.URL)}, &configured)
			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: tc.id}, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			var model vercel.ProjectEnvironmentVariable
			if d := imported.State.Get(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			if !tc.writeOnly {
				if model.Value.ValueString() != stored {
					t.Fatal("legacy import no longer retains readable value")
				}
				return
			}
			assertSecretAbsent := func(state tfsdk.State) {
				t.Helper()
				var value vercel.ProjectEnvironmentVariable
				if d := state.Get(ctx, &value); d.HasError() {
					t.Fatal(d)
				}
				if !value.Value.IsNull() || !value.ValueWO.IsNull() {
					t.Fatal("credential was persisted in state")
				}
				if strings.Contains(state.Raw.String(), "fake-secret-") {
					t.Fatal("serialized state contains fake credential")
				}
				if value.ID.ValueString() != "env_test" || value.TeamID.ValueString() != "team_test" {
					t.Fatal("import changed resource identity")
				}
			}
			assertSecretAbsent(imported.State)
			current := imported.State
			refresh := func() {
				t.Helper()
				response := resource.ReadResponse{State: tfsdk.State{Schema: schema.Schema}}
				r.Read(ctx, resource.ReadRequest{State: current}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				assertSecretAbsent(response.State)
				current = response.State
			}
			refresh()
			update := func(version int64, secret string) {
				t.Helper()
				var m vercel.ProjectEnvironmentVariable
				if d := current.Get(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				m.ValueWOVersion = types.Int64Value(version)
				plan := tfsdk.Plan{Schema: schema.Schema}
				if d := plan.Set(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				if strings.Contains(plan.Raw.String(), "fake-secret-") {
					t.Fatal("plan contains fake credential")
				}
				m.ValueWO = types.StringValue(secret)
				configState := tfsdk.State{Schema: schema.Schema}
				if d := configState.Set(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				response := resource.UpdateResponse{State: tfsdk.State{Schema: schema.Schema}}
				r.Update(ctx, resource.UpdateRequest{State: current, Plan: plan, Config: tfsdk.Config{Schema: schema.Schema, Raw: configState.Raw}}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				assertSecretAbsent(response.State)
				current = response.State
			}
			update(1, "fake-secret-before-import")
			if stored != "fake-secret-before-import" {
				t.Fatal("initial adoption changed the value")
			}
			refresh()
			update(1, "fake-secret-not-selected")
			if _, ok := patches[len(patches)-1]["value"]; ok {
				t.Fatal("unchanged version sent a credential")
			}
			update(2, "fake-secret-after-rotation")
			if stored != "fake-secret-after-rotation" {
				t.Fatal("version increment did not rotate the credential")
			}
			refresh()
		})
	}
}

type localEnvImportProvider struct {
	provider.Provider
	url string
}

func (p localEnvImportProvider) Configure(_ context.Context, _ provider.ConfigureRequest, response *provider.ConfigureResponse) {
	c := client.New("fake-token").WithTeam(client.Team{ID: "team_test"}).WithBaseURL(p.url)
	response.ResourceData = c
	response.DataSourceData = c
}

type noImportCredentialPlanCheck struct{}

func (noImportCredentialPlanCheck) CheckPlan(_ context.Context, request plancheck.CheckPlanRequest, response *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(request.Plan)
	if err != nil {
		response.Error = err
		return
	}
	if strings.Contains(string(encoded), "fake-secret-") {
		response.Error = fmt.Errorf("saved plan contains a fake credential")
	}
}

func TestProjectEnvironmentVariableWriteOnlyImportTerraform(t *testing.T) {
	var mu sync.Mutex
	stored := "fake-secret-initial"
	t.Setenv("TF_VAR_import_secret", stored)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodDelete && req.URL.Path == "/v8/projects/prj_test/env/env_test" {
			w.WriteHeader(200)
			return
		}
		if req.URL.Path != "/v10/projects/prj_test/env/env_test" || req.URL.Query().Get("teamId") != "team_test" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			w.WriteHeader(400)
			return
		}
		if req.Method == http.MethodPatch {
			var payload map[string]any
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if v, ok := payload["value"].(string); ok {
				stored = v
			}
		} else if req.Method != http.MethodGet {
			t.Errorf("unexpected mutation: %s", req.Method)
			w.WriteHeader(400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "env_test", "key": "TEST_SECRET", "value": stored, "type": "encrypted", "target": []string{"development", "preview"}, "comment": ""})
	}))
	defer server.Close()
	config := func(version int, includeImport bool) string {
		c := fmt.Sprintf(`
provider "vercel" {}
variable "import_secret" {
 type = string
 sensitive = true
 ephemeral = true
}
resource "vercel_project_environment_variable" "test" {
 project_id = "prj_test"
 team_id = "team_test"
 key = "TEST_SECRET"
 target = ["development", "preview"]
 sensitive = false
 value_wo = var.import_secret
 value_wo_version = %d
}
`, version)
		if includeImport {
			c += `import {
 to = vercel_project_environment_variable.test
 id = "write-only:team_test/prj_test/env_test"
}
`
		}
		return c
	}
	check := func(want string) tfresource.TestCheckFunc {
		return func(s *terraform.State) error {
			encoded, err := json.Marshal(s)
			if err != nil {
				return err
			}
			if strings.Contains(string(encoded), "fake-secret-") {
				return fmt.Errorf("saved state contains a fake credential")
			}
			mu.Lock()
			defer mu.Unlock()
			if stored != want {
				return fmt.Errorf("unexpected remote credential after apply")
			}
			r := s.RootModule().Resources["vercel_project_environment_variable.test"]
			if r == nil || r.Primary.ID != "env_test" {
				return fmt.Errorf("import did not preserve the entry ID")
			}
			return nil
		}
	}
	checks := tfresource.ConfigPlanChecks{
		PreApply:             []plancheck.PlanCheck{noImportCredentialPlanCheck{}},
		PostApplyPreRefresh:  []plancheck.PlanCheck{noImportCredentialPlanCheck{}},
		PostApplyPostRefresh: []plancheck.PlanCheck{noImportCredentialPlanCheck{}},
	}
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"vercel": providerserver.NewProtocol6WithError(localEnvImportProvider{Provider: vercel.New(), url: server.URL}),
		},
		Steps: []tfresource.TestStep{
			{Config: config(1, true), ConfigPlanChecks: checks, Check: check("fake-secret-initial")},
			{Config: config(1, false), ConfigPlanChecks: checks, Check: check("fake-secret-initial")},
			{PreConfig: func() { t.Setenv("TF_VAR_import_secret", "fake-secret-rotated") }, Config: config(2, false), ConfigPlanChecks: checks, Check: check("fake-secret-rotated")},
			{RefreshState: true, Check: check("fake-secret-rotated")},
		},
	})
}
