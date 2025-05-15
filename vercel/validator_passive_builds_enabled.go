package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// passiveBuildsEnabledValidator is a validator that ensures builds_enabled is false if passive is true.
type passiveBuildsEnabledValidator struct{}

func (v passiveBuildsEnabledValidator) Description(_ context.Context) string {
	return "builds_enabled cannot be true if passive is true"
}

func (v passiveBuildsEnabledValidator) MarkdownDescription(_ context.Context) string {
	return "builds_enabled cannot be true if passive is true"
}

func (v passiveBuildsEnabledValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	// Iterate through each element in the set
	var networks []ProjectSecureComputeNetwork
	diags := req.ConfigValue.ElementsAs(ctx, &networks, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	for _, network := range networks {
		// Check the condition
		if network.Passive.ValueBool() && network.BuildsEnabled.ValueBool() {
			resp.Diagnostics.AddError(
				"Invalid Secure Compute Network Configuration",
				"builds_enabled cannot be `true` if passive is `true`.",
			)
		}
	}
}

// NewPassiveBuildsEnabledValidator returns a validator that ensures builds_enabled is false if passive is true.
func NewPassiveBuildsEnabledValidator() validator.Set {
	return passiveBuildsEnabledValidator{}
}
