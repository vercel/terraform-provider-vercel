package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &userTokenResource{}
	_ resource.ResourceWithConfigure   = &userTokenResource{}
	_ resource.ResourceWithImportState = &userTokenResource{}
)

func newUserTokenResource() resource.Resource {
	return &userTokenResource{}
}

type userTokenResource struct {
	client *client.Client
}

func (r *userTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_token"
}

func (r *userTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userTokenResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a User Token resource.

A User Token is an authentication token that can be used to access the Vercel API.

Creating user tokens requires a Vercel API token with full account access. Limited tokens cannot create additional user tokens.

The ` + "`bearer_token`" + ` value is only returned during creation and cannot be retrieved again later. Imported resources will not populate ` + "`bearer_token`" + `.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The unique identifier of the token.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:   "The human-readable name of the token.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
				},
			},
			"expires_at": schema.Int64Attribute{
				Description:   "The Unix timestamp in milliseconds when the token should expire.",
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project this token should be scoped to. Requires team scope.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the Vercel team scope for this token. Required when creating a team-scoped token if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"bearer_token": schema.StringAttribute{
				Description:   "The actual token value. This is only returned during create and then preserved from Terraform state.",
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Description:   "The type of the token.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"origin": schema.StringAttribute{
				Description:   "How the token was created.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"prefix": schema.StringAttribute{
				Description:   "The token prefix used for identification.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"suffix": schema.StringAttribute{
				Description:   "The token suffix used for identification.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Description:   "The Unix timestamp in milliseconds when the token was created.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"active_at": schema.Int64Attribute{
				Description:   "The Unix timestamp in milliseconds when the token was last active.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"leaked_at": schema.Int64Attribute{
				Description:   "The Unix timestamp in milliseconds when the token was marked as leaked.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"leaked_url": schema.StringAttribute{
				Description:   "The URL where the token was discovered as leaked.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

type UserToken struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ExpiresAt   types.Int64  `tfsdk:"expires_at"`
	ProjectID   types.String `tfsdk:"project_id"`
	TeamID      types.String `tfsdk:"team_id"`
	BearerToken types.String `tfsdk:"bearer_token"`
	Type        types.String `tfsdk:"type"`
	Origin      types.String `tfsdk:"origin"`
	Prefix      types.String `tfsdk:"prefix"`
	Suffix      types.String `tfsdk:"suffix"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	ActiveAt    types.Int64  `tfsdk:"active_at"`
	LeakedAt    types.Int64  `tfsdk:"leaked_at"`
	LeakedURL   types.String `tfsdk:"leaked_url"`
}

func responseToUserToken(out client.UserToken, bearerToken types.String) UserToken {
	if out.BearerToken != nil {
		bearerToken = types.StringPointerValue(out.BearerToken)
	}

	return UserToken{
		ID:          types.StringValue(out.ID),
		Name:        types.StringValue(out.Name),
		ExpiresAt:   types.Int64PointerValue(out.ExpiresAt),
		ProjectID:   types.StringPointerValue(out.ProjectID),
		TeamID:      types.StringPointerValue(out.TeamID),
		BearerToken: bearerToken,
		Type:        types.StringValue(out.Type),
		Origin:      types.StringPointerValue(out.Origin),
		Prefix:      types.StringPointerValue(out.Prefix),
		Suffix:      types.StringPointerValue(out.Suffix),
		CreatedAt:   types.Int64Value(out.CreatedAt),
		ActiveAt:    types.Int64Value(out.ActiveAt),
		LeakedAt:    types.Int64PointerValue(out.LeakedAt),
		LeakedURL:   types.StringPointerValue(out.LeakedURL),
	}
}

func (r *userTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserToken
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(plan.Name.ValueString())
	if name == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Invalid user token name",
			"Token name cannot be empty or whitespace.",
		)
		return
	}
	if name != plan.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Invalid user token name",
			"Token name must not have leading or trailing whitespace.",
		)
		return
	}
	if !plan.ProjectID.IsNull() && r.client.TeamID(plan.TeamID.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"),
			"Project-scoped token requires team scope",
			"`project_id` can only be used when the token is created in a team scope. Set `team_id` or configure a default team on the provider.",
		)
		return
	}

	out, err := r.client.CreateUserToken(ctx, client.CreateUserTokenRequest{
		Name:      name,
		ExpiresAt: plan.ExpiresAt.ValueInt64Pointer(),
		ProjectID: plan.ProjectID.ValueStringPointer(),
		TeamID:    plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating User Token",
			"Could not create User Token, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToUserToken(out, types.StringNull())
	tflog.Info(ctx, "created user token", map[string]any{
		"token_id": result.ID.ValueString(),
		"team_id":  result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *userTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserToken
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetUserToken(ctx, state.ID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading User Token",
			fmt.Sprintf("Could not get User Token %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	result := responseToUserToken(out, state.BearerToken)
	tflog.Info(ctx, "read user token", map[string]any{
		"token_id": result.ID.ValueString(),
		"team_id":  result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *userTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a User Token is not supported",
		"Updating a User Token is not supported",
	)
}

func (r *userTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserToken
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUserToken(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting User Token",
			fmt.Sprintf("Could not delete User Token %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted user token", map[string]any{
		"token_id": state.ID.ValueString(),
		"team_id":  state.TeamID.ValueString(),
	})
}

func (r *userTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, tokenID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing User Token",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/token_id\" or \"token_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetUserToken(ctx, tokenID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing User Token",
			fmt.Sprintf("Could not get User Token %s, unexpected error: %s", tokenID, err),
		)
		return
	}

	result := responseToUserToken(out, types.StringNull())
	tflog.Info(ctx, "import user token", map[string]any{
		"token_id": result.ID.ValueString(),
		"team_id":  result.TeamID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
