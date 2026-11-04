package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// projectPassportOwnershipValidator prevents inline and standalone settings from sharing an owner.
type projectPassportOwnershipValidator struct{}

func (*projectPassportOwnershipValidator) Description(context.Context) string {
	return "Omit inline passport when manage_passport is false."
}
func (v *projectPassportOwnershipValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (*projectPassportOwnershipValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var managed types.Bool
	var passport types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("manage_passport"), &managed)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("passport"), &passport)...)
	if managed.Equal(types.BoolValue(false)) && !passport.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("passport"), "Conflicting Passport ownership", "Omit passport when manage_passport is false. Manage it with vercel_project_passport instead.")
	}
}

type projectProtectionOwnershipValidator struct{}

func (*projectProtectionOwnershipValidator) Description(context.Context) string {
	return "Omit inline deployment protection when manage_deployment_protection is false."
}
func (v *projectProtectionOwnershipValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (*projectProtectionOwnershipValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var managed types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("manage_deployment_protection"), &managed)...)
	if !managed.Equal(types.BoolValue(false)) {
		return
	}
	for _, name := range []string{"vercel_authentication", "password_protection", "trusted_ips"} {
		var value types.Object
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &value)...)
		if !value.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Conflicting deployment protection ownership", "Omit this attribute when manage_deployment_protection is false. Use vercel_project_deployment_protection instead.")
		}
	}
}
