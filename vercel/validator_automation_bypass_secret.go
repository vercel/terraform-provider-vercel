package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.String = validateAutomationBypassSecretConditional{}

func validateAutomationBypassSecret() validateAutomationBypassSecretConditional {
	return validateAutomationBypassSecretConditional{}
}

type validateAutomationBypassSecretConditional struct{}

func (v validateAutomationBypassSecretConditional) Description(ctx context.Context) string {
	return "protection_bypass_for_automation_secret is only allowed when protection_bypass_for_automation is true"
}

func (v validateAutomationBypassSecretConditional) MarkdownDescription(ctx context.Context) string {
	return "protection_bypass_for_automation_secret is only allowed when protection_bypass_for_automation is true"
}

func (v validateAutomationBypassSecretConditional) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	enabled := types.Bool{}
	if diags := req.Config.GetAttribute(ctx, path.Root("protection_bypass_for_automation"), &enabled); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if !enabled.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Configuration",
			"protection_bypass_for_automation_secret is not allowed unless protection_bypass_for_automation is true.",
		)
	}
}
