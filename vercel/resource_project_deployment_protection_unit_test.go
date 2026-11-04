package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestStandaloneDeploymentProtectionTerraformLifecycle(t *testing.T) {
	var mu sync.Mutex
	// An unrelated password already exists; a partial attachment must leave it alone.
	project := client.ProjectResponse{ID: "prj_existing", TeamID: "team_123", AccountID: "team_123", PasswordProtection: &client.PasswordProtection{DeploymentType: "all"}}
	var patches []map[string]json.RawMessage
	var passwords []string
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
			patches = append(patches, fields)
			for key, value := range fields {
				switch key {
				case "ssoProtection":
					json.Unmarshal(value, &project.VercelAuthentication)
				case "passwordProtection":
					var pp *client.PasswordProtectionWithPassword
					json.Unmarshal(value, &pp)
					if pp == nil {
						project.PasswordProtection = nil
					} else {
						project.PasswordProtection = &client.PasswordProtection{DeploymentType: pp.DeploymentType}
						if pp.Password != "" {
							passwords = append(passwords, pp.Password)
						}
					}
				case "trustedIps":
					json.Unmarshal(value, &project.TrustedIps)
					if project.TrustedIps != nil {
						project.TrustedIps.ProtectionMode = "additional"
					}
				default:
					t.Errorf("unrelated setting written: %s", key)
				}
			}
		default:
			t.Errorf("must never create/delete the project: %s", req.Method)
		}
		json.NewEncoder(w).Encode(project)
	}))
	defer server.Close()
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{Provider: New(), client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"})})}
	config := func(auth, ip, password string) string {
		fields := ""
		if auth != "" {
			fields += fmt.Sprintf("vercel_authentication = { deployment_type = %q }\n", auth)
		}
		if ip != "" {
			fields += fmt.Sprintf("trusted_ips = { deployment_type = \"all\", addresses = [{value = %q}] }\n", ip)
		}
		fields += password
		return `resource "vercel_project_deployment_protection" "settings" {
 project_id = "prj_existing"
 ` + fields + "\n}"
	}
	passwordConfig := func(secret string, version int) string {
		return fmt.Sprintf("password_protection = { deployment_type = \"preview\", password_wo = %q, password_wo_version = %d }", secret, version)
	}
	checkUnownedPassword := func(_ *terraform.State) error {
		mu.Lock()
		defer mu.Unlock()
		if project.PasswordProtection == nil || project.PasswordProtection.DeploymentType != "all" {
			return fmt.Errorf("unowned password protection was changed")
		}
		return nil
	}
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: config("preview", "203.0.113.0/24", ""), Check: tfresource.ComposeTestCheckFunc(tfresource.TestCheckResourceAttr("vercel_project_deployment_protection.settings", "team_id", "team_123"), checkUnownedPassword)},
		{Config: config("all", "203.0.113.0/24", ""), Check: checkUnownedPassword},
		{Config: config("all", "203.0.113.0/24", passwordConfig("first-password", 1)), Check: tfresource.TestCheckNoResourceAttr("vercel_project_deployment_protection.settings", "password_protection.password_wo")},
		{Config: config("all", "203.0.113.0/24", passwordConfig("no-version-bump", 1)), PlanOnly: true},
		{Config: config("all", "203.0.113.0/24", passwordConfig("second-password", 2))},
		{Config: config("all", "203.0.113.0/24", "")},
		{Config: config("", "203.0.113.0/24", "")},
		{Config: config("", "198.51.100.0/24", "")},
		{Config: config("", "", "")},
		{Config: config("all", "", "")},
	}})
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(passwords, []string{"first-password", "second-password"}) {
		t.Fatalf("password writes = %v", passwords)
	}
	expected := [][]string{{"ssoProtection", "trustedIps"}, {"ssoProtection"}, {"passwordProtection"}, {"passwordProtection"}, {"passwordProtection"}, {"ssoProtection"}, {"trustedIps"}, {"trustedIps"}, {"ssoProtection"}, {"ssoProtection"}}
	var got [][]string
	for _, patch := range patches {
		keys := make([]string, 0, len(patch))
		for key := range patch {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		got = append(got, keys)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("PATCH ownership: got %v, want %v", got, expected)
	}
	if project.VercelAuthentication != nil || project.TrustedIps != nil || project.PasswordProtection != nil {
		t.Fatalf("owned methods not disabled on removal/destroy: %#v", project)
	}
}

