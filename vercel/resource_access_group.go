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
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &accessGroupResource{}
	_ resource.ResourceWithConfigure   = &accessGroupResource{}
	_ resource.ResourceWithImportState = &accessGroupResource{}
)

func newAccessGroupResource() resource.Resource {
	return &accessGroupResource{}
}

type accessGroupResource struct {
	client *client.Client
}

func (r *accessGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group"
}

func (r *accessGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accessGroupResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Access Group Resource.

Access Groups provide a way to manage groups of Vercel users across projects on your team. They are a set of project role assignations, a combination of Vercel users and the projects they work on.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/accounts/team-members-and-roles/access-groups).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Access Group.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team the Access Group should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Access Group",
				Required:    true,
			},
		},
	}
}

type AccessGroup struct {
	ID     types.String `tfsdk:"id"`
	TeamID types.String `tfsdk:"team_id"`
	Name   types.String `tfsdk:"name"`
}

func (r *accessGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateAccessGroup(ctx, client.CreateAccessGroupRequest{
		TeamID: plan.TeamID.ValueString(),
		Name:   plan.Name.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Access Group",
			"Could not create Access Group, unexpected error: "+err.Error(),
		)
		return
	}
	result := AccessGroup{
		ID:     types.StringValue(out.ID),
		Name:   types.StringValue(out.Name),
		TeamID: types.StringValue(out.TeamID),
	}

	tflog.Info(ctx, "created Access Group", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"id":      result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAccessGroup(ctx, client.GetAccessGroupRequest{
		AccessGroupID: state.ID.ValueString(),
		TeamID:        state.TeamID.ValueString(),
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group",
			fmt.Sprintf("Could not get Access Group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroup{
		ID:     types.StringValue(out.ID),
		TeamID: toTeamID(out.TeamID),
		Name:   types.StringValue(out.Name),
	}

	tflog.Info(ctx, "read Access Group", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"id":      result.ID.ValueString(),
		"name":    result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccessGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateAccessGroup(ctx, client.UpdateAccessGroupRequest{
		TeamID:        plan.TeamID.ValueString(),
		AccessGroupID: plan.ID.ValueString(),
		Name:          plan.Name.ValueString(),
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Acccess Group",
			fmt.Sprintf("Could not update Access Group %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroup{
		ID:     types.StringValue(out.ID),
		TeamID: types.StringValue(out.TeamID),
		Name:   types.StringValue(out.Name),
	}

	tflog.Trace(ctx, "update Access Group", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"id":      result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAccessGroup(ctx, client.DeleteAccessGroupRequest{
		TeamID:        state.TeamID.ValueString(),
		AccessGroupID: state.ID.ValueString(),
	})

	if client.NotFound(err) {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Access Group",
			fmt.Sprintf(
				"Could not delete Access Group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Access Group", map[string]interface{}{
		"team_id": state.TeamID.ValueString(),
		"id":      state.ID.ValueString(),
	})
}

func (r *accessGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Access Group",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/access_group_id\" or \"access_group_id\"", req.ID),
		)
	}

	out, err := r.client.GetAccessGroup(ctx, client.GetAccessGroupRequest{
		TeamID:        teamID,
		AccessGroupID: id,
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group",
			fmt.Sprintf("Could not get Accesss Group %s %s, unexpected error: %s",
				teamID,
				id,
				err,
			),
		)
		return
	}

	result := AccessGroup{
		ID:     types.StringValue(out.ID),
		TeamID: toTeamID(out.TeamID),
		Name:   types.StringValue(out.Name),
	}

	tflog.Info(ctx, "import Access Group", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"id":      result.ID.ValueString(),
		"name":    result.Name.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
