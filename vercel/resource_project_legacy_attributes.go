package vercel

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A deprecated project attribute and its modern counterpart are mutually
// exclusive in configuration, but both halves of the pair are Computed and both
// are read back from the same API fields. Without plan modifiers the half that
// is not configured becomes "known after apply" as soon as anything else in the
// project changes, and the stale prior value can be sent back to the API
// instead of the configured one. These plan modifiers make the configured half
// of each pair decide the planned value for both.
var (
	previewCommentsPath          = path.Root("preview_comments")
	enablePreviewFeedbackPath    = path.Root("enable_preview_feedback")
	serverlessFunctionRegionPath = path.Root("serverless_function_region")
)

// mirrorBoolPlanModifier plans an unconfigured boolean attribute as the value
// configured for the attribute it is mutually exclusive with.
type mirrorBoolPlanModifier struct {
	counterpart path.Path
}

func (m mirrorBoolPlanModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Mirrors the configured value of %s when this attribute is not configured.", m.counterpart)
}

func (m mirrorBoolPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m mirrorBoolPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	var counterpart types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, m.counterpart, &counterpart)...)
	if resp.Diagnostics.HasError() || counterpart.IsNull() || counterpart.IsUnknown() {
		return
	}

	resp.PlanValue = counterpart
}

// serverlessFunctionRegionPlanModifier keeps the deprecated
// serverless_function_region null unless it is configured, matching how it is
// read back from the API.
type serverlessFunctionRegionPlanModifier struct{}

func (serverlessFunctionRegionPlanModifier) Description(_ context.Context) string {
	return "Stays null unless this deprecated attribute is configured."
}

func (m serverlessFunctionRegionPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (serverlessFunctionRegionPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	resp.PlanValue = types.StringNull()
}

// functionDefaultRegionsPlanModifier plans the regions implied by a configured
// serverless_function_region. It only applies when the resource config block
// itself is configured, as nested plan modifiers do not run for an omitted
// block.
type functionDefaultRegionsPlanModifier struct{}

func (functionDefaultRegionsPlanModifier) Description(_ context.Context) string {
	return "Mirrors a configured serverless_function_region when this attribute is not configured."
}

func (m functionDefaultRegionsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (functionDefaultRegionsPlanModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	region, diags := configuredServerlessFunctionRegion(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || region == nil {
		return
	}

	resp.PlanValue = functionDefaultRegionsValue(*region)
}

// resourceConfigPlanModifier plans the function default regions implied by a
// configured serverless_function_region, keeping the rest of the omitted block
// at its prior value.
type resourceConfigPlanModifier struct{}

func (resourceConfigPlanModifier) Description(_ context.Context) string {
	return "Mirrors a configured serverless_function_region into the function default regions."
}

func (m resourceConfigPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (resourceConfigPlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !req.ConfigValue.IsNull() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	region, diags := configuredServerlessFunctionRegion(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || region == nil {
		return
	}

	attributes := maps.Clone(req.StateValue.Attributes())
	attributes["function_default_regions"] = functionDefaultRegionsValue(*region)
	resp.PlanValue = types.ObjectValueMust(resourceConfigAttrType.AttrTypes, attributes)
}

func configuredServerlessFunctionRegion(ctx context.Context, config tfsdk.Config) (*string, diag.Diagnostics) {
	var region types.String
	diags := config.GetAttribute(ctx, serverlessFunctionRegionPath, &region)
	if diags.HasError() || region.IsNull() || region.IsUnknown() {
		return nil, diags
	}
	return region.ValueStringPointer(), diags
}

func functionDefaultRegionsValue(region string) types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{types.StringValue(region)})
}
