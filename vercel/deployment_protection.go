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

type ProtectionBypass struct {
	Scope types.String `tfsdk:"scope"`
}
