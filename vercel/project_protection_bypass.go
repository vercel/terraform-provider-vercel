package vercel

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var protectionBypassForAutomationSecretAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"secret":     types.StringType,
		"note":       types.StringType,
		"is_env_var": types.BoolType,
	},
}

type projectProtectionBypassForAutomationSecret struct {
	Secret   types.String `tfsdk:"secret"`
	Note     types.String `tfsdk:"note"`
	IsEnvVar types.Bool   `tfsdk:"is_env_var"`
}

type desiredProjectProtectionBypassForAutomationSecret struct {
	Secret   string
	Note     *string
	IsEnvVar bool
}

func (p *Project) protectionBypassForAutomationSecrets(ctx context.Context) ([]projectProtectionBypassForAutomationSecret, diag.Diagnostics) {
	if p.ProtectionBypassForAutomationSecrets.IsNull() || p.ProtectionBypassForAutomationSecrets.IsUnknown() {
		return nil, nil
	}

	var secrets []projectProtectionBypassForAutomationSecret
	diags := p.ProtectionBypassForAutomationSecrets.ElementsAs(ctx, &secrets, false)
	return secrets, diags
}

func (p Project) hasConfiguredProtectionBypassForAutomationSecrets() bool {
	return !p.ProtectionBypassForAutomationSecrets.IsNull() && !p.ProtectionBypassForAutomationSecrets.IsUnknown()
}

func automationBypassProtectionEntries(protectionBypass map[string]client.ProtectionBypass) map[string]client.ProtectionBypass {
	entries := map[string]client.ProtectionBypass{}
	for secret, bypass := range protectionBypass {
		if bypass.Scope != "automation-bypass" {
			continue
		}
		entries[secret] = bypass
	}
	return entries
}

func isAutomationBypassEnvVar(bypass client.ProtectionBypass) bool {
	if bypass.IsEnvVar == nil {
		return true
	}
	return *bypass.IsEnvVar
}

func automationBypassEnvVarSecret(protectionBypass map[string]client.ProtectionBypass) string {
	var implicitEnvVar string
	for secret, bypass := range protectionBypass {
		if isAutomationBypassEnvVar(bypass) {
			if bypass.IsEnvVar != nil {
				return secret
			}
			if implicitEnvVar == "" {
				implicitEnvVar = secret
			}
		}
	}
	return implicitEnvVar
}

func sortedProtectionBypassSecrets(protectionBypass map[string]client.ProtectionBypass) []string {
	secrets := make([]string, 0, len(protectionBypass))
	for secret := range protectionBypass {
		secrets = append(secrets, secret)
	}
	sort.Strings(secrets)
	return secrets
}

func sortedProtectionBypassSecretsForRevocation(protectionBypass map[string]client.ProtectionBypass) []string {
	secrets := sortedProtectionBypassSecrets(protectionBypass)
	sort.SliceStable(secrets, func(i, j int) bool {
		leftEnvVar := isAutomationBypassEnvVar(protectionBypass[secrets[i]])
		rightEnvVar := isAutomationBypassEnvVar(protectionBypass[secrets[j]])
		if leftEnvVar == rightEnvVar {
			return secrets[i] < secrets[j]
		}
		return !leftEnvVar && rightEnvVar
	})
	return secrets
}

func protectionBypassForAutomationSecretsSet(protectionBypass map[string]client.ProtectionBypass) types.Set {
	if len(protectionBypass) == 0 {
		return types.SetNull(protectionBypassForAutomationSecretAttrType)
	}

	values := make([]attr.Value, 0, len(protectionBypass))
	for _, secret := range sortedProtectionBypassSecrets(protectionBypass) {
		bypass := protectionBypass[secret]
		values = append(values, types.ObjectValueMust(
			protectionBypassForAutomationSecretAttrType.AttrTypes,
			map[string]attr.Value{
				"secret":     types.StringValue(secret),
				"note":       types.StringPointerValue(bypass.Note),
				"is_env_var": types.BoolValue(isAutomationBypassEnvVar(bypass)),
			},
		))
	}

	return types.SetValueMust(protectionBypassForAutomationSecretAttrType, values)
}

func desiredProtectionBypassForAutomationSecretsMap(secrets []projectProtectionBypassForAutomationSecret) map[string]desiredProjectProtectionBypassForAutomationSecret {
	desired := make(map[string]desiredProjectProtectionBypassForAutomationSecret, len(secrets))
	for _, secret := range secrets {
		desired[secret.Secret.ValueString()] = desiredProjectProtectionBypassForAutomationSecret{
			Secret:   secret.Secret.ValueString(),
			Note:     stringPointerFromValue(secret.Note),
			IsEnvVar: secret.IsEnvVar.ValueBool(),
		}
	}
	return desired
}

func protectionBypassNoteEqual(current *string, desired *string) bool {
	switch {
	case current == nil && desired == nil:
		return true
	case current == nil || desired == nil:
		return false
	default:
		return *current == *desired
	}
}

func protectionBypassUpdateNote(current *string, desired *string) *string {
	if desired != nil {
		return desired
	}
	if current != nil {
		empty := ""
		return &empty
	}
	return nil
}

func stringPointerFromValue(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueString()
	return &value
}

func protectionBypassNeedsUpdate(current client.ProtectionBypass, desired desiredProjectProtectionBypassForAutomationSecret) bool {
	if isAutomationBypassEnvVar(current) != desired.IsEnvVar {
		return true
	}

	return !protectionBypassNoteEqual(current.Note, desired.Note)
}