func TestStandaloneDeploymentProtectionImport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			t.Errorf("import must be read-only: %s", req.Method)
		}
		fmt.Fprint(w, `{"id":"prj_existing","accountId":"team_123","ssoProtection":{"deploymentType":"all"},"passwordProtection":{"deploymentType":"preview"},"trustedIps":{"deploymentType":"all","addresses":[{"value":"203.0.113.1"}],"protectionMode":"exclusive"}}`)
	}))
	defer server.Close()
	ctx := context.Background()
	r := &projectDeploymentProtectionResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "team_123/prj_existing"}, &imported)
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	var model projectDeploymentProtectionModel
	if d := imported.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if model.Authentication.IsNull() || model.Password.IsNull() || model.TrustedIPs.IsNull() {
		t.Fatalf("import did not adopt all enabled methods: %#v", model)
	}
	if !model.Password.Attributes()["password_wo"].IsNull() || !model.Password.Attributes()["password_wo_version"].IsNull() {
		t.Fatal("import must not invent credentials or a rotation counter")
	}
	// Unowned methods must not appear during ordinary refresh.
	partial := emptyProjectDeploymentProtection()
	partial.Authentication = model.Authentication
	out, err := r.client.GetProject(ctx, "prj_existing", "team_123")
	if err != nil {
		t.Fatal(err)
	}
	if d := partial.read(ctx, out, false); d.HasError() {
		t.Fatal(d)
	}
	if !partial.Password.IsNull() || !partial.TrustedIPs.IsNull() {
		t.Fatal("refresh adopted unrelated protection")
	}
}

func TestProjectDeploymentProtectionOwnership(t *testing.T) {
	ctx := context.Background()
	project := projectForUpdateRequestTests()
	project.ManageDeploymentProtection = types.BoolValue(false)
	project.VercelAuthentication = types.ObjectValueMust(vercelAuthenticationAttrType.AttrTypes, map[string]attr.Value{"deployment_type": types.StringValue("all_deployments")})
	project.PasswordProtection = types.ObjectValueMust(passwordProtectionWithPasswordAttrType.AttrTypes, map[string]attr.Value{"deployment_type": types.StringValue("all_deployments"), "password": types.StringValue("ignored")})
	request, d := project.toUpdateProjectRequest(ctx, "before-rename")
	if d.HasError() {
		t.Fatal(d)
	}
	payload, _ := json.Marshal(request)
	for _, field := range []string{"ssoProtection", "passwordProtection", "trustedIps"} {
		if strings.Contains(string(payload), `"`+field+`"`) {
			t.Fatalf("project wrote delegated %s: %s", field, payload)
		}
	}
	create, d := project.toCreateProjectRequest(ctx, nil)
	if d.HasError() {
		t.Fatal(d)
	}
	payload, _ = json.Marshal(create)
	if strings.Contains(string(payload), `"ssoProtection"`) {
		t.Fatalf("project creation wrote delegated authentication: %s", payload)
	}
	result, err := convertResponseToProject(ctx, client.ProjectResponse{ID: "prj_existing", VercelAuthentication: &client.VercelAuthentication{DeploymentType: "all"}, PasswordProtection: &client.PasswordProtection{DeploymentType: "all"}, TrustedIps: &client.TrustedIps{DeploymentType: "all"}}, project, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.PasswordProtection.IsNull() || !result.TrustedIps.IsNull() || !result.ManageDeploymentProtection.Equal(types.BoolValue(false)) {
		t.Fatal("delegation lost on refresh")
	}
}

func TestStandaloneDeploymentProtectionRequiresPasswordToEnable(t *testing.T) {
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			patches++
			t.Errorf("must reject missing password before PATCH")
		}
		fmt.Fprint(w, `{"id":"prj_existing","accountId":"team_123"}`)
	}))
	defer server.Close()
	r := &projectDeploymentProtectionResource{client: client.New("test").WithBaseURL(server.URL)}
	model := emptyProjectDeploymentProtection()
	model.ProjectID = types.StringValue("prj_existing")
	model.Password = types.ObjectValueMust(standalonePasswordTypes, map[string]attr.Value{"deployment_type": types.StringValue("preview"), "password_wo": types.StringNull(), "password_wo_version": types.Int64Null()})
	diags := r.apply(context.Background(), &model, emptyProjectDeploymentProtection(), model)
	if !diags.HasError() || patches != 0 {
		t.Fatalf("missing password was accepted: %v", diags)
	}
}

