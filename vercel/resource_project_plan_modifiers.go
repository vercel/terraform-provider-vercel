package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type projectPreviewFeedbackPlanModifier struct {
	otherAttribute string
}

func (projectPreviewFeedbackPlanModifier) Description(context.Context) string {
	return "Preserves observed preview feedback unless its other attribute is configured."
}

func (m projectPreviewFeedbackPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m projectPreviewFeedbackPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	var other types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(m.otherAttribute), &other)...)
	// Both attributes read the same API field. Configuring either can change the
	// other during apply, so its unconfigured alias must remain unknown.
	if resp.Diagnostics.HasError() || !other.IsNull() {
		return
	}
	boolplanmodifier.UseStateForUnknown().PlanModifyBool(ctx, req, resp)
}

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
