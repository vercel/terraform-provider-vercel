package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ConfigValidator = &fluidComputeBasicCPUValidator{}
var _ resource.ConfigValidator = &automationBypassSecretsValidator{}

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

type automationBypassSecretsValidator struct{}

func (v *automationBypassSecretsValidator) Description(ctx context.Context) string {
	return "Validates multi-secret protection bypass configuration."
}

func (v *automationBypassSecretsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *automationBypassSecretsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var project Project
	diags := req.Config.Get(ctx, &project)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !project.hasConfiguredProtectionBypassForAutomationSecrets() {
		return
	}

	if !project.ProtectionBypassForAutomation.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("protection_bypass_for_automation_secrets"),
			"Invalid Configuration",
			"protection_bypass_for_automation_secrets is only allowed when protection_bypass_for_automation is true.",
		)
	}

	if !project.ProtectionBypassForAutomationSecret.IsNull() && !project.ProtectionBypassForAutomationSecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("protection_bypass_for_automation_secrets"),
			"Invalid Configuration",
			"protection_bypass_for_automation_secrets cannot be used together with protection_bypass_for_automation_secret.",
		)
	}

	secrets, diags := project.protectionBypassForAutomationSecrets(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envVarCount := 0
	for _, secret := range secrets {
		if secret.IsEnvVar.ValueBool() {
			envVarCount++
		}
	}

	if envVarCount == 1 {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("protection_bypass_for_automation_secrets"),
		"Invalid Configuration",
		"Exactly one protection_bypass_for_automation_secrets entry must set is_env_var to true.",
	)
}
