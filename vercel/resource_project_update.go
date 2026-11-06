package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// A nil outer pointer omits a setting; a nil inner pointer explicitly clears it.
func projectSetting[T any](value *T) **T { return &value }

func intPointer(value int) *int { return &value }

func projectSettingChanged(ctx context.Context, update *settingsUpdate, p path.Path, plan, state attr.Value) bool {
	if !update.changed(p, plan, state) {
		return false
	}
	value, err := plan.ToTerraformValue(ctx)
	return err == nil && value.IsFullyKnown()
}

func (p *Project) changedProjectSettings(ctx context.Context, state Project) (client.UpdateProjectRequest, settingsUpdate, diag.Diagnostics) {
	request, diags := p.toUpdateProjectRequest(ctx, state.Name.ValueString())
	var update settingsUpdate
	if diags.HasError() {
		return client.UpdateProjectRequest{}, update, diags
	}
	if !projectSettingChanged(ctx, &update, path.Root("passport"), p.Passport, state.Passport) {
		request.Passport = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("build_command"), p.BuildCommand, state.BuildCommand) {
		request.BuildCommand = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("ignore_command"), p.IgnoreCommand, state.IgnoreCommand) {
		request.CommandForIgnoringBuildStep = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("dev_command"), p.DevCommand, state.DevCommand) {
		request.DevCommand = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("framework"), p.Framework, state.Framework) {
		request.Framework = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("install_command"), p.InstallCommand, state.InstallCommand) {
		request.InstallCommand = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("name"), p.Name, state.Name) {
		request.Name = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("output_directory"), p.OutputDirectory, state.OutputDirectory) {
		request.OutputDirectory = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("preview_deployment_suffix"), p.PreviewDeploymentSuffix, state.PreviewDeploymentSuffix) {
		request.PreviewDeploymentSuffix = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("root_directory"), p.RootDirectory, state.RootDirectory) {
		request.RootDirectory = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("vercel_authentication"), p.VercelAuthentication, state.VercelAuthentication) {
		request.VercelAuthentication = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("password_protection"), p.PasswordProtection, state.PasswordProtection) {
		request.PasswordProtection = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("trusted_ips"), p.TrustedIps, state.TrustedIps) {
		request.TrustedIps = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("trusted_sources"), p.TrustedSources, state.TrustedSources) {
		request.TrustedSources = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("oidc_token_config"), p.OIDCTokenConfig, state.OIDCTokenConfig) {
		request.OIDCTokenConfig = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("options_allowlist"), p.OptionsAllowlist, state.OptionsAllowlist) {
		request.OptionsAllowlist = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("automatically_expose_system_environment_variables"), p.AutoExposeSystemEnvVars, state.AutoExposeSystemEnvVars) {
		request.AutoExposeSystemEnvVars = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("enable_production_feedback"), p.EnableProductionFeedback, state.EnableProductionFeedback) {
		request.EnableProductionFeedback = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("enable_affected_projects_deployments"), p.EnableAffectedProjectsDeployments, state.EnableAffectedProjectsDeployments) {
		request.EnableAffectedProjectsDeployments = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("preview_deployments_disabled"), p.PreviewDeploymentsDisabled, state.PreviewDeploymentsDisabled) {
		request.PreviewDeploymentsDisabled = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("auto_assign_custom_domains"), p.AutoAssignCustomDomains, state.AutoAssignCustomDomains) {
		request.AutoAssignCustomDomains = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("git_lfs"), p.GitLFS, state.GitLFS) {
		request.GitLFS = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("function_failover"), p.FunctionFailover, state.FunctionFailover) {
		request.ServerlessFunctionZeroConfigFailover = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("customer_success_code_visibility"), p.CustomerSuccessCodeVisibility, state.CustomerSuccessCodeVisibility) {
		request.CustomerSupportCodeVisibility = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("git_fork_protection"), p.GitForkProtection, state.GitForkProtection) {
		request.GitForkProtection = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("prioritise_production_builds"), p.PrioritiseProductionBuilds, state.PrioritiseProductionBuilds) {
		request.ProductionDeploymentsFastLane = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("directory_listing"), p.DirectoryListing, state.DirectoryListing) {
		request.DirectoryListing = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("protected_sourcemaps"), p.ProtectedSourcemaps, state.ProtectedSourcemaps) {
		request.ProtectedSourcemaps = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("skew_protection"), p.SkewProtection, state.SkewProtection) {
		request.SkewProtectionMaxAge = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("git_comments"), p.GitComments, state.GitComments) {
		request.GitComments = nil
	}
	if !projectSettingChanged(ctx, &update, path.Root("node_version"), p.NodeVersion, state.NodeVersion) {
		request.NodeVersion = ""
	}

	// Both names control one API setting. An unknown alias must not cause an
	// unchanged configured value to be sent again.
	plannedFeedback := oneBoolPointer(p.EnablePreviewFeedback, p.PreviewComments)
	priorFeedback := oneBoolPointer(state.EnablePreviewFeedback, state.PreviewComments)
	if types.BoolPointerValue(plannedFeedback).Equal(types.BoolPointerValue(priorFeedback)) {
		update.changed(path.Root("enable_preview_feedback"), p.EnablePreviewFeedback, state.EnablePreviewFeedback)
		update.changed(path.Root("preview_comments"), p.PreviewComments, state.PreviewComments)
		request.EnablePreviewFeedback = nil
	}
	if request.ResourceConfig != nil {
		if !changedProjectObjectSetting(ctx, &update, "resource_config", "function_default_cpu_type", p.ResourceConfig, state.ResourceConfig) {
			request.ResourceConfig.FunctionDefaultMemoryType = nil
		}
		if !changedProjectObjectSetting(ctx, &update, "resource_config", "function_default_timeout", p.ResourceConfig, state.ResourceConfig) {
			request.ResourceConfig.FunctionDefaultTimeout = nil
		}
		if !changedProjectObjectSetting(ctx, &update, "resource_config", "fluid", p.ResourceConfig, state.ResourceConfig) {
			request.ResourceConfig.Fluid = nil
		}
		// Regions have a deprecated scalar alias, so compare the effective API
		// value before retaining either Terraform representation.
		priorConfig, d := state.resourceConfig(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return request, update, diags
		}
		priorResourceConfig := priorConfig.toClientResourceConfig(ctx, state.OnDemandConcurrentBuilds, state.BuildMachineType, state.ServerlessFunctionRegion)
		var priorRegions []string
		if priorResourceConfig != nil {
			priorRegions = priorResourceConfig.FunctionDefaultRegions
		}
		plannedRegions, d := types.SetValueFrom(ctx, types.StringType, request.ResourceConfig.FunctionDefaultRegions)
		diags.Append(d...)
		previousRegions, d := types.SetValueFrom(ctx, types.StringType, priorRegions)
		diags.Append(d...)
		if diags.HasError() {
			return request, update, diags
		}
		if plannedRegions.Equal(previousRegions) {
			request.ResourceConfig.FunctionDefaultRegions = nil
			changedProjectObjectSetting(ctx, &update, "resource_config", "function_default_regions", p.ResourceConfig, state.ResourceConfig)
			update.changed(path.Root("serverless_function_region"), p.ServerlessFunctionRegion, state.ServerlessFunctionRegion)
		}
		if !projectSettingChanged(ctx, &update, path.Root("on_demand_concurrent_builds"), p.OnDemandConcurrentBuilds, state.OnDemandConcurrentBuilds) {
			request.ResourceConfig.ElasticConcurrencyEnabled = nil
		}
		if !projectSettingChanged(ctx, &update, path.Root("build_machine_type"), p.BuildMachineType, state.BuildMachineType) {
			request.ResourceConfig.BuildMachineType = nil
			request.ResourceConfig.BuildMachineSelection = nil
		}
		rc := request.ResourceConfig
		if rc.FunctionDefaultMemoryType == nil && rc.FunctionDefaultTimeout == nil && len(rc.FunctionDefaultRegions) == 0 && rc.Fluid == nil && rc.ElasticConcurrencyEnabled == nil && rc.BuildMachineType == nil && rc.BuildMachineSelection == nil {
			request.ResourceConfig = nil
		}
	}
	// The deprecated region alias is derived from resource_config by the API.
	if p.ResourceConfig.Equal(state.ResourceConfig) && p.ServerlessFunctionRegion.Equal(state.ServerlessFunctionRegion) {
		update.changed(path.Root("resource_config"), p.ResourceConfig, state.ResourceConfig)
		update.changed(path.Root("serverless_function_region"), p.ServerlessFunctionRegion, state.ServerlessFunctionRegion)
	}
	if request.GitProviderOptions != nil {
		if !changedProjectObjectSetting(ctx, &update, "git_provider_options", "require_verified_commits", p.GitProviderOptions, state.GitProviderOptions) {
			request.GitProviderOptions.RequireVerifiedCommits = nil
		}
		if !changedProjectObjectSetting(ctx, &update, "git_provider_options", "create_deployments", p.GitProviderOptions, state.GitProviderOptions) {
			request.GitProviderOptions.CreateDeployments = nil
		}
		if !changedProjectObjectSetting(ctx, &update, "git_provider_options", "repository_dispatch_events", p.GitProviderOptions, state.GitProviderOptions) {
			request.GitProviderOptions.DisableRepositoryDispatchEvents = nil
		}
		if !changedProjectObjectSetting(ctx, &update, "git_provider_options", "git_commit_status", p.GitProviderOptions, state.GitProviderOptions) {
			request.GitProviderOptions.GitCommitStatus = nil
		}
		if !changedProjectObjectSetting(ctx, &update, "git_provider_options", "consolidated_git_commit_status", p.GitProviderOptions, state.GitProviderOptions) {
			request.GitProviderOptions.ConsolidatedGitCommitStatus = nil
		}
		if *request.GitProviderOptions == (client.GitProviderOptions{}) {
			request.GitProviderOptions = nil
		}
	}
	if p.GitProviderOptions.Equal(state.GitProviderOptions) {
		update.changed(path.Root("git_provider_options"), p.GitProviderOptions, state.GitProviderOptions)
	}
	return request, update, diags
}

func changedProjectObjectSetting(ctx context.Context, update *settingsUpdate, object, name string, plan, state types.Object) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	planned := plan.Attributes()[name]
	prior := state.Attributes()[name]
	if prior == nil {
		return true
	}
	return projectSettingChanged(ctx, update, path.Root(object).AtName(name), planned, prior)
}
