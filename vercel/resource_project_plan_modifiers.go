package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type projectAuthenticationPlanModifier struct{}

func (projectAuthenticationPlanModifier) Description(context.Context) string {
	return "Preserves observed authentication settings unless their management is delegated."
}

func (m projectAuthenticationPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (projectAuthenticationPlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	var managed types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("manage_deployment_protection"), &managed)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A standalone resource can change authentication during the same apply.
	// Keep the framework's unknown value instead of promising the previous scope.
	if managed.IsUnknown() || managed.Equal(types.BoolValue(false)) {
		return
	}
	objectplanmodifier.UseNonNullStateForUnknown().PlanModifyObject(ctx, req, resp)
}