func TestStandaloneDeploymentProtectionReadFailures(t *testing.T) {
	for _, status := range []int{404, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"error":{"code":"test_error","message":"failure"}}`)
			}))
			defer server.Close()
			ctx := context.Background()
			r := &projectDeploymentProtectionResource{client: client.New("test").WithBaseURL(server.URL)}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			model := emptyProjectDeploymentProtection()
			model.ID, model.ProjectID = types.StringValue("prj_existing"), types.StringValue("prj_existing")
			model.Authentication = types.ObjectValueMust(standaloneAuthenticationTypes, map[string]attr.Value{"deployment_type": types.StringValue("all")})
			state := tfsdk.State{Schema: schemaResp.Schema}
			if d := state.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			read := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if status == 404 {
				if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
					t.Fatalf("missing project: %v", read.Diagnostics)
				}
			} else if !read.Diagnostics.HasError() || !read.State.Raw.Equal(state.Raw) {
				t.Fatalf("read failure lost ownership: %v", read.Diagnostics)
			}
			deleted := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
			if deleted.Diagnostics.HasError() != (status != 404) {
				t.Fatalf("delete diagnostics = %v", deleted.Diagnostics)
			}
		})
	}
}

func TestStandaloneDeploymentProtectionRejectsUnreadableAllowlist(t *testing.T) {
	model := emptyProjectDeploymentProtection()
	diags := model.read(context.Background(), client.ProjectResponse{ID: "prj_existing", TrustedIps: &client.TrustedIps{DeploymentType: "all"}}, true)
	if !diags.HasError() {
		t.Fatal("partial API response must not be imported as an empty allowlist")
	}
}

func TestDelegatedAuthenticationPlan(t *testing.T) {
	ctx := context.Background()
	observed := types.ObjectValueMust(vercelAuthenticationAttrType.AttrTypes, map[string]attr.Value{"deployment_type": types.StringValue("all_deployments")})
	unknown := types.ObjectUnknown(vercelAuthenticationAttrType.AttrTypes)
	var schemaResp resource.SchemaResponse
	newProjectResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	for _, tc := range []struct {
		name       string
		managed    types.Bool
		plan, want types.Object
	}{
		{"managed update preserves scope", types.BoolValue(true), unknown, observed},
		{"delegated update leaves scope unknown", types.BoolValue(false), unknown, unknown},
		{"unknown ownership leaves scope unknown", types.BoolUnknown(), unknown, unknown},
		{"delegated unchanged plan stays known", types.BoolValue(false), observed, observed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			fields := make(map[string]tftypes.Value)
			for name, fieldType := range objectType.AttributeTypes {
				fields[name] = tftypes.NewValue(fieldType, nil)
			}
			ownership, err := tc.managed.ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			fields["manage_deployment_protection"] = ownership
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, fields)}
			observedRaw, err := observed.ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Raw: observedRaw}

			req := planmodifier.ObjectRequest{Config: config, State: state, Path: path.Root("vercel_authentication"), ConfigValue: types.ObjectNull(vercelAuthenticationAttrType.AttrTypes), StateValue: observed, PlanValue: tc.plan}
			resp := planmodifier.ObjectResponse{PlanValue: tc.plan}
			modifier := schemaResp.Schema.Attributes["vercel_authentication"].(interface{ ObjectPlanModifiers() []planmodifier.Object }).ObjectPlanModifiers()[0]
			modifier.PlanModifyObject(ctx, req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Fatalf("plan = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}
