package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &kmsSigningKeyResource{}
	_ resource.ResourceWithConfigure   = &kmsSigningKeyResource{}
	_ resource.ResourceWithImportState = &kmsSigningKeyResource{}
)

func newKMSSigningKeyResource() resource.Resource {
	return &kmsSigningKeyResource{}
}

type kmsSigningKeyResource struct {
	client *client.Client
}

type kmsSigningKeyResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	TeamID               types.String `tfsdk:"team_id"`
	IssuerID             types.String `tfsdk:"issuer_id"`
	Keepers              types.Map    `tfsdk:"keepers"`
	ImportKey            types.String `tfsdk:"import_key"`
	KeyID                types.String `tfsdk:"key_id"`
	RevokePreviousAt     types.String `tfsdk:"revoke_previous_at"`
	Algorithm            types.String `tfsdk:"algorithm"`
	Status               types.String `tfsdk:"status"`
	PublicKey            types.String `tfsdk:"public_key"`
	PublicKeyFingerprint types.String `tfsdk:"public_key_fingerprint"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	RevokeAt             types.String `tfsdk:"revoke_at"`
}

func (r *kmsSigningKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_signing_key"
}

func (r *kmsSigningKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *kmsSigningKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel KMS signing key created by rotating an issuer's key.

Rotating a key is an imperative operation. This resource creates a new signing
key when it is first applied, and creates a replacement key whenever a
` + "`RequiresReplace`" + ` input changes. Use the ` + "`keepers`" + ` map to force a
rotation on demand. Signing keys cannot be deleted individually; removing this
resource only drops it from Terraform state (the underlying key is retired the
next time the issuer's key is rotated).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID (`kid`) of the signing key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the issuer exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"issuer_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the issuer to rotate a key for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"keepers": schema.MapAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Arbitrary map of values that, when changed, forces a new signing key to be rotated in.",
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"import_key": schema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				Description:   "A PEM-encoded private key to rotate in for an external issuer. Changing this forces a new signing key. This value is never returned by the API.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key_id": schema.StringAttribute{
				Optional:      true,
				Description:   "The key ID (`kid`) to assign to the imported key. Only valid when `import_key` is set. Changing this forces a new signing key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("import_key")),
				},
			},
			"revoke_previous_at": schema.StringAttribute{
				Optional:      true,
				Description:   "An RFC3339 timestamp at which the issuer's previous signing keys should stop being used. Changing this forces a new signing key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"algorithm": schema.StringAttribute{
				Computed:      true,
				Description:   "The signing algorithm of the key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The status of the key, either `active` or `revoking`.",
			},
			"public_key": schema.StringAttribute{
				Computed:      true,
				Description:   "The public key as a JSON-encoded JWK.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"public_key_fingerprint": schema.StringAttribute{
				Computed:      true,
				Description:   "The fingerprint of the public key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "The time the key was created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time the key was last updated.",
			},
			"revoke_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the key is scheduled to be revoked, if any.",
			},
		},
	}
}

func (r *kmsSigningKeyResource) modelFromResponse(key client.KMSSigningKey, teamID, issuerID types.String, keepers types.Map, importKey, keyID, revokePreviousAt types.String) kmsSigningKeyResourceModel {
	return kmsSigningKeyResourceModel{
		ID:                   types.StringValue(key.KeyID),
		TeamID:               teamID,
		IssuerID:             issuerID,
		Keepers:              keepers,
		ImportKey:            importKey,
		KeyID:                keyID,
		RevokePreviousAt:     revokePreviousAt,
		Algorithm:            types.StringValue(key.Algorithm),
		Status:               types.StringValue(key.Status),
		PublicKey:            jsonRawToStringValue(key.PublicKey),
		PublicKeyFingerprint: emptyToNull(key.PublicKeyFingerprint),
		CreatedAt:            types.StringValue(key.CreatedAt),
		UpdatedAt:            types.StringValue(key.UpdatedAt),
		RevokeAt:             emptyToNull(key.RevokeAt),
	}
}

func (r *kmsSigningKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan kmsSigningKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.RotateKMSIssuerKey(ctx, client.RotateKMSIssuerKeyRequest{
		IssuerID:         plan.IssuerID.ValueString(),
		TeamID:           plan.TeamID.ValueString(),
		RevokePreviousAt: plan.RevokePreviousAt.ValueString(),
		ImportKey:        plan.ImportKey.ValueString(),
		KeyID:            plan.KeyID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error rotating KMS signing key",
			"Could not rotate KMS signing key, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "rotated kms signing key", map[string]any{
		"issuer_id": plan.IssuerID.ValueString(),
		"key_id":    out.KeyID,
	})

	teamID := toTeamID(r.client.TeamID(plan.TeamID.ValueString()))
	result := r.modelFromResponse(out, teamID, plan.IssuerID, plan.Keepers, plan.ImportKey, plan.KeyID, plan.RevokePreviousAt)
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsSigningKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state kmsSigningKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, state.IssuerID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading KMS signing key",
			fmt.Sprintf("Could not read KMS Issuer %s, unexpected error: %s", state.IssuerID.ValueString(), err),
		)
		return
	}

	key, ok := kmsFindSigningKey(out.SigningKeys, state.ID.ValueString())
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	teamID := toTeamID(r.client.TeamID(state.TeamID.ValueString()))
	result := r.modelFromResponse(key, teamID, state.IssuerID, state.Keepers, state.ImportKey, state.KeyID, state.RevokePreviousAt)
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

// Update is unreachable in practice because every configurable attribute forces
// replacement, but the framework requires the method. It refreshes the computed
// values from the API for the current key.
func (r *kmsSigningKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan kmsSigningKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, plan.IssuerID.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating KMS signing key",
			fmt.Sprintf("Could not read KMS Issuer %s, unexpected error: %s", plan.IssuerID.ValueString(), err),
		)
		return
	}

	key, ok := kmsFindSigningKey(out.SigningKeys, plan.ID.ValueString())
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	teamID := toTeamID(r.client.TeamID(plan.TeamID.ValueString()))
	result := r.modelFromResponse(key, teamID, plan.IssuerID, plan.Keepers, plan.ImportKey, plan.KeyID, plan.RevokePreviousAt)
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsSigningKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state kmsSigningKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "removing kms signing key from state; individual keys cannot be deleted and are retired on the next rotation", map[string]any{
		"issuer_id": state.IssuerID.ValueString(),
		"key_id":    state.ID.ValueString(),
	})
}

func (r *kmsSigningKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, issuerID, keyID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing KMS signing key",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/issuer_id/key_id\" or \"issuer_id/key_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, issuerID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing KMS signing key",
			fmt.Sprintf("Could not import KMS signing key %s %s/%s, unexpected error: %s", teamID, issuerID, keyID, err),
		)
		return
	}

	key, ok := kmsFindSigningKey(out.SigningKeys, keyID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing KMS signing key",
			fmt.Sprintf("No signing key %s found on issuer %s", keyID, issuerID),
		)
		return
	}

	result := r.modelFromResponse(key, toTeamID(r.client.TeamID(teamID)), types.StringValue(issuerID), types.MapNull(types.StringType), types.StringNull(), types.StringNull(), types.StringNull())
	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func kmsFindSigningKey(keys []client.KMSSigningKey, keyID string) (client.KMSSigningKey, bool) {
	for _, key := range keys {
		if key.KeyID == keyID {
			return key, true
		}
	}
	return client.KMSSigningKey{}, false
}
