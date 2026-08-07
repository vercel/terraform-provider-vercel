package vercel

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// kmsSigningKeyModel mirrors a `client.KMSSigningKey` for use as a computed,
// read-only nested attribute on the issuer resource and data source.
type kmsSigningKeyModel struct {
	KeyID                types.String `tfsdk:"key_id"`
	IssuerID             types.String `tfsdk:"issuer_id"`
	Algorithm            types.String `tfsdk:"algorithm"`
	Status               types.String `tfsdk:"status"`
	PublicKey            types.String `tfsdk:"public_key"`
	PublicKeyFingerprint types.String `tfsdk:"public_key_fingerprint"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	RevokeAt             types.String `tfsdk:"revoke_at"`
}

func kmsSigningKeyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key_id":                 types.StringType,
		"issuer_id":              types.StringType,
		"algorithm":              types.StringType,
		"status":                 types.StringType,
		"public_key":             types.StringType,
		"public_key_fingerprint": types.StringType,
		"created_at":             types.StringType,
		"updated_at":             types.StringType,
		"revoke_at":              types.StringType,
	}
}

func kmsSigningKeyValue(key client.KMSSigningKey) kmsSigningKeyModel {
	return kmsSigningKeyModel{
		KeyID:                types.StringValue(key.KeyID),
		IssuerID:             types.StringValue(key.IssuerID),
		Algorithm:            types.StringValue(key.Algorithm),
		Status:               types.StringValue(key.Status),
		PublicKey:            jsonRawToStringValue(key.PublicKey),
		PublicKeyFingerprint: emptyToNull(key.PublicKeyFingerprint),
		CreatedAt:            types.StringValue(key.CreatedAt),
		UpdatedAt:            types.StringValue(key.UpdatedAt),
		RevokeAt:             emptyToNull(key.RevokeAt),
	}
}

func kmsSigningKeysToList(ctx context.Context, keys []client.KMSSigningKey) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: kmsSigningKeyAttrTypes()}
	models := make([]kmsSigningKeyModel, 0, len(keys))
	for _, key := range keys {
		models = append(models, kmsSigningKeyValue(key))
	}
	return types.ListValueFrom(ctx, objType, models)
}

func kmsIssuerPolicyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"kind":         types.StringType,
		"team_id":      types.StringType,
		"project_id":   types.StringType,
		"client_id":    types.StringType,
		"environments": types.ListType{ElemType: types.StringType},
		"token_claims": types.StringType,
		"created_at":   types.StringType,
		"updated_at":   types.StringType,
	}
}

func kmsIssuerPoliciesToList(ctx context.Context, policies []client.KMSIssuerPolicy) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: kmsIssuerPolicyAttrTypes()}
	values := make([]attr.Value, 0, len(policies))
	for _, policy := range policies {
		environments, d := types.ListValueFrom(ctx, types.StringType, policy.Environments)
		diags.Append(d...)
		obj, d := types.ObjectValue(kmsIssuerPolicyAttrTypes(), map[string]attr.Value{
			"kind":         types.StringValue(policy.Kind),
			"team_id":      emptyToNull(policy.TeamID),
			"project_id":   emptyToNull(policy.ProjectID),
			"client_id":    emptyToNull(policy.ClientID),
			"environments": environments,
			"token_claims": jsonRawToStringValue(policy.TokenClaims),
			"created_at":   types.StringValue(policy.CreatedAt),
			"updated_at":   types.StringValue(policy.UpdatedAt),
		})
		diags.Append(d...)
		values = append(values, obj)
	}
	list, d := types.ListValue(objType, values)
	diags.Append(d...)
	return list, diags
}

func jsonRawToStringValue(raw json.RawMessage) types.String {
	if len(raw) == 0 {
		return types.StringNull()
	}
	return types.StringValue(string(raw))
}

func emptyToNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// normalizeJSON re-marshals a JSON document so two semantically-equal documents
// with different formatting compare equal.
func normalizeJSON(s string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func kmsSameStringSet(left, right []string) bool {
	leftCopy := slices.Clone(left)
	rightCopy := slices.Clone(right)
	slices.Sort(leftCopy)
	slices.Sort(rightCopy)
	return slices.Equal(leftCopy, rightCopy)
}

// kmsEnvironmentsValue keeps the prior (configured) environments list when it
// contains the same set of values returned by the API, so that ordering
// differences do not produce a perpetual diff.
func kmsEnvironmentsValue(ctx context.Context, server []string, prior types.List) (types.List, diag.Diagnostics) {
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorEnvs []string
		diags := prior.ElementsAs(ctx, &priorEnvs, false)
		if !diags.HasError() && kmsSameStringSet(priorEnvs, server) {
			return prior, diags
		}
	}
	return types.ListValueFrom(ctx, types.StringType, server)
}

// kmsTokenClaimsValue keeps the prior (configured) token_claims string when it
// is semantically equal to the value returned by the API, so that whitespace or
// key-ordering differences do not produce a perpetual diff.
func kmsTokenClaimsValue(raw json.RawMessage, prior types.String) types.String {
	if len(raw) == 0 {
		return types.StringNull()
	}
	server := string(raw)
	if !prior.IsNull() && !prior.IsUnknown() {
		priorNorm, priorErr := normalizeJSON(prior.ValueString())
		serverNorm, serverErr := normalizeJSON(server)
		if priorErr == nil && serverErr == nil && priorNorm == serverNorm {
			return prior
		}
	}
	return types.StringValue(server)
}
