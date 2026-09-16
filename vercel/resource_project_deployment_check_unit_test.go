package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectDeploymentCheckRoundTrip(t *testing.T) {
	ctx := context.Background()
	state, diags := projectDeploymentCheckFromClient(ctx, client.ProjectDeploymentCheck{
		ID: "check_123", ProjectID: "prj_123", Name: "E2E", Requires: "deployment-url", Blocks: "deployment-promotion",
		Targets: []string{"production"}, Timeout: 300, Source: client.ProjectDeploymentCheckSource{Kind: "git-provider", Provider: "github", ExternalCheckName: "e2e"},
	}, types.StringValue("prj_123"), types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatalf("diagnostics = %v", diags)
	}
	request, diags := state.createRequest(ctx)
	if diags.HasError() {
		t.Fatalf("diagnostics = %v", diags)
	}
	if request.ProjectID != "prj_123" || request.Source == nil || request.Source.ExternalCheckName != "e2e" || request.Targets == nil || len(*request.Targets) != 1 {
		t.Fatalf("request = %#v", request)
	}
}

func TestProjectDeploymentCheckUpdateOnlySendsChanges(t *testing.T) {
	ctx := context.Background()
	source := types.ObjectValueMust(projectDeploymentCheckSourceAttrTypes, map[string]attr.Value{
		"kind": types.StringValue("git-provider"), "external_check_name": types.StringValue("e2e"), "provider": types.StringValue("github"),
		"webhook_id": types.StringNull(), "external_resource_id": types.StringNull(), "integration_id": types.StringNull(),
		"integration_configuration_id": types.StringNull(), "resource_id": types.StringNull(), "job_name": types.StringNull(), "origin": types.StringNull(), "sub_kind": types.StringNull(),
	})
	state := ProjectDeploymentCheck{ID: types.StringValue("check_123"), ProjectID: types.StringValue("prj_123"), TeamID: types.StringValue("team_123"), Name: types.StringValue("E2E"), Requires: types.StringValue("deployment-url"), IsRerequestable: types.BoolValue(false), Blocks: types.StringValue("deployment-promotion"), Targets: types.SetValueMust(types.StringType, []attr.Value{}), Timeout: types.Int64Value(300), Source: source}
	plan := state
	plan.ID = types.StringUnknown()
	plan.Name = types.StringValue("E2E tests")
	request, diags := plan.updateRequest(ctx, state)
	if diags.HasError() {
		t.Fatalf("diagnostics = %v", diags)
	}
	if request.ID != "check_123" || request.Name == nil || *request.Name != "E2E tests" || request.Blocks != nil || request.Targets != nil {
		t.Fatalf("request = %#v", request)
	}
}
