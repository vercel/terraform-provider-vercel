package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// kmsAlgorithms is the set of signing algorithms supported by Vercel KMS.
var kmsAlgorithms = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
	"EdDSA",
}

var (
	_ resource.Resource                = &kmsIssuerResource{}
	_ resource.ResourceWithConfigure   = &kmsIssuerResource{}
	_ resource.ResourceWithImportState = &kmsIssuerResource{}
)

func newKMSIssuerResource() resource.Resource {
	return &kmsIssuerResource{}
}

type kmsIssuerResource struct {
	client *client.Client
}

type kmsIssuerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	TeamID      types.String `tfsdk:"team_id"`
	Name        types.String `tfsdk:"name"`
	Algorithm   types.String `tfsdk:"algorithm"`
	ImportKey   types.String `tfsdk:"import_key"`
	KeyID       types.String `tfsdk:"key_id"`
	OwnerID     types.String `tfsdk:"owner_id"`
	Origin      types.String `tfsdk:"origin"`
	ManagedBy   types.String `tfsdk:"managed_by"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	SigningKeys types.List   `tfsdk:"signing_keys"`
}

func (r *kmsIssuerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_issuer"
}

func (r *kmsIssuerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// kmsSigningKeysResourceAttribute is the computed, read-only nested list of an
// issuer's signing keys as exposed on the resource.
func kmsSigningKeysResourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "The signing keys belonging to the issuer. Keys are created automatically and can be rotated with the `vercel_kms_signing_key` resource.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key_id":                 schema.StringAttribute{Computed: true, Description: "The ID of the signing key. Used as the JWT/JWKS `kid`."},
				"issuer_id":              schema.StringAttribute{Computed: true, Description: "The ID of the issuer the key belongs to."},
				"algorithm":              schema.StringAttribute{Computed: true, Description: "The signing algorithm of the key."},
				"status":                 schema.StringAttribute{Computed: true, Description: "The status of the key, either `active` or `revoking`."},
				"public_key":             schema.StringAttribute{Computed: true, Description: "The public key as a JSON-encoded JWK."},
				"public_key_fingerprint": schema.StringAttribute{Computed: true, Description: "The fingerprint of the public key."},
				"created_at":             schema.StringAttribute{Computed: true, Description: "The time the key was created."},
				"updated_at":             schema.StringAttribute{Computed: true, Description: "The time the key was last updated."},
				"revoke_at":              schema.StringAttribute{Computed: true, Description: "The time at which the key is scheduled to be revoked, if any."},
			},
		},
	}
}

func (r *kmsIssuerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel KMS Issuer resource.

A KMS Issuer signs JSON Web Tokens (JWTs) and exposes a JWKS endpoint at
` + "`https://kms.vercel.com/{id}/jwks.json`" + `. An initial signing key is created
automatically; use ` + "`vercel_kms_signing_key`" + ` to rotate keys and
` + "`vercel_kms_issuer_policy`" + ` to grant projects the ability to sign tokens.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the issuer.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the issuer should be created under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the issuer.",
			},
			"algorithm": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The signing algorithm to use for the issuer. Defaults to `RS512`. Changing this forces a new issuer to be created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseNonNullStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.OneOf(kmsAlgorithms...),
				},
			},
			"import_key": schema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				Description:   "A PEM-encoded private key to import for the issuer. When provided, the issuer's origin becomes `external` and the algorithm is inferred from the key. Changing this forces a new issuer to be created. This value is never returned by the API and cannot be recovered on import.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key_id": schema.StringAttribute{
				Optional:      true,
				Description:   "The key ID (`kid`) to assign to the imported key. Only valid when `import_key` is set. Changing this forces a new issuer to be created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("import_key")),
				},
			},
			"owner_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the team that owns the issuer.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"origin": schema.StringAttribute{
				Computed:      true,
				Description:   "The origin of the issuer's key material, either `vercel` or `external`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"managed_by": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifies the policy that manages this issuer, if any.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "The time the issuer was created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time the issuer was last updated.",
			},
			"signing_keys": kmsSigningKeysResourceAttribute(),
		},
	}
}

func (r *kmsIssuerResource) modelFromResponse(ctx context.Context, issuer client.KMSIssuer, importKey, keyID types.String) (kmsIssuerResourceModel, diag.Diagnostics) {
	signingKeys, diags := kmsSigningKeysToList(ctx, issuer.SigningKeys)
	model := kmsIssuerResourceModel{
		ID:          types.StringValue(issuer.ID),
		TeamID:      toTeamID(issuer.TeamID),
		Name:        types.StringValue(issuer.Name),
		Algorithm:   types.StringValue(issuer.Algorithm),
		ImportKey:   importKey,
		KeyID:       keyID,
		OwnerID:     types.StringValue(issuer.OwnerID),
		Origin:      types.StringValue(issuer.Origin),
		ManagedBy:   emptyToNull(issuer.ManagedBy),
		CreatedAt:   types.StringValue(issuer.CreatedAt),
		UpdatedAt:   types.StringValue(issuer.UpdatedAt),
		SigningKeys: signingKeys,
	}
	return model, diags
}

func (r *kmsIssuerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan kmsIssuerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateKMSIssuer(ctx, client.CreateKMSIssuerRequest{
		TeamID:    plan.TeamID.ValueString(),
		Name:      plan.Name.ValueString(),
		Algorithm: plan.Algorithm.ValueString(),
		ImportKey: plan.ImportKey.ValueString(),
		KeyID:     plan.KeyID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating KMS Issuer",
			"Could not create KMS Issuer, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "created kms issuer", map[string]any{
		"team_id":   out.TeamID,
		"issuer_id": out.ID,
		"name":      out.Name,
	})

	result, diags := r.modelFromResponse(ctx, out, plan.ImportKey, plan.KeyID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsIssuerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state kmsIssuerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading KMS Issuer",
			fmt.Sprintf("Could not read KMS Issuer %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	// import_key and key_id are never returned by the API, so preserve the
	// values already tracked in state.
	result, diags := r.modelFromResponse(ctx, out, state.ImportKey, state.KeyID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsIssuerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan kmsIssuerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateKMSIssuer(ctx, client.UpdateKMSIssuerRequest{
		IssuerID: plan.ID.ValueString(),
		TeamID:   plan.TeamID.ValueString(),
		Name:     plan.Name.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating KMS Issuer",
			fmt.Sprintf("Could not update KMS Issuer %s, unexpected error: %s", plan.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "updated kms issuer", map[string]any{
		"team_id":   out.TeamID,
		"issuer_id": out.ID,
		"name":      out.Name,
	})

	result, diags := r.modelFromResponse(ctx, out, plan.ImportKey, plan.KeyID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsIssuerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state kmsIssuerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteKMSIssuer(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting KMS Issuer",
			fmt.Sprintf("Could not delete KMS Issuer %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted kms issuer", map[string]any{
		"team_id":   state.TeamID.ValueString(),
		"issuer_id": state.ID.ValueString(),
	})
}

func (r *kmsIssuerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, issuerID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing KMS Issuer",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/issuer_id\" or \"issuer_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, issuerID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing KMS Issuer",
			fmt.Sprintf("Could not import KMS Issuer %s %s, unexpected error: %s", teamID, issuerID, err),
		)
		return
	}

	result, diags := r.modelFromResponse(ctx, out, types.StringNull(), types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
