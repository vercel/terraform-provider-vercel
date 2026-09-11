package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &oauthAppPermissionsResource{}
	_ resource.ResourceWithConfigure   = &oauthAppPermissionsResource{}
	_ resource.ResourceWithImportState = &oauthAppPermissionsResource{}
)

func newOAuthAppPermissionsResource() resource.Resource {
	return &oauthAppPermissionsResource{}
}

type oauthAppPermissionsResource struct {
	client *client.Client
}

func (r *oauthAppPermissionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_app_permissions"
}

func (r *oauthAppPermissionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *oauthAppPermissionsResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Grants Vercel REST API permissions to a ` + "`vercel_oauth_app`" + ` ([Sign in with Vercel](https://vercel.com/docs/sign-in-with-vercel)) — what the app's tokens may do on the platform on the signed-in user's behalf (e.g. ` + "`read:team`" + `, ` + "`read:project`" + `, ` + "`read-write:deployment`" + `). Users review and consent to these on the sign-in consent page.

At most ONE ` + "`vercel_oauth_app_permissions`" + ` should exist per OAuth App: it owns the app's whole permission set, and destroying it revokes all grants.

~> Managing OAuth App permissions requires the Owner role on the team.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The unique identifier of this permission grant (the OAuth App's client ID).",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"oauth_app_id": schema.StringAttribute{
				Description:   "The client ID of the OAuth App (`cl_...`) to grant permissions to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the OAuth App exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"permissions": schema.SetAttribute{
				Description: "The Vercel REST API permissions granted to the app's tokens (e.g. `read:team`, `read:project`, `read-write:project`, `read:deployment`, `read-write:deployment`).",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

type OAuthAppPermissions struct {
	ID          types.String `tfsdk:"id"`
	OAuthAppID  types.String `tfsdk:"oauth_app_id"`
	TeamID      types.String `tfsdk:"team_id"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func responseToOAuthAppPermissions(ctx context.Context, out client.OAuthApp) (OAuthAppPermissions, error) {
	permissions, diags := types.SetValueFrom(ctx, types.StringType, out.Permissions)
	if diags.HasError() {
		return OAuthAppPermissions{}, fmt.Errorf("could not convert permissions: %s", diags.Errors())
	}
	return OAuthAppPermissions{
		ID:          types.StringValue(out.ClientID),
		OAuthAppID:  types.StringValue(out.ClientID),
		TeamID:      types.StringValue(out.TeamID),
		Permissions: permissions,
	}, nil
}

func (r *oauthAppPermissionsResource) apply(ctx context.Context, plan OAuthAppPermissions) (OAuthAppPermissions, error) {
	var permissions []string
	diags := plan.Permissions.ElementsAs(ctx, &permissions, false)
	if diags.HasError() {
		return OAuthAppPermissions{}, fmt.Errorf("could not read permissions from plan: %s", diags.Errors())
	}
	out, err := r.client.UpdateOAuthAppPermissions(ctx, plan.OAuthAppID.ValueString(), permissions, plan.TeamID.ValueString())
	if err != nil {
		return OAuthAppPermissions{}, err
	}
	return responseToOAuthAppPermissions(ctx, out)
}

func (r *oauthAppPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuthAppPermissions
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.apply(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating OAuth App Permissions",
			"Could not grant OAuth App permissions, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Info(ctx, "granted oauth app permissions", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.OAuthAppID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *oauthAppPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OAuthAppPermissions
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetOAuthApp(ctx, state.OAuthAppID.ValueString(), state.TeamID.ValueString())
	if client.OAuthAppNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading OAuth App Permissions",
			fmt.Sprintf("Could not get OAuth App %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.OAuthAppID.ValueString(),
				err,
			),
		)
		return
	}
	// No grants on the app means this resource no longer exists.
	if len(out.Permissions) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	result, err := responseToOAuthAppPermissions(ctx, out)
	if err != nil {
		resp.Diagnostics.AddError("Error reading OAuth App Permissions", err.Error())
		return
	}
	tflog.Info(ctx, "read oauth app permissions", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.OAuthAppID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *oauthAppPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OAuthAppPermissions
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.apply(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating OAuth App Permissions",
			fmt.Sprintf("Could not update OAuth App permissions %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.OAuthAppID.ValueString(),
				err,
			),
		)
		return
	}
	tflog.Info(ctx, "updated oauth app permissions", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.OAuthAppID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *oauthAppPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OAuthAppPermissions
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Revoke every grant. A vanished app means there is nothing to revoke.
	_, err := r.client.UpdateOAuthAppPermissions(ctx, state.OAuthAppID.ValueString(), []string{}, state.TeamID.ValueString())
	if client.OAuthAppNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting OAuth App Permissions",
			fmt.Sprintf("Could not revoke OAuth App permissions %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.OAuthAppID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "revoked oauth app permissions", map[string]any{
		"team_id":   state.TeamID.ValueString(),
		"client_id": state.OAuthAppID.ValueString(),
	})
}

func (r *oauthAppPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, clientID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing OAuth App Permissions",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/client_id\" or \"client_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetOAuthApp(ctx, clientID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing OAuth App Permissions",
			fmt.Sprintf("Could not get OAuth App %s %s, unexpected error: %s",
				teamID,
				clientID,
				err,
			),
		)
		return
	}
	if len(out.Permissions) == 0 {
		resp.Diagnostics.AddError(
			"Error importing OAuth App Permissions",
			fmt.Sprintf("OAuth App %s has no permissions granted — there is nothing to import.", clientID),
		)
		return
	}

	result, err := responseToOAuthAppPermissions(ctx, out)
	if err != nil {
		resp.Diagnostics.AddError("Error importing OAuth App Permissions", err.Error())
		return
	}
	tflog.Info(ctx, "imported oauth app permissions", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.OAuthAppID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
