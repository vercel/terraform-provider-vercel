package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &accessGroupProjectResource{}
	_ resource.ResourceWithConfigure   = &accessGroupProjectResource{}
	_ resource.ResourceWithImportState = &accessGroupProjectResource{}
)

func newAccessGroupProjectResource() resource.Resource {
	return &accessGroupProjectResource{}
}

type accessGroupProjectResource struct {
	client *client.Client
}

func (r *accessGroupProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group_project"
}

func (r *accessGroupProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accessGroupProjectResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Access Group Project Resource.

An Access Group Project resource defines the relationship between a ` + "`vercel_access_group` and a `vercel_project`." + `

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/accounts/team-members-and-roles/access-groups).
`,
		Attributes: map[string]schema.Attribute{
			"access_group_id": schema.StringAttribute{
				Description:   "The ID of the Access Group.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the access group project should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The Project ID to assign to the access group.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				Description: "The project role to assign to the access group. Must be either `ADMIN`, `PROJECT_DEVELOPER`, or `PROJECT_VIEWER`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ADMIN", "PROJECT_DEVELOPER", "PROJECT_VIEWER"),
				},
			},
		},
	}
}

type AccessGroupProject struct {
	AccessGroupID types.String `tfsdk:"access_group_id"`
	TeamID        types.String `tfsdk:"team_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	Role          types.String `tfsdk:"role"`
}

func (r *accessGroupProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessGroupProject
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateAccessGroupProject(ctx, client.CreateAccessGroupProjectRequest{
		TeamID:        plan.TeamID.ValueString(),
		AccessGroupID: plan.AccessGroupID.ValueString(),
		ProjectID:     plan.ProjectID.ValueString(),
		Role:          plan.Role.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Access Group Project",
			fmt.Sprintf("Could not create Access Group Project %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.AccessGroupID.ValueString(),
				plan.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroupProject{
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		ProjectID:     types.StringValue(out.ProjectID),
		Role:          types.StringValue(out.Role),
	}

	tflog.Info(ctx, "created Access Group", map[string]interface{}{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"project_id":      result.ProjectID.ValueString(),
		"role":            result.Role.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessGroupProject
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAccessGroupProject(ctx, client.GetAccessGroupProjectRequest{
		AccessGroupID: state.AccessGroupID.ValueString(),
		TeamID:        state.TeamID.ValueString(),
		ProjectID:     state.ProjectID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group Project",
			fmt.Sprintf("Could not get Access Group Project %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.AccessGroupID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroupProject{
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		ProjectID:     types.StringValue(out.ProjectID),
		Role:          types.StringValue(out.Role),
	}
	tflog.Info(ctx, "read Access Group Project", map[string]interface{}{
		"team_id":         state.TeamID.ValueString(),
		"access_group_id": state.AccessGroupID.ValueString(),
		"project_id":      state.ProjectID.ValueString(),
		"role":            state.Role.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccessGroupProject
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateAccessGroupProject(ctx, client.UpdateAccessGroupProjectRequest{
		TeamID:        plan.TeamID.ValueString(),
		AccessGroupID: plan.AccessGroupID.ValueString(),
		ProjectID:     plan.ProjectID.ValueString(),
		Role:          plan.Role.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Access Group Project",
			fmt.Sprintf("Could not create Access Group Project %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.AccessGroupID.ValueString(),
				plan.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroupProject{
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		ProjectID:     types.StringValue(out.ProjectID),
		Role:          types.StringValue(out.Role),
	}

	tflog.Info(ctx, "updated Access Group Project", map[string]interface{}{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"project_id":      result.ProjectID.ValueString(),
		"role":            result.Role.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessGroupProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessGroupProject
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAccessGroupProject(ctx, client.DeleteAccessGroupProjectRequest{
		TeamID:        state.TeamID.ValueString(),
		AccessGroupID: state.AccessGroupID.ValueString(),
		ProjectID:     state.ProjectID.ValueString(),
	})

	if client.NotFound(err) {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Access Group Project",
			fmt.Sprintf(
				"Could not delete Access Group %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.AccessGroupID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Access Group", map[string]interface{}{
		"team_id":         state.TeamID.ValueString(),
		"access_group_id": state.AccessGroupID.ValueString(),
		"project_id":      state.ProjectID.ValueString(),
		"role":            state.Role.ValueString(),
	})
}

func (r *accessGroupProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, accessGroupID, projectID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Access Group",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/access_group_id/project_id\" or \"access_group_id/project_id\"", req.ID),
		)
	}

	out, err := r.client.GetAccessGroupProject(ctx, client.GetAccessGroupProjectRequest{
		TeamID:        teamID,
		AccessGroupID: accessGroupID,
		ProjectID:     projectID,
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group Project",
			fmt.Sprintf("Could not get Accesss Group %s %s %s, unexpected error: %s",
				teamID,
				accessGroupID,
				projectID,
				err,
			),
		)
		return
	}

	result := AccessGroupProject{
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		ProjectID:     types.StringValue(out.ProjectID),
		Role:          types.StringValue(out.Role),
	}

	tflog.Info(ctx, "import Access Group Project", map[string]interface{}{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"project_id":      result.ProjectID.ValueString(),
		"role":            result.Role.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