func (r *projectResource) patchProtectionBypassForAutomation(ctx context.Context, projectID, teamID string, patch client.PatchProtectionBypassForAutomationRequest) (map[string]client.ProtectionBypass, error) {
	patch.ProjectID = projectID
	patch.TeamID = teamID

	protectionBypass, err := r.client.PatchProtectionBypassForAutomation(ctx, patch)
	if err != nil {
		return nil, err
	}

	return automationBypassProtectionEntries(protectionBypass), nil
}

func (r *projectResource) revokeAllProtectionBypassForAutomation(ctx context.Context, projectID, teamID string, current map[string]client.ProtectionBypass) (map[string]client.ProtectionBypass, error) {
	for _, secret := range sortedProtectionBypassSecretsForRevocation(current) {
		var err error
		current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
			Revoke: &client.RevokeProtectionBypassRequest{
				Regenerate: false,
				Secret:     secret,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

func (r *projectResource) reconcileLegacyProtectionBypassForAutomation(ctx context.Context, projectID, teamID string, current map[string]client.ProtectionBypass, plannedSecret string) (map[string]client.ProtectionBypass, error) {
	currentEnvVarSecret := automationBypassEnvVarSecret(current)

	if plannedSecret == "" {
		if currentEnvVarSecret != "" {
			return current, nil
		}

		next, err := r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{})
		if err != nil {
			return nil, err
		}

		if automationBypassEnvVarSecret(next) == "" {
			return nil, fmt.Errorf("unable to determine generated protection bypass secret")
		}

		return next, nil
	}

	if existing, ok := current[plannedSecret]; ok {
		if currentEnvVarSecret == plannedSecret && isAutomationBypassEnvVar(existing) {
			return current, nil
		}
	} else {
		var err error
		current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
			Generate: &client.GenerateProtectionBypassRequest{
				Secret: plannedSecret,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	var err error
	current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
		Update: &client.UpdateProtectionBypassRequest{
			Secret:   plannedSecret,
			IsEnvVar: boolPointer(true),
		},
	})
	if err != nil {
		return nil, err
	}

	if currentEnvVarSecret != "" && currentEnvVarSecret != plannedSecret {
		current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
			Revoke: &client.RevokeProtectionBypassRequest{
				Regenerate: false,
				Secret:     currentEnvVarSecret,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

func (r *projectResource) reconcileManagedProtectionBypassForAutomation(ctx context.Context, projectID, teamID string, current map[string]client.ProtectionBypass, desiredSecrets []projectProtectionBypassForAutomationSecret) (map[string]client.ProtectionBypass, error) {
	desired := desiredProtectionBypassForAutomationSecretsMap(desiredSecrets)

	var desiredEnvVarSecret string
	for secret, bypass := range desired {
		if bypass.IsEnvVar {
			desiredEnvVarSecret = secret
			break
		}
	}

	currentEnvVarSecret := automationBypassEnvVarSecret(current)

	for _, secret := range sortedDesiredProtectionBypassSecrets(desired) {
		if _, ok := current[secret]; ok {
			continue
		}

		var err error
		current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
			Generate: &client.GenerateProtectionBypassRequest{
				Secret: secret,
				Note:   desired[secret].Note,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	if desiredEnvVarSecret != "" {
		desiredEnvVar := desired[desiredEnvVarSecret]
		currentEnvVar, ok := current[desiredEnvVarSecret]
		if !ok || protectionBypassNeedsUpdate(currentEnvVar, desiredEnvVar) || currentEnvVarSecret != desiredEnvVarSecret {
			var err error
			current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
				Update: &client.UpdateProtectionBypassRequest{
					Secret:   desiredEnvVarSecret,
					IsEnvVar: boolPointer(true),
					Note:     protectionBypassUpdateNote(currentEnvVar.Note, desiredEnvVar.Note),
				},
			})
			if err != nil {
				return nil, err
			}
		}
	}

	for _, secret := range sortedDesiredProtectionBypassSecrets(desired) {
		if secret == desiredEnvVarSecret {
			continue
		}

		currentBypass, ok := current[secret]
		if !ok {
			continue
		}

		if !protectionBypassNeedsUpdate(currentBypass, desired[secret]) {
			continue
		}

		var err error
		current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
			Update: &client.UpdateProtectionBypassRequest{
				Secret:   secret,
				IsEnvVar: boolPointer(false),
				Note:     protectionBypassUpdateNote(currentBypass.Note, desired[secret].Note),
			},
		})
		if err != nil {
			return nil, err
		}
	}

	for _, secret := range sortedProtectionBypassSecrets(current) {
		if _, ok := desired[secret]; ok {
			continue
		}

		var err error
		current, err = r.patchProtectionBypassForAutomation(ctx, projectID, teamID, client.PatchProtectionBypassForAutomationRequest{
			Revoke: &client.RevokeProtectionBypassRequest{
				Regenerate: false,
				Secret:     secret,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

func sortedDesiredProtectionBypassSecrets(desired map[string]desiredProjectProtectionBypassForAutomationSecret) []string {
	secrets := make([]string, 0, len(desired))
	for secret := range desired {
		secrets = append(secrets, secret)
	}
	sort.Strings(secrets)
	return secrets
}

func boolPointer(value bool) *bool {
	return &value
}
