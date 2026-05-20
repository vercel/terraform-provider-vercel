package vercel

import "github.com/hashicorp/terraform-plugin-framework/types"

type VercelAuthentication struct {
	DeploymentType types.String `tfsdk:"deployment_type"`
}

type PasswordProtection struct {
	DeploymentType types.String `tfsdk:"deployment_type"`
}
type PasswordProtectionWithPassword struct {
	DeploymentType types.String `tfsdk:"deployment_type"`
	Password       types.String `tfsdk:"password"`
}

type TrustedIpAddress struct {
	Value types.String `tfsdk:"value"`
	Note  types.String `tfsdk:"note"`
}

type TrustedIps struct {
	DeploymentType types.String       `tfsdk:"deployment_type"`
	Addresses      []TrustedIpAddress `tfsdk:"addresses"`
	ProtectionMode types.String       `tfsdk:"protection_mode"`
}

type TrustedSources struct {
	Projects        types.Set `tfsdk:"projects"`
	ExternalSources types.Set `tfsdk:"external_sources"`
}

type TrustedSourcesProject struct {
	ProjectID   types.String `tfsdk:"project_id"`
	Label       types.String `tfsdk:"label"`
	CustomAllow types.Set    `tfsdk:"custom_allow"`
}

type TrustedSourcesExternalSource struct {
	Issuer types.String `tfsdk:"issuer"`
	Label  types.String `tfsdk:"label"`
	To     types.Object `tfsdk:"to"`
	Claims types.Map    `tfsdk:"claims"`
}

type TrustedSourcesAccessRule struct {
	From types.Object `tfsdk:"from"`
	To   types.Object `tfsdk:"to"`
}

type TrustedSourcesEnvMatcher struct {
	Slugs  types.Set    `tfsdk:"slugs"`
	Preset types.String `tfsdk:"preset"`
}

type ProtectionBypass struct {
	Scope types.String `tfsdk:"scope"`
}

type OptionsAllowlist struct {
	Paths []OptionsAllowlistPath `tfsdk:"paths"`
}

type OptionsAllowlistPath struct {
	Value types.String `tfsdk:"value"`
}
