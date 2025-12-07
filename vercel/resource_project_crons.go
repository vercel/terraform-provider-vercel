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
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Compile-time assertions to ensure the implementation conforms to the expected interfaces.
var (
	_ resource.Resource                = &projectCronsResource{}
	_ resource.ResourceWithConfigure   = &projectCronsResource{}
	_ resource.ResourceWithImportState = &projectCronsResource{}
)

func newProjectCronsResource() resource.Resource {
	return &projectCronsResource{}
}

type projectCronsResource struct {
	client *client.Client
}

func (r *projectCronsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_crons"
}

func (r *projectCronsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cli, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = cli
}

func (r *projectCronsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "\nProvides a Project Crons resource.\n\nThe resource toggles whether crons are enabled for a Vercel project.\n",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project to toggle crons for.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether crons are enabled for the project.",
			},
		},
	}
}

// ProjectCrons mirrors the Terraform state for the resource.
type ProjectCrons struct {
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
}

// mapResponseToProjectCrons converts the API response into the internal ProjectCrons model.
func mapResponseToProjectCrons(out client.ProjectCrons) ProjectCrons {
	return ProjectCrons{
		ProjectID: types.StringValue(out.ProjectID),
		TeamID:    toTeamID(out.TeamID),
		Enabled:   types.BoolValue(out.Enabled),
	}
}

func (r *projectCronsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectCrons
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure the project exists – this provides a friendly error message if the ID is wrong.
	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project crons",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to configure.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project crons",
			"Error reading project information, unexpected error: "+err.Error(),
		)
		return
	}

	out, err := r.client.UpdateProjectCrons(ctx, client.ProjectCrons{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project crons",
			"Could not create project crons, unexpected error: "+err.Error(),
		)
		return
	}

	result := mapResponseToProjectCrons(out)
	tflog.Info(ctx, "created project crons", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCronsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectCrons
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectCrons(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project crons",
			fmt.Sprintf("Could not get project crons %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	result := mapResponseToProjectCrons(out)
	tflog.Info(ctx, "read project crons", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCronsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectCrons
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateProjectCrons(ctx, client.ProjectCrons{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project crons",
			fmt.Sprintf("Could not update project crons %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), err),
		)
		return
	}

	result := mapResponseToProjectCrons(out)
	tflog.Trace(ctx, "updated project crons", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCronsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectCrons
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Disable crons on deletion to align with existing boolean-toggle resources (e.g. attack_challenge_mode).
	_, err := r.client.UpdateProjectCrons(ctx, client.ProjectCrons{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		Enabled:   false,
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project crons",
			fmt.Sprintf("Could not delete project crons %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted project crons", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

func (r *projectCronsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project crons",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetProjectCrons(ctx, projectID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project crons",
			fmt.Sprintf("Could not get project crons %s %s, unexpected error: %s", teamID, projectID, err),
		)
		return
	}

	result := mapResponseToProjectCrons(out)
	tflog.Info(ctx, "imported project crons", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
