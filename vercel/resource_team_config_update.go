package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func teamString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	return value.ValueStringPointer()
}

func teamBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	return value.ValueBoolPointer()
}

func (u *settingsUpdate) string(p path.Path, plan, state types.String) *string {
	if !u.changed(p, plan, state) {
		return nil
	}
	return teamString(plan)
}

func (u *settingsUpdate) bool(p path.Path, plan, state types.Bool) *bool {
	if !u.changed(p, plan, state) {
		return nil
	}
	return teamBool(plan)
}

func (p *TeamConfig) changedTeamSettings(ctx context.Context, state TeamConfig) (client.UpdateTeamRequest, settingsUpdate, diag.Diagnostics) {
	var update settingsUpdate
	request := client.UpdateTeamRequest{
		TeamID:                             state.ID.ValueString(),
		Name:                               update.string(path.Root("name"), p.Name, state.Name),
		Slug:                               update.string(path.Root("slug"), p.Slug, state.Slug),
		Description:                        update.string(path.Root("description"), p.Description, state.Description),
		EmailDomain:                        update.string(path.Root("email_domain"), p.EmailDomain, state.EmailDomain),
		PreviewDeploymentSuffix:            update.string(path.Root("preview_deployment_suffix"), p.PreviewDeploymentSuffix, state.PreviewDeploymentSuffix),
		EnablePreviewFeedback:              update.string(path.Root("enable_preview_feedback"), p.EnablePreviewFeedback, state.EnablePreviewFeedback),
		EnableProductionFeedback:           update.string(path.Root("enable_production_feedback"), p.EnableProductionFeedback, state.EnableProductionFeedback),
		SensitiveEnvironmentVariablePolicy: update.string(path.Root("sensitive_environment_variable_policy"), p.SensitiveEnvironmentVariablePolicy, state.SensitiveEnvironmentVariablePolicy),
		DisjunctiveProductionSecretPolicy:  update.string(path.Root("disjunctive_production_secret_policy"), p.DisjunctiveProductionSecretPolicy, state.DisjunctiveProductionSecretPolicy),
		HideIPAddresses:                    update.bool(path.Root("hide_ip_addresses"), p.HideIPAddresses, state.HideIPAddresses),
		HideIPAddressesInLogDrains:         update.bool(path.Root("hide_ip_addresses_in_log_drains"), p.HideIPAddressesInLogDrains, state.HideIPAddressesInLogDrains),
	}
	// Invite codes are read-only here, but can also change outside Terraform.
	update.changed(path.Root("invite_code"), p.InviteCode, state.InviteCode)

	var diags diag.Diagnostics
	if update.changed(path.Root("default_passport"), p.DefaultPassport, state.DefaultPassport) && !p.DefaultPassport.IsNull() && !p.DefaultPassport.IsUnknown() {
		values := p.DefaultPassport.Attributes()
		if !values["enabled"].IsUnknown() && !values["connector_id"].IsUnknown() && !values["deployment_type"].IsUnknown() {
			var d diag.Diagnostics
			request.DefaultPassport, d = passportUpdate(ctx, p.DefaultPassport)
			diags.Append(d...)
		}
	}
	if update.changed(path.Root("default_deployment_protection"), p.DefaultDeploymentProtection, state.DefaultDeploymentProtection) && !p.DefaultDeploymentProtection.IsNull() && !p.DefaultDeploymentProtection.IsUnknown() {
		var d diag.Diagnostics
		request.DefaultDeploymentProtection, d = teamDeploymentProtection(ctx, p.DefaultDeploymentProtection)
		diags.Append(d...)
	}
	if update.changed(path.Root("default_build_machine_type"), p.DefaultBuildMachineType, state.DefaultBuildMachineType) {
		if value := teamString(p.DefaultBuildMachineType); value != nil {
			request.ResourceConfig = &client.TeamResourceConfig{BuildMachine: &client.TeamBuildMachine{Default: value}}
		}
	}
	if update.changed(path.Root("remote_caching"), p.RemoteCaching, state.RemoteCaching) && !p.RemoteCaching.IsNull() && !p.RemoteCaching.IsUnknown() {
		var caching RemoteCaching
		diags.Append(p.RemoteCaching.As(ctx, &caching, basetypes.ObjectAsOptions{})...)
		request.RemoteCaching = caching.toUpdateTeamRequest()
	}
	if update.changed(path.Root("saml"), p.Saml, state.Saml) && !p.Saml.IsNull() && !p.Saml.IsUnknown() {
		var d diag.Diagnostics
		request.Saml, d = changedTeamSaml(ctx, &update, p.Saml, state.Saml)
		diags.Append(d...)
	}
	return request, update, diags
}

func changedTeamSaml(ctx context.Context, update *settingsUpdate, plan, state types.Object) (*client.UpdateSamlConfig, diag.Diagnostics) {
	planned := plan.Attributes()
	prior := state.Attributes()
	if state.IsNull() {
		prior = map[string]attr.Value{
			"enforced": types.BoolNull(),
			"roles":    types.MapNull(samlRoleType),
		}
	}
	request := &client.UpdateSamlConfig{
		Enforced: update.bool(path.Root("saml").AtName("enforced"), planned["enforced"].(types.Bool), prior["enforced"].(types.Bool)),
	}
	var diags diag.Diagnostics
	roles := planned["roles"].(types.Map)
	if update.changed(path.Root("saml").AtName("roles"), roles, prior["roles"]) && !roles.IsNull() && !roles.IsUnknown() {
		value, err := roles.ToTerraformValue(ctx)
		if err != nil {
			diags.AddError("Error reading SAML roles", err.Error())
			return nil, diags
		}
		if value.IsFullyKnown() {
			var mappings map[string]SamlRoles
			diags.Append(roles.ElementsAs(ctx, &mappings, false)...)
			if diags.HasError() {
				return nil, diags
			}
			request.Roles = samlRolesToClient(mappings)
		}
	}
	if request.Enforced == nil && request.Roles == nil {
		return nil, diags
	}
	return request, diags
}

func teamDeploymentProtection(ctx context.Context, value types.Object) (*client.DefaultDeploymentProtection, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var defaults DefaultDeploymentProtection
	diags := value.As(ctx, &defaults, basetypes.ObjectAsOptions{})
	if diags.HasError() || defaults.VercelAuthentication.IsNull() || defaults.VercelAuthentication.IsUnknown() {
		return nil, diags
	}
	var authentication VercelAuthentication
	diags.Append(defaults.VercelAuthentication.As(ctx, &authentication, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || !knownString(authentication.DeploymentType) {
		return nil, diags
	}
	return &client.DefaultDeploymentProtection{VercelAuthentication: authentication.toVercelAuthentication()}, diags
}
