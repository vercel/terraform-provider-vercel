package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestTeamConfigDefaultBuildMachineTypeAcceptsBasic(t *testing.T) {
	res := newTeamConfigResource()
	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attribute := resp.Schema.Attributes["default_build_machine_type"].(resourceschema.StringAttribute)
	for _, v := range attribute.Validators {
		validationResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("default_build_machine_type"),
			ConfigValue: types.StringValue("basic"),
		}, validationResp)
		if validationResp.Diagnostics.HasError() {
			t.Fatalf("basic should be valid: %s", validationResp.Diagnostics)
		}
	}
}

func TestTeamConfigBuildMachineDefaultMapping(t *testing.T) {
	config := TeamConfig{
		ID:                      types.StringValue("team_1"),
		Slug:                    types.StringNull(),
		RemoteCaching:           types.ObjectNull(remoteCachingAttrTypes),
		Saml:                    types.ObjectNull(samlAttrTypes),
		DefaultBuildMachineType: types.StringValue("basic"),
	}
	request, diags := config.toUpdateTeamRequest(context.Background(), "", types.StringNull())
	if diags.HasError() {
		t.Fatal(diags)
	}
	if request.ResourceConfig == nil || request.ResourceConfig.BuildMachine == nil || request.ResourceConfig.BuildMachine.Default == nil {
		t.Fatal("expected build machine default in request")
	}
	if got := *request.ResourceConfig.BuildMachine.Default; got != "basic" {
		t.Fatalf("buildMachine.default = %q, want basic", got)
	}

	basic := "basic"
	state, diags := convertResponseToTeamConfig(context.Background(), client.Team{
		ResourceConfig: &client.TeamResourceConfig{
			BuildMachine: &client.TeamBuildMachine{Default: &basic},
		},
	}, types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatal(diags)
	}
	if got := state.DefaultBuildMachineType.ValueString(); got != "basic" {
		t.Fatalf("default_build_machine_type = %q, want basic", got)
	}
}
