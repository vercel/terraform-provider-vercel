package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectFunctionMaxDurationResource{}
	_ resource.ResourceWithConfigure   = &projectFunctionMaxDurationResource{}
	_ resource.ResourceWithImportState = &projectFunctionMaxDurationResource{}
)

func newProjectFunctionMaxDurationResource() resource.Resource {
	return &projectFunctionMaxDurationResource{}
}

type projectFunctionMaxDurationResource struct {
	client *client.Client
}

func (r *projectFunctionMaxDurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_function_max_duration"
}

func (r *projectFunctionMaxDurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for an alias resource.
func (r *projectFunctionMaxDurationResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a Function Max Duration resource for a Project.

This controls the default maximum duration of your Serverless Functions can use while executing. 10s is recommended for most workloads. Can be configured from 1 to 900 seconds (plan limits apply). You can override this per function using the vercel.json file.

A new Deployment is required for your changes to take effect.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the resource.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project to adjust the max duration for.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"max_duration": schema.Int64Attribute{
				Description: "The default max duration for your Serverless Functions. Must be between 1 and 900 (plan limits apply)",
				Required:    true,
				Validators: []validator.Int64{
					int64GreaterThan(1),
					int64LessThan(900),
				},
			},
		},
	}
}

type ProjectFunctionMaxDuration struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	TeamID      types.String `tfsdk:"team_id"`
	MaxDuration types.Int64  `tfsdk:"max_duration"`
}

func convertResponseToProjectFunctionMaxDuration(response client.ProjectFunctionMaxDuration) ProjectFunctionMaxDuration {
	return ProjectFunctionMaxDuration{
		ID:          types.StringValue(response.ProjectID),
		TeamID:      toTeamID(response.TeamID),
		ProjectID:   types.StringValue(response.ProjectID),
		MaxDuration: types.Int64PointerValue(response.MaxDuration),
	}
}

func (r *projectFunctionMaxDurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectFunctionMaxDuration
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateProjectFunctionMaxDuration(ctx, client.ProjectFunctionMaxDurationRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		TeamID:      plan.TeamID.ValueString(),
		MaxDuration: plan.MaxDuration.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project Function max duration",
			"Could not update function max duration, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectFunctionMaxDuration(out)
	tflog.Info(ctx, "created project function max duration", map[string]interface{}{
		"team_id":      plan.TeamID.ValueString(),
		"project_id":   plan.ProjectID.ValueString(),
		"max_duration": result.MaxDuration.ValueInt64(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectFunctionMaxDurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectFunctionMaxDuration
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectFunctionMaxDuration(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project Function max duration",
			fmt.Sprintf("Could not get Project Function max duration %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectFunctionMaxDuration(out)
	tflog.Info(ctx, "read project function max duration", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectFunctionMaxDurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectFunctionMaxDuration
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateProjectFunctionMaxDuration(ctx, client.ProjectFunctionMaxDurationRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		TeamID:      plan.TeamID.ValueString(),
		MaxDuration: plan.MaxDuration.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project Function max duration",
			"Could not update function max duration, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectFunctionMaxDuration(out)
	tflog.Info(ctx, "created project function max duration", map[string]interface{}{
		"team_id":      plan.TeamID.ValueString(),
		"project_id":   plan.ProjectID.ValueString(),
		"max_duration": result.MaxDuration.ValueInt64(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectFunctionMaxDurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "deleted project function max duration", map[string]interface{}{})
}

func (r *projectFunctionMaxDurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Project Function Max Duration",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetProjectFunctionMaxDuration(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Function max duration",
			fmt.Sprintf("Could not get Project Function Max Duration %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectFunctionMaxDuration(out)
	tflog.Info(ctx, "import project function max duration", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
