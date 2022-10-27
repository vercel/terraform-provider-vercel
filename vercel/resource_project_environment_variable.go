package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

func newProjectEnvironmentVariableResource() resource.Resource {
	return &projectEnvironmentVariableResource{}
}

type projectEnvironmentVariableResource struct {
	client *client.Client
}

func (r *projectEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variable"
}

func (r *projectEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// GetSchema returns the schema information for a project environment variable resource.
func (r *projectEnvironmentVariableResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a Project Environment Variable resource.

A Project Environment Variable resource defines an Environment Variable on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables).

~> Terraform currently provides both a standalone Project Environment Variable resource (a single Environment Variable), and a Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
`,
		Attributes: map[string]tfsdk.Attribute{
			"target": {
				Required:    true,
				Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
				Type: types.SetType{
					ElemType: types.StringType,
				},
			},
			"key": {
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Description:   "The name of the Environment Variable.",
				Type:          types.StringType,
			},
			"value": {
				Required:    true,
				Description: "The value of the Environment Variable.",
				Type:        types.StringType,
				Sensitive:   true,
			},
			"git_branch": {
				Optional:    true,
				Description: "The git branch of the Environment Variable.",
				Type:        types.StringType,
			},
			"project_id": {
				Required:      true,
				Description:   "The ID of the Vercel project.",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Type:          types.StringType,
			},
			"team_id": {
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team.",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace(), resource.UseStateForUnknown()},
				Type:          types.StringType,
			},
			"id": {
				Description:   "The ID of the Environment Variable.",
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown()},
				Computed:      true,
			},
		},
	}, nil
}

// Create will create a new project environment variable for a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r *projectEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), false)
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project environment variable",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	response, err := r.client.CreateEnvironmentVariable(ctx, plan.toCreateEnvironmentVariableRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment variable",
			"Could not create project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(response, plan.ProjectID)

	tflog.Trace(ctx, "created project environment variable", map[string]interface{}{
		"id":         result.ID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an environment variable of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetEnvironmentVariable(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variable",
			fmt.Sprintf("Could not get project environment variable %s %s %s, unexpected error: %s",
				state.ID.ValueString(),
				state.ProjectID.ValueString(),
				state.TeamID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(out, state.ProjectID)
	tflog.Trace(ctx, "read project environment variable", map[string]interface{}{
		"id":         result.ID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the project environment variable of a Vercel project state.
func (r *projectEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.UpdateEnvironmentVariable(ctx, plan.toUpdateEnvironmentVariableRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project environment variable",
			"Could not update project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(response, plan.ProjectID)

	tflog.Trace(ctx, "updated project environment variable", map[string]interface{}{
		"id":         result.ID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel project environment variable.
func (r *projectEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEnvironmentVariable(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project environment variable",
			fmt.Sprintf(
				"Could not delete project environment variable %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "deleted project environment variable", map[string]interface{}{
		"id":         state.ID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

// splitID is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitProjectEnvironmentVariableID(id string) (teamID, projectID, envID string, ok bool) {
	attributes := strings.Split(id, "/")
	if len(attributes) == 3 {
		return attributes[0], attributes[1], attributes[2], true
	}
	if len(attributes) == 2 {
		return "", attributes[0], attributes[1], true
	}

	return "", "", "", false
}

// ImportState takes an identifier and reads all the project environment variable information from the Vercel API.
// The results are then stored in terraform state.
func (r *projectEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, envID, ok := splitProjectEnvironmentVariableID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project environment variable",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/env_id\" or \"project_id/env_id\"", req.ID),
		)
	}

	out, err := r.client.GetEnvironmentVariable(ctx, projectID, teamID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variable",
			fmt.Sprintf("Could not get project environment variable %s %s %s, unexpected error: %s",
				teamID,
				projectID,
				envID,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(out, types.StringValue(projectID))
	tflog.Trace(ctx, "imported project environment variable", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"env_id":     result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
