package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestImportStateInvalidIDReturnsBeforeClientCall(t *testing.T) {
	ctx := context.Background()
	req := resource.ImportStateRequest{ID: "too/many/id/parts"}

	tests := []struct {
		name string
		run  func(*resource.ImportStateResponse)
	}{
		{name: "access group", run: func(resp *resource.ImportStateResponse) { (&accessGroupResource{}).ImportState(ctx, req, resp) }},
		{name: "access group project", run: func(resp *resource.ImportStateResponse) { (&accessGroupProjectResource{}).ImportState(ctx, req, resp) }},
		{name: "attack challenge mode", run: func(resp *resource.ImportStateResponse) { (&attackChallengeModeResource{}).ImportState(ctx, req, resp) }},
		{name: "custom environment", run: func(resp *resource.ImportStateResponse) { (&customEnvironmentResource{}).ImportState(ctx, req, resp) }},
		{name: "dns record", run: func(resp *resource.ImportStateResponse) { (&dnsRecordResource{}).ImportState(ctx, req, resp) }},
		{name: "edge config", run: func(resp *resource.ImportStateResponse) { (&edgeConfigResource{}).ImportState(ctx, req, resp) }},
		{name: "edge config item", run: func(resp *resource.ImportStateResponse) { (&edgeConfigItemResource{}).ImportState(ctx, req, resp) }},
		{name: "edge config schema", run: func(resp *resource.ImportStateResponse) { (&edgeConfigSchemaResource{}).ImportState(ctx, req, resp) }},
		{name: "edge config token", run: func(resp *resource.ImportStateResponse) { (&edgeConfigTokenResource{}).ImportState(ctx, req, resp) }},
		{name: "firewall config", run: func(resp *resource.ImportStateResponse) { (&firewallConfigResource{}).ImportState(ctx, req, resp) }},
		{name: "log drain", run: func(resp *resource.ImportStateResponse) { (&logDrainResource{}).ImportState(ctx, req, resp) }},
		{name: "microfrontend group", run: func(resp *resource.ImportStateResponse) { (&microfrontendGroupResource{}).ImportState(ctx, req, resp) }},
		{name: "microfrontend group membership", run: func(resp *resource.ImportStateResponse) {
			(&microfrontendGroupMembershipResource{}).ImportState(ctx, req, resp)
		}},
		{name: "project", run: func(resp *resource.ImportStateResponse) { (&projectResource{}).ImportState(ctx, req, resp) }},
		{name: "project deployment retention", run: func(resp *resource.ImportStateResponse) {
			(&projectDeploymentRetentionResource{}).ImportState(ctx, req, resp)
		}},
		{name: "project domain", run: func(resp *resource.ImportStateResponse) { (&projectDomainResource{}).ImportState(ctx, req, resp) }},
		{name: "project environment variable", run: func(resp *resource.ImportStateResponse) {
			(&projectEnvironmentVariableResource{}).ImportState(ctx, req, resp)
		}},
		{name: "project rolling release", run: func(resp *resource.ImportStateResponse) {
			(&projectRollingReleaseResource{}).ImportState(ctx, req, resp)
		}},
		{name: "shared environment variable", run: func(resp *resource.ImportStateResponse) {
			(&sharedEnvironmentVariableResource{}).ImportState(ctx, req, resp)
		}},
		{name: "team member", run: func(resp *resource.ImportStateResponse) { (&teamMemberResource{}).ImportState(ctx, req, resp) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resource.ImportStateResponse{}
			tt.run(resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("ImportState() returned no diagnostics for invalid ID")
			}
		})
	}
}
