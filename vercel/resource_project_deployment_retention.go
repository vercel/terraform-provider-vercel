package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &projectDeploymentRetentionResource{}
	_ resource.ResourceWithConfigure   = &projectDeploymentRetentionResource{}
	_ resource.ResourceWithImportState = &projectDeploymentRetentionResource{}
)

func newProjectDeploymentRetentionResource() resource.Resource {
	return &projectDeploymentRetentionResource{}
}

type projectDeploymentRetentionResource struct {
	client *client.Client
}

func (r *projectDeploymentRetentionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_deployment_retention"
}

func (r *projectDeploymentRetentionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a project deployment retention resource.
func (r *projectDeploymentRetentionResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Deployment Retention resource.

A Project Deployment Retention resource defines an Deployment Retention on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/security/deployment-retention).
`,
		Attributes: map[string]schema.Attribute{
			"expiration_preview": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The retention period for preview deployments. Should be one of '1m', '2m', '3m', '6m', '1y', 'unlimited'.",
				Default:     stringdefault.StaticString("unlimited"),
				Validators: []validator.String{
					stringvalidator.OneOf("1m", "2m", "3m", "6m", "1y", "unlimited"),
				},
			},
			"expiration_production": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The retention period for production deployments. Should be one of '1m', '2m', '3m', '6m', '1y', 'unlimited'.",
				Default:     stringdefault.StaticString("unlimited"),
				Validators: []validator.String{
					stringvalidator.OneOf("1m", "2m", "3m", "6m", "1y", "unlimited"),
				},
			},
			"expiration_canceled": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The retention period for canceled deployments. Should be one of '1m', '2m', '3m', '6m', '1y', 'unlimited'.",
				Default:     stringdefault.StaticString("unlimited"),
				Validators: []validator.String{
					stringvalidator.OneOf("1m", "2m", "3m", "6m", "1y", "unlimited"),
				},
			},
			"expiration_errored": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The retention period for errored deployments. Should be one of '1m', '2m', '3m', '6m', '1y', 'unlimited'.",
				Default:     stringdefault.StaticString("unlimited"),
				Validators: []validator.String{
					stringvalidator.OneOf("1m", "2m", "3m", "6m", "1y", "unlimited"),
				},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project for the retention policy",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// ProjectDeploymentRetention reflects the state terraform stores internally for a project deployment retention.
type ProjectDeploymentRetention struct {
	ExpirationPreview    types.String `tfsdk:"expiration_preview"`
	ExpirationProduction types.String `tfsdk:"expiration_production"`
	ExpirationCanceled   types.String `tfsdk:"expiration_canceled"`
	ExpirationErrored    types.String `tfsdk:"expiration_errored"`
	ProjectID            types.String `tfsdk:"project_id"`
	TeamID               types.String `tfsdk:"team_id"`
}

func (e *ProjectDeploymentRetention) toUpdateDeploymentRetentionRequest() client.UpdateDeploymentRetentionRequest {
	return client.UpdateDeploymentRetentionRequest{
		DeploymentRetention: client.DeploymentRetentionRequest{
			ExpirationPreview:    e.ExpirationPreview.ValueString(),
			ExpirationProduction: e.ExpirationProduction.ValueString(),
			ExpirationCanceled:   e.ExpirationCanceled.ValueString(),
			ExpirationErrored:    e.ExpirationErrored.ValueString(),
		},
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
	}
}

// convertResponseToProjectDeploymentRetention is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToProjectDeploymentRetention(response client.DeploymentExpiration, projectID types.String, teamID types.String) ProjectDeploymentRetention {
	return ProjectDeploymentRetention{
		ExpirationPreview:    types.StringValue(client.DeploymentRetentionDaysToString[response.ExpirationPreview]),
		ExpirationProduction: types.StringValue(client.DeploymentRetentionDaysToString[response.ExpirationProduction]),
		ExpirationCanceled:   types.StringValue(client.DeploymentRetentionDaysToString[response.ExpirationCanceled]),
		ExpirationErrored:    types.StringValue(client.DeploymentRetentionDaysToString[response.ExpirationErrored]),
		TeamID:               teamID,
		ProjectID:            projectID,
	}
}

// Create will create a new project deployment retention for a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r *projectDeploymentRetentionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectDeploymentRetention
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project deployment retention",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project deployment retention",
			"Error reading project information, unexpected error: "+err.Error(),
		)
		return
	}

	response, err := r.client.UpdateDeploymentRetention(ctx, plan.toUpdateDeploymentRetentionRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project deployment retention",
			"Could not create project deployment retention, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectDeploymentRetention(response, plan.ProjectID, plan.TeamID)

	tflog.Info(ctx, "created project deployment retention", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an deployment retention of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectDeploymentRetentionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectDeploymentRetention
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetDeploymentRetention(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project deployment retention",
			fmt.Sprintf("Could not get project deployment retention %s %s, unexpected error: %s",
				state.ProjectID.ValueString(),
				state.TeamID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDeploymentRetention(out, state.ProjectID, state.TeamID)
	tflog.Info(ctx, "read project deployment retention", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel project deployment retention.
func (r *projectDeploymentRetentionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectDeploymentRetention
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDeploymentRetention(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project deployment retention",
			fmt.Sprintf(
				"Could not delete project deployment retention %s, unexpected error: %s",
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted project deployment retention", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

// Update updates the project deployment retention of a Vercel project state.
func (r *projectDeploymentRetentionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectDeploymentRetention
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.UpdateDeploymentRetention(ctx, plan.toUpdateDeploymentRetentionRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project deployment retention",
			"Could not update project deployment retention, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectDeploymentRetention(response, plan.ProjectID, plan.TeamID)

	tflog.Info(ctx, "updated project deployment retention", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ImportState takes an identifier and reads all the project deployment retention information from the Vercel API.
// The results are then stored in terraform state.
func (r *projectDeploymentRetentionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project deployment retention",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetDeploymentRetention(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project deployment retention",
			fmt.Sprintf("Could not get project deployment retention %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDeploymentRetention(out, types.StringValue(projectID), types.StringValue(teamID))
	tflog.Info(ctx, "imported project deployment retention", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
