package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestTeamConfigSparseUpdate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		changes map[string]attr.Value
		want    string
	}{
		{name: "description only", changes: map[string]attr.Value{"description": types.StringValue("Updated")}, want: `{"description":"Updated"}`},
		{name: "false is a write", changes: map[string]attr.Value{"hide_ip_addresses": types.BoolValue(false)}, want: `{"hideIpAddresses":false}`},
		{name: "SAML enforcement only", changes: map[string]attr.Value{"saml.enforced": types.BoolValue(false)}, want: `{"saml":{"enforced":false}}`},
		{name: "SAML roles only", changes: map[string]attr.Value{"saml.roles": types.MapValueMust(samlRoleType, map[string]attr.Value{"group": types.ObjectValueMust(samlRoleAttrType, map[string]attr.Value{"role": types.StringValue("DEVELOPER"), "access_group_id": types.StringNull()})})}, want: `{"saml":{"roles":{"group":"DEVELOPER"}}}`},
		{name: "clear SAML roles", changes: map[string]attr.Value{"saml.roles": types.MapValueMust(samlRoleType, map[string]attr.Value{})}, want: `{"saml":{"roles":{}}}`},
		{name: "disable Passport", changes: map[string]attr.Value{"default_passport": passportState(nil)}, want: `{"defaultPassport":null}`},
		{name: "unknown SAML role omits whole mapping", changes: map[string]attr.Value{"description": types.StringValue("Updated"), "saml.roles": types.MapValueMust(samlRoleType, map[string]attr.Value{"group": types.ObjectValueMust(samlRoleAttrType, map[string]attr.Value{"role": types.StringUnknown(), "access_group_id": types.StringNull()})})}, want: `{"description":"Updated"}`},
		{name: "unknown SAML child is omitted", changes: map[string]attr.Value{"description": types.StringValue("Updated"), "saml.enforced": types.BoolUnknown()}, want: `{"description":"Updated"}`},
		{name: "unknown is omitted", changes: map[string]attr.Value{"description": types.StringValue("Updated"), "enable_preview_feedback": types.StringUnknown(), "hide_ip_addresses": types.BoolUnknown()}, want: `{"description":"Updated"}`},
		{name: "null is omitted", changes: map[string]attr.Value{"description": types.StringValue("Updated"), "enable_preview_feedback": types.StringNull()}, want: `{"description":"Updated"}`},
		{name: "unchanged uses GET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var patches []map[string]any
			gets := 0
			team := sparseTeamAPIResponse()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if req.URL.Path != "/v2/teams/team_test" {
					t.Errorf("unexpected path: %s", req.URL)
					http.NotFound(w, req)
					return
				}
				switch req.Method {
				case http.MethodGet:
					gets++
				case http.MethodPatch:
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					patches = append(patches, body)
					for key, value := range body {
						if key == "saml" {
							for name, setting := range value.(map[string]any) {
								team["saml"].(map[string]any)[name] = setting
							}
						} else {
							team[key] = value
						}
					}
				default:
					t.Errorf("unexpected method: %s", req.Method)
				}
				if err := json.NewEncoder(w).Encode(team); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			r := &teamConfigResource{client: client.New("test").WithBaseURL(server.URL)}
			ctx := context.Background()
			var schemaResponse resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "team_test"}, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: imported.State.Raw}
			for name, value := range tc.changes {
				p := path.Root(name)
				if name == "saml.enforced" {
					p = path.Root("saml").AtName("enforced")
				} else if name == "saml.roles" {
					p = path.Root("saml").AtName("roles")
				}
				if diags := plan.SetAttribute(ctx, p, value); diags.HasError() {
					t.Fatal(diags)
				}
			}
			response := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Update(ctx, resource.UpdateRequest{State: imported.State, Plan: plan}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			mu.Lock()
			defer mu.Unlock()
			if tc.want == "" {
				if len(patches) != 0 || gets != 2 {
					t.Fatalf("no-op sent %d PATCH and %d GET requests, want 0 PATCH and import/update GETs", len(patches), gets)
				}
				return
			}
			var want map[string]any
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			if len(patches) != 1 || !reflect.DeepEqual(patches[0], want) {
				t.Fatalf("PATCHes = %#v, want only %#v", patches, want)
			}
		})
	}
}

