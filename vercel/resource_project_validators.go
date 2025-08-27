package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ConfigValidator = &fluidComputeBasicCPUValidator{}

type fluidComputeBasicCPUValidator struct{}

func (v *fluidComputeBasicCPUValidator) Description(ctx context.Context) string {
	return "Validates that the CPU type is one of the allowed values for Vercel fluid compute."
}

func (v *fluidComputeBasicCPUValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *fluidComputeBasicCPUValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var project Project
	diags := req.Config.Get(ctx, &project)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If ResourceConfig is unknown (computed), skip validation
	// since we can't determine the configuration's validity during planning.
	if project.ResourceConfig.IsUnknown() || project.ResourceConfig.IsNull() {
		return
	}

	resourceConfig, diags := project.resourceConfig(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if resourceConfig == nil {
		return
	}

	// Note: Due to UnhandledUnknownAsEmpty: true in resourceConfig(),
	// unknown values are converted to zero values (false, "").
	// We can't distinguish between explicitly set zero values and unknown values here.
	// This is acceptable because:
	// 1. If values are truly unknown, they'll be validated at apply time by the API
	// 2. If values are explicitly set to zero values, validation should proceed

	if !resourceConfig.Fluid.ValueBool() {
		return
	}
	if resourceConfig.FunctionDefaultCPUType.ValueString() != "standard_legacy" {
		return
	}

	resp.Diagnostics.AddError(
		"Error validating project fluid compute configuration",
		"Fluid compute is only supported with the standard or performance CPU types.",
	)
}
