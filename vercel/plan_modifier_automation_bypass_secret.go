package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// useStateForAutomationBypassSecret preserves the previously generated/stored
// protection_bypass_for_automation_secret across plans when the user has not
// supplied a new secret in config and bypass is not being disabled. Without
// this, the attribute is recomputed to Unknown on every plan which produces a
// spurious diff and, during apply, causes the Update path to issue a
// revoke-only request to the API (see issue #473).
func useStateForAutomationBypassSecret() planmodifier.String {
	return automationBypassSecretPlanModifier{}
}

type automationBypassSecretPlanModifier struct{}

func (m automationBypassSecretPlanModifier) Description(_ context.Context) string {
	return "Preserve the stored protection_bypass_for_automation_secret when it isn't being rotated or revoked."
}

func (m automationBypassSecretPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m automationBypassSecretPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}

	planBypass := types.Bool{}
	diags := req.Plan.GetAttribute(ctx, path.Root("protection_bypass_for_automation"), &planBypass)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !planBypass.ValueBool() {
		resp.PlanValue = types.StringUnknown()
		return
	}

	resp.PlanValue = req.StateValue
}