func TestTeamConfigSparseUpdateLifecycle(t *testing.T) {
	for _, mode := range []string{"concurrent SAML edit", "failed atomic readback"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			team := sparseTeamAPIResponse()
			patches := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if req.Method == http.MethodPatch {
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					patches++
					if patches == 2 {
						if mode == "concurrent SAML edit" {
							if len(body) != 1 || body["description"] != "Updated" {
								t.Errorf("description update resent unrelated settings: %#v", body)
							}
							// A dashboard edit after refresh must survive the unrelated PATCH.
							team["saml"].(map[string]any)["enforced"] = false
						} else {
							passport, ok := body["defaultPassport"].(map[string]any)
							if !ok || passport["deploymentType"] != "preview" {
								t.Errorf("missing requested Passport change: %#v", body)
							}
							// The server accepts the new connector but fails to keep the supplied scope.
							team["defaultPassport"] = map[string]any{"connectorId": "scl_new", "deploymentType": "all"}
						}
					}
					for key, value := range body {
						if mode == "failed atomic readback" && patches == 2 && key == "defaultPassport" {
							continue
						}
						team[key] = value
					}
				} else if req.Method != http.MethodGet {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				if err := json.NewEncoder(w).Encode(team); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{Provider: New(), client: client.New("test").WithBaseURL(server.URL)})}
			config := func(description, connector string) string {
				passport := "lifecycle { ignore_changes = [saml] }"
				if mode == "failed atomic readback" {
					passport = fmt.Sprintf(`default_passport = { connector_id = %q, deployment_type = "preview" }`, connector)
				}
				return fmt.Sprintf("resource \"vercel_team_config\" \"test\" {\nid = \"team_test\"\ndescription = %q\n%s\n}\n", description, passport)
			}
			second := tfresource.TestStep{Config: config("Updated", "scl_new")}
			if mode == "failed atomic readback" {
				second.ExpectError = regexp.MustCompile("Provider produced inconsistent result after apply")
			} else {
				second.Check = tfresource.TestCheckResourceAttr("vercel_team_config.test", "saml.enforced", "true")
			}
			steps := []tfresource.TestStep{{Config: config("Initial", "scl_test")}, second}
			if mode == "concurrent SAML edit" {
				steps = append(steps, tfresource.TestStep{RefreshState: true, Check: tfresource.TestCheckResourceAttr("vercel_team_config.test", "saml.enforced", "false")})
			}
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: steps})
			mu.Lock()
			defer mu.Unlock()
			if patches != 2 {
				t.Errorf("PATCH count = %d, want create and update only", patches)
			}
			if mode == "concurrent SAML edit" && team["saml"].(map[string]any)["enforced"] != false {
				t.Fatal("unrelated update overwrote the concurrent SAML edit")
			}
		})
	}
}

func TestTeamConfigCreatePreservesUnconfiguredSamlRoles(t *testing.T) {
	for _, tc := range []struct {
		name  string
		roles types.Map
	}{
		{name: "unconfigured roles", roles: types.MapUnknown(samlRoleType)},
		{name: "partially unknown mapping", roles: types.MapValueMust(samlRoleType, map[string]attr.Value{
			"group":      types.ObjectValueMust(samlRoleAttrType, map[string]attr.Value{"role": types.StringValue("OWNER"), "access_group_id": types.StringNull()}),
			"unresolved": types.ObjectValueMust(samlRoleAttrType, map[string]attr.Value{"role": types.StringUnknown(), "access_group_id": types.StringNull()}),
		})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			team := sparseTeamAPIResponse()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPatch || req.URL.Path != "/v2/teams/team_test" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				want := map[string]any{"saml": map[string]any{"enforced": false}}
				if !reflect.DeepEqual(body, want) {
					t.Errorf("Create PATCH = %#v, want only SAML enforcement", body)
				}
				team["saml"].(map[string]any)["enforced"] = false
				if err := json.NewEncoder(w).Encode(team); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			r := &teamConfigResource{client: client.New("test").WithBaseURL(server.URL)}
			var schemaResponse resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			typ := schemaResponse.Schema.Type().TerraformType(ctx).(tftypes.Object)
			fields := make(map[string]tftypes.Value, len(typ.AttributeTypes))
			for name, attributeType := range typ.AttributeTypes {
				var value any
				if schemaResponse.Schema.Attributes[name].IsComputed() {
					value = tftypes.UnknownValue
				}
				fields[name] = tftypes.NewValue(attributeType, value)
			}
			fields["id"] = tftypes.NewValue(tftypes.String, "team_test")
			plannedRoles, err := tc.roles.ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			fields["saml"] = tftypes.NewValue(typ.AttributeTypes["saml"], map[string]tftypes.Value{
				"enforced": tftypes.NewValue(tftypes.Bool, false),
				"roles":    plannedRoles,
			})
			plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(typ, fields)}
			response := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var roles types.Map
			if diags := response.State.GetAttribute(ctx, path.Root("saml").AtName("roles"), &roles); diags.HasError() {
				t.Fatal(diags)
			}
			if len(roles.Elements()) != 1 || !roles.Elements()["group"].(types.Object).Attributes()["role"].Equal(types.StringValue("OWNER")) {
				t.Fatalf("Create did not retain inherited SAML roles: %s", roles)
			}
		})
	}
}

func sparseTeamAPIResponse() map[string]any {
	return map[string]any{
		"id": "team_test", "name": "Example", "slug": "example", "description": "Initial", "emailDomain": "example.com", "inviteCode": "invite",
		"saml":          map[string]any{"enforced": true, "roles": map[string]any{"group": "OWNER"}},
		"remoteCaching": map[string]any{"enabled": true}, "hideIpAddresses": true, "hideIpAddressesInLogDrains": true,
		"enablePreviewFeedback": "on", "enableProductionFeedback": "off", "sensitiveEnvironmentVariablePolicy": "on", "disjunctiveProductionSecretPolicy": "off",
		"previewDeploymentSuffix": "preview.example.com", "defaultPassport": map[string]any{"connectorId": "scl_test", "deploymentType": "preview"},
		"defaultDeploymentProtection": map[string]any{"ssoProtection": map[string]any{"deploymentType": "all"}},
		"resourceConfig":              map[string]any{"buildMachine": map[string]any{"default": "basic"}},
	}
}
