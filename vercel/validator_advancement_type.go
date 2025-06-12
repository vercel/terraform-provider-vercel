package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Custom validator for advancement_type
type advancementTypeValidator struct{}

func (v advancementTypeValidator) Description(ctx context.Context) string {
	return "advancement_type must be either 'automatic' or 'manual-approval'"
}

func (v advancementTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v advancementTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Get the value of enabled from the parent object
	var enabled types.Bool
	diags := req.Config.GetAttribute(ctx, path.Root("rolling_release").AtName("enabled"), &enabled)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Error validating advancement_type",
			"Could not get enabled value from configuration",
		)
		return
	}

	// Only validate when enabled is true
	if enabled.ValueBool() {
		if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
			resp.Diagnostics.AddError(
				"Invalid advancement_type",
				"advancement_type is required when enabled is true",
			)
			return
		}

		value := req.ConfigValue.ValueString()
		if value != "automatic" && value != "manual-approval" {
			resp.Diagnostics.AddError(
				"Invalid advancement_type",
				fmt.Sprintf("advancement_type must be either 'automatic' or 'manual-approval', got: %s", value),
			)
			return
		}
	}
}
