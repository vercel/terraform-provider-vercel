package vercel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestTeamDefaultDeploymentProtection(t *testing.T) {
	for _, tc := range []struct {
		name, response, want string
		absent               bool
	}{
		{name: "absent", response: `{}`, absent: true},
		{name: "password only", response: `{"defaultDeploymentProtection":{"passwordProtection":{"deploymentType":"all"}}}`, absent: true},
		{name: "disabled", response: `{"defaultDeploymentProtection":{"ssoProtection":null}}`, want: "none"},
		{name: "all", response: `{"defaultDeploymentProtection":{"ssoProtection":{"deploymentType":"all"}}}`, want: "all_deployments"},
		{name: "standard", response: `{"defaultDeploymentProtection":{"ssoProtection":{"deploymentType":"all_except_custom_domains"}}}`, want: "standard_protection_new"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var team client.Team
			if err := json.Unmarshal([]byte(tc.response), &team); err != nil {
				t.Fatal(err)
			}
			config, diags := convertResponseToTeamConfig(context.Background(), team, types.MapNull(types.StringType))
			if diags.HasError() {
				t.Fatal(diags)
			}
			request, diags := config.toUpdateTeamRequest(context.Background(), "", types.StringNull())
			if diags.HasError() {
				t.Fatal(diags)
			}
			if tc.absent {
				if !config.DefaultDeploymentProtection.IsNull() || request.DefaultDeploymentProtection != nil {
					t.Fatal("absent authentication must remain unmanaged")
				}
				return
			}
			var defaults DefaultDeploymentProtection
			if diags := config.DefaultDeploymentProtection.As(context.Background(), &defaults, basetypes.ObjectAsOptions{}); diags.HasError() {
				t.Fatal(diags)
			}
			got := defaults.VercelAuthentication.Attributes()["deployment_type"].(types.String).ValueString()
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
			encoded, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]json.RawMessage
			if err = json.Unmarshal(encoded, &body); err != nil {
				t.Fatal(err)
			}
			var protection map[string]json.RawMessage
			if err = json.Unmarshal(body["defaultDeploymentProtection"], &protection); err != nil {
				t.Fatal(err)
			}
			if len(protection) != 1 {
				t.Fatalf("must only patch authentication: %s", encoded)
			}
			if tc.want == "none" && string(protection["ssoProtection"]) != "null" {
				t.Fatalf("disable must send explicit null: %s", encoded)
			}
		})
	}
}

func TestProjectAuthenticationInheritsOnCreate(t *testing.T) {
	for _, authentication := range []types.Object{
		types.ObjectNull(vercelAuthenticationAttrType.AttrTypes),
		types.ObjectUnknown(vercelAuthenticationAttrType.AttrTypes),
	} {
		project := Project{VercelAuthentication: authentication}
		va, diags := project.vercelAuthentication(context.Background())
		if diags.HasError() {
			t.Fatal(diags)
		}
		request := client.CreateProjectRequest{VercelAuthentication: va.toVercelAuthentication()}
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		if _, ok := fields["ssoProtection"]; ok {
			t.Fatalf("omitted authentication must inherit, got %s", body)
		}
	}
	project := Project{VercelAuthentication: types.ObjectValueMust(vercelAuthenticationAttrType.AttrTypes, map[string]attr.Value{"deployment_type": types.StringValue("none")})}
	va, diags := project.vercelAuthentication(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	body, err := json.Marshal(client.CreateProjectRequest{VercelAuthentication: va.toVercelAuthentication()})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["ssoProtection"]) != "null" {
		t.Fatalf("explicit none must disable: %s", body)
	}
}
