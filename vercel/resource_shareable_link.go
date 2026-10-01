package vercel

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &shareableLinkResource{}
	_ resource.ResourceWithConfigure   = &shareableLinkResource{}
	_ resource.ResourceWithImportState = &shareableLinkResource{}
)

type shareableLinkResource struct{ client *client.Client }

func newShareableLinkResource() resource.Resource { return &shareableLinkResource{} }
func (r *shareableLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shareable_link"
}
func (r *shareableLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
	}
}

var shareableAliasValidators = []validator.String{
	stringvalidator.LengthAtMost(253),
	stringvalidator.RegexMatches(regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`), "alias must be a lowercase hostname without a scheme, path, or query string"),
}

func (r *shareableLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a shareable link for an existing Vercel alias. Anyone with the link can bypass Deployment Protection for that alias. Ordinary deployment URLs without an alias record are not supported.

~> **Hobby plans:** Only one shareable link can exist per account. Creating a link automatically revokes other shareable links across the account, including links managed outside Terraform. Managing multiple links on a Hobby account will cause recurring drift.

The secret and URL are sensitive but are stored in Terraform state. Protect access to your state. Only one shareable link can exist per alias; import an existing link before managing it. Custom shareable-link parameters are not supported.

Changing ttl_seconds or rotation_id rotates the secret and immediately invalidates the old link. Expired links remain managed and are not automatically renewed. Create-before-destroy is not supported for the same alias.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The non-secret ID of the alias owning the link.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"alias": schema.StringAttribute{
				Required:      true,
				Description:   "The existing alias hostname, without a scheme or path.",
				Validators:    shareableAliasValidators,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The team ID. Defaults to the team configured in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Computed:    true,
				Description: "The project owning the alias.",
			},
			"ttl_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "How long a newly created or rotated link is valid, in seconds (1 to 63072000). Omit for no expiry. Changing this rotates the link.",
				Validators:  []validator.Int64{int64validator.Between(1, 63072000)},
			},
			"rotation_id": schema.StringAttribute{
				Optional:    true,
				Description: "Change this value to explicitly rotate the link, including renewing an expired link.",
			},
			"secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The generated bearer secret. Stored in Terraform state.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The HTTPS shareable URL, including the secret. Stored in Terraform state.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation time in Unix milliseconds.",
			},
			"created_by": schema.StringAttribute{
				Computed:    true,
				Description: "The creator's ID.",
			},
			"expires_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Expiry time in Unix seconds, or null for no expiry.",
			},
			"active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the link has not expired, as of the last refresh.",
			},
		}}
}

// ShareableLinkResourceState adds resource configuration to the shared link state.
type ShareableLinkResourceState struct {
	ShareableLink
	TTLSeconds types.Int64  `tfsdk:"ttl_seconds"`
	RotationID types.String `tfsdk:"rotation_id"`
}

func (r *shareableLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ShareableLinkResourceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alias, err := r.client.GetAlias(ctx, plan.Alias.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading alias", err.Error())
		return
	}
	if alias.UID == "" || alias.Alias != plan.Alias.ValueString() {
		resp.Diagnostics.AddError("Invalid alias response", "Could not resolve the configured alias to its owning document.")
		return
	}
	for _, bypass := range alias.ProtectionBypass {
		if bypass.Scope == "shareable-link" {
			resp.Diagnostics.AddError("Shareable link already exists", "Import the existing shareable link before managing it.")
			return
		}
	}
	bypasses, err := r.client.UpdateShareableLink(ctx, client.UpdateShareableLinkRequest{AliasID: alias.UID, TeamID: alias.TeamID, TTLSeconds: plan.TTLSeconds.ValueInt64Pointer()})
	if err != nil {
		resp.Diagnostics.AddError("Error creating shareable link", err.Error())
		return
	}
	secret, link, err := findShareableLink(bypasses)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created shareable link", err.Error())
		return
	}
	plan.ShareableLink = shareableLinkState(alias, secret, link)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *shareableLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ShareableLinkResourceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alias, err := r.client.GetAlias(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading shareable link", err.Error())
		return
	}
	if alias.UID != state.ID.ValueString() || alias.ProjectID != state.ProjectID.ValueString() {
		resp.Diagnostics.AddError("Shareable link ownership changed", "The alias no longer belongs to the document and project recorded in state.")
		return
	}
	link, ok := alias.ProtectionBypass[state.Secret.ValueString()]
	if !ok || link.Scope != "shareable-link" {
		for _, bypass := range alias.ProtectionBypass {
			if bypass.Scope == "shareable-link" {
				resp.Diagnostics.AddError("Shareable link rotated outside Terraform", "Import the replacement link to explicitly adopt it. Terraform will not take ownership of a different secret automatically.")
				return
			}
		}
		// The API omits this map when the caller cannot read protection bypasses.
		if alias.ProtectionBypass == nil {
			resp.Diagnostics.AddError("Cannot read protection bypasses", "The response omitted protectionBypass. Check the API token's permissions before treating the link as revoked.")
			return
		}
		resp.State.RemoveResource(ctx)
		return
	}
	state.ShareableLink = shareableLinkState(alias, state.Secret.ValueString(), link)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *shareableLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ShareableLinkResourceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.TTLSeconds.Equal(state.TTLSeconds) && plan.RotationID.Equal(state.RotationID) {
		plan.ShareableLink = state.ShareableLink
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}
	alias, err := r.client.GetAlias(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading alias before rotation", err.Error())
		return
	}
	if alias.UID != state.ID.ValueString() || alias.ProjectID != state.ProjectID.ValueString() {
		resp.Diagnostics.AddError("Shareable link ownership changed", "The alias no longer belongs to the document and project recorded in state.")
		return
	}
	bypasses, err := r.client.UpdateShareableLink(ctx, client.UpdateShareableLinkRequest{AliasID: state.ID.ValueString(), TeamID: state.TeamID.ValueString(), Secret: state.Secret.ValueString(), Revoke: true, Regenerate: true, TTLSeconds: plan.TTLSeconds.ValueInt64Pointer()})
	if err != nil {
		resp.Diagnostics.AddError("Error rotating shareable link", err.Error())
		return
	}
	secret, link, err := findShareableLink(bypasses)
	if err != nil {
		resp.Diagnostics.AddError("Error reading rotated shareable link", err.Error())
		return
	}
	plan.ShareableLink = shareableLinkState(client.AliasResponse{UID: state.ID.ValueString(), Alias: state.Alias.ValueString(), ProjectID: state.ProjectID.ValueString(), TeamID: state.TeamID.ValueString()}, secret, link)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *shareableLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ShareableLinkResourceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.UpdateShareableLink(ctx, client.UpdateShareableLinkRequest{AliasID: state.ID.ValueString(), TeamID: state.TeamID.ValueString(), Secret: state.Secret.ValueString(), Revoke: true})
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error revoking shareable link", err.Error())
	}
}

func (r *shareableLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	teamID, aliasName := "", parts[0]
	if len(parts) == 2 {
		teamID, aliasName = parts[0], parts[1]
	}
	if len(parts) > 2 || aliasName == "" || (len(parts) == 2 && teamID == "") {
		resp.Diagnostics.AddError("Invalid import ID", "Use alias or team_id/alias. No secret is required.")
		return
	}
	alias, err := r.client.GetAlias(ctx, aliasName, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing shareable link", err.Error())
		return
	}
	secret, link, err := findShareableLink(alias.ProtectionBypass)
	if err != nil {
		resp.Diagnostics.AddError("Error importing shareable link", err.Error())
		return
	}
	state := ShareableLinkResourceState{ShareableLink: shareableLinkState(alias, secret, link), TTLSeconds: types.Int64Null(), RotationID: types.StringNull()}
	if link.Expires != nil {
		state.TTLSeconds = types.Int64Value(*link.Expires - link.CreatedAt/1000)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
