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
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectFunctionCPUResource{}
	_ resource.ResourceWithConfigure   = &projectFunctionCPUResource{}
	_ resource.ResourceWithImportState = &projectFunctionCPUResource{}
)

func newProjectFunctionCPUResource() resource.Resource {
	return &projectFunctionCPUResource{}
}

type projectFunctionCPUResource struct {
	client *client.Client
}

func (r *projectFunctionCPUResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_function_cpu"
}

func (r *projectFunctionCPUResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectFunctionCPUResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		DeprecationMessage: "This resource is deprecated and no longer works. Please use the `vercel_project` resource and its `resource_config` attribute instead.",
		Description: `
~> This resource has been deprecated and no longer works. Please use the ` + "`vercel_project`" + ` resource and its ` + "`resource_config`" + ` attribute instead.

Provides a Function CPU resource for a Project.

This controls the maximum amount of CPU utilization your Serverless Functions can use while executing. Standard is optimal for most frontend workloads. You can override this per function using the vercel.json file.

A new Deployment is required for your changes to take effect.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the resource.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project to adjust the CPU for.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"cpu": schema.StringAttribute{
				Description: "The amount of CPU available to your Serverless Functions. Should be one of 'basic' (0.6vCPU), 'standard' (1vCPU) or 'performance' (1.7vCPUs).",
				Required:    true,
			},
		},
	}
}

type ProjectFunctionCPU struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	CPU       types.String `tfsdk:"cpu"`
}

func convertResponseToProjectFunctionCPU(response client.ProjectFunctionCPU) ProjectFunctionCPU {
	return ProjectFunctionCPU{
		ID:        types.StringValue(response.ProjectID),
		TeamID:    toTeamID(response.TeamID),
		ProjectID: types.StringValue(response.ProjectID),
		CPU:       types.StringPointerValue(response.CPU),
	}
}

func (r *projectFunctionCPUResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("`vercel_project_function_cpu` resource deprecated", "use `vercel_project` resource and its `resource_config` attribute instead"),
	)

	var plan ProjectFunctionCPU
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateProjectFunctionCPU(ctx, client.ProjectFunctionCPURequest{
		ProjectID: plan.ProjectID.ValueString(),
		CPU:       plan.CPU.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project Function CPU",
			"Could not update function CPU, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectFunctionCPU(out)
	tflog.Info(ctx, "created project function cpu", map[string]interface{}{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
		"cpu":        result.CPU.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectFunctionCPUResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("`vercel_project_function_cpu` resource deprecated", "use `vercel_project` resource and its `resource_config` attribute instead"),
	)
	var state ProjectFunctionCPU
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectFunctionCPU(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project Function CPU",
			fmt.Sprintf("Could not get Project Function CPU %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectFunctionCPU(out)
	tflog.Info(ctx, "read project function cpu", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectFunctionCPUResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("`vercel_project_function_cpu` resource deprecated", "use `vercel_project` resource and its `resource_config` attribute instead"),
	)
	var plan ProjectFunctionCPU
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateProjectFunctionCPU(ctx, client.ProjectFunctionCPURequest{
		ProjectID: plan.ProjectID.ValueString(),
		CPU:       plan.CPU.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project Function CPU",
			"Could not update function CPU, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectFunctionCPU(out)
	tflog.Info(ctx, "created project function cpu", map[string]interface{}{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
		"cpu":        result.CPU.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectFunctionCPUResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("`vercel_project_function_cpu` resource deprecated", "use `vercel_project` resource and its `resource_config` attribute instead"),
	)
	tflog.Info(ctx, "deleted project function cpu", map[string]interface{}{})
}

func (r *projectFunctionCPUResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Project Function CPU",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetProjectFunctionCPU(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Function CPU",
			fmt.Sprintf("Could not get Project Function CPU %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectFunctionCPU(out)
	tflog.Info(ctx, "import project function cpu", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
