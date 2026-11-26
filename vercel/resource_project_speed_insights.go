package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &projectSpeedInsightsResource{}
	_ resource.ResourceWithConfigure   = &projectSpeedInsightsResource{}
	_ resource.ResourceWithImportState = &projectSpeedInsightsResource{}
)

func newProjectSpeedInsightsResource() resource.Resource {
	return &projectSpeedInsightsResource{}
}

type projectSpeedInsightsResource struct {
	client *client.Client
}

func (r *projectSpeedInsightsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_speed_insights"
}

func (r *projectSpeedInsightsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectSpeedInsightsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Speed Insights resource.

Speed Insights collects Core Web Vitals and other performance metrics from a project's deployments. For more detail, see the [Speed Insights documentation](https://vercel.com/docs/speed-insights).

Enabling Speed Insights is available on Pro and Enterprise plans and may incur charges on your account.

Deleting this resource disables Speed Insights for the project.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Project to enable Speed Insights for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

type ProjectSpeedInsights struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
}

func responseToProjectSpeedInsights(out client.ProjectSpeedInsights) ProjectSpeedInsights {
	return ProjectSpeedInsights{
		ID:        types.StringValue(out.ProjectID),
		ProjectID: types.StringValue(out.ProjectID),
		TeamID:    toTeamID(out.TeamID),
	}
}

func (r *projectSpeedInsightsResource) enable(ctx context.Context, plan ProjectSpeedInsights, diags *diag.Diagnostics) (ProjectSpeedInsights, bool) {
	out, err := r.client.UpdateProjectSpeedInsights(ctx, client.ProjectSpeedInsights{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Enabled:   true,
	})
	if err != nil {
		diags.AddError(
			"Error enabling Project Speed Insights",
			"Could not enable Project Speed Insights, unexpected error: "+err.Error(),
		)
		return ProjectSpeedInsights{}, false
	}
	if !out.Enabled {
		diags.AddError(
			"Error enabling Project Speed Insights",
			fmt.Sprintf("Speed Insights is still disabled for project %s after enabling it", plan.ProjectID.ValueString()),
		)
		return ProjectSpeedInsights{}, false
	}
	return responseToProjectSpeedInsights(out), true
}

func (r *projectSpeedInsightsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectSpeedInsights
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, ok := r.enable(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	tflog.Info(ctx, "enabled project speed insights", map[string]any{"team_id": result.TeamID.ValueString(), "project_id": result.ProjectID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectSpeedInsightsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectSpeedInsights
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectSpeedInsights(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Speed Insights",
			fmt.Sprintf("Could not get Project Speed Insights %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	if !out.Enabled {
		resp.State.RemoveResource(ctx)
		return
	}

	result := responseToProjectSpeedInsights(out)
	tflog.Info(ctx, "read project speed insights", map[string]any{"team_id": result.TeamID.ValueString(), "project_id": result.ProjectID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

// Update is only reached when team_id moves from unknown to a known value,
// since every configurable attribute otherwise forces replacement.
func (r *projectSpeedInsightsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectSpeedInsights
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, ok := r.enable(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	tflog.Info(ctx, "updated project speed insights", map[string]any{"team_id": result.TeamID.ValueString(), "project_id": result.ProjectID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectSpeedInsightsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectSpeedInsights
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateProjectSpeedInsights(ctx, client.ProjectSpeedInsights{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		Enabled:   false,
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error disabling Project Speed Insights",
			fmt.Sprintf("Could not disable Project Speed Insights %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}
	tflog.Info(ctx, "disabled project speed insights", map[string]any{"team_id": state.TeamID.ValueString(), "project_id": state.ProjectID.ValueString()})
}

func (r *projectSpeedInsightsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Project Speed Insights",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetProjectSpeedInsights(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Speed Insights",
			fmt.Sprintf("Could not get Project Speed Insights %s %s, unexpected error: %s", teamID, projectID, err),
		)
		return
	}
	if !out.Enabled {
		resp.Diagnostics.AddError(
			"Cannot import disabled Project Speed Insights",
			fmt.Sprintf("Speed Insights is not enabled for project %s", projectID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, responseToProjectSpeedInsights(out))...)
}
