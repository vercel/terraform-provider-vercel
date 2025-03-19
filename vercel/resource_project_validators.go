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
	resourceConfig, diags := project.resourceConfig(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if resourceConfig == nil {
		return
	}
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
