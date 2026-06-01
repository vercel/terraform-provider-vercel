package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &accessGroupMemberResource{}
	_ resource.ResourceWithConfigure   = &accessGroupMemberResource{}
	_ resource.ResourceWithImportState = &accessGroupMemberResource{}
)

func newAccessGroupMemberResource() resource.Resource {
	return &accessGroupMemberResource{}
}

type accessGroupMemberResource struct {
	client *client.Client
}

func (r *accessGroupMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group_member"
}

func (r *accessGroupMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accessGroupMemberResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Access Group Member Resource.

An Access Group Member resource defines the membership of a Vercel user (by their user ID) in a ` + "`vercel_access_group`." + `

~> Access group membership can also be managed through the ` + "`access_groups`" + ` attribute of the ` + "`vercel_team_member`" + ` resource. Do not manage the same user's membership of the same access group with both resources, as they will conflict and produce a perpetual diff.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/accounts/team-members-and-roles/access-groups).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource. Format: access_group_id/user_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_group_id": schema.StringAttribute{
				Description:   "The ID of the Access Group.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the access group member should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the user to add to the access group.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type AccessGroupMember struct {
	ID            types.String `tfsdk:"id"`
	AccessGroupID types.String `tfsdk:"access_group_id"`
	TeamID        types.String `tfsdk:"team_id"`
	UserID        types.String `tfsdk:"user_id"`
}

func (r *accessGroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessGroupMember
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateAccessGroupMember(ctx, client.CreateAccessGroupMemberRequest{
		TeamID:        plan.TeamID.ValueString(),
		AccessGroupID: plan.AccessGroupID.ValueString(),
		UserID:        plan.UserID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Access Group Member",
			fmt.Sprintf("Could not create Access Group Member %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.AccessGroupID.ValueString(),
				plan.UserID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroupMember{
		ID:            types.StringValue(out.AccessGroupID + "/" + out.UserID),
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		UserID:        types.StringValue(out.UserID),
	}

	tflog.Info(ctx, "created Access Group Member", map[string]any{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"user_id":         result.UserID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessGroupMember
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAccessGroupMember(ctx, client.GetAccessGroupMemberRequest{
		AccessGroupID: state.AccessGroupID.ValueString(),
		TeamID:        state.TeamID.ValueString(),
		UserID:        state.UserID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group Member",
			fmt.Sprintf("Could not get Access Group Member %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.AccessGroupID.ValueString(),
				state.UserID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroupMember{
		ID:            types.StringValue(out.AccessGroupID + "/" + out.UserID),
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		UserID:        types.StringValue(out.UserID),
	}
	tflog.Info(ctx, "read Access Group Member", map[string]any{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"user_id":         result.UserID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update is a no-op. All attributes of this resource force replacement, so an
// in-place update should never be planned; this method exists only to satisfy
// the resource.Resource interface.
func (r *accessGroupMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccessGroupMember
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *accessGroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessGroupMember
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAccessGroupMember(ctx, client.DeleteAccessGroupMemberRequest{
		TeamID:        state.TeamID.ValueString(),
		AccessGroupID: state.AccessGroupID.ValueString(),
		UserID:        state.UserID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Access Group Member",
			fmt.Sprintf(
				"Could not delete Access Group Member %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.AccessGroupID.ValueString(),
				state.UserID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Access Group Member", map[string]any{
		"team_id":         state.TeamID.ValueString(),
		"access_group_id": state.AccessGroupID.ValueString(),
		"user_id":         state.UserID.ValueString(),
	})
}

func (r *accessGroupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, accessGroupID, userID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Access Group Member",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/access_group_id/user_id\" or \"access_group_id/user_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetAccessGroupMember(ctx, client.GetAccessGroupMemberRequest{
		TeamID:        teamID,
		AccessGroupID: accessGroupID,
		UserID:        userID,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group Member",
			fmt.Sprintf("Could not get Access Group Member %s %s %s, unexpected error: %s",
				teamID,
				accessGroupID,
				userID,
				err,
			),
		)
		return
	}

	result := AccessGroupMember{
		ID:            types.StringValue(out.AccessGroupID + "/" + out.UserID),
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		UserID:        types.StringValue(out.UserID),
	}

	tflog.Info(ctx, "import Access Group Member", map[string]any{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"user_id":         result.UserID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
