package vercel

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

type resourceProjectEnvironmentVariableType struct{}

// GetSchema returns the schema information for a project environment variable resource.
func (r resourceProjectEnvironmentVariableType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a Project environment variable resource.

A Project environment variable resource defines an environment variable on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables).`,
		Attributes: map[string]tfsdk.Attribute{
			"target": {
				Required:    true,
				Description: "The environments that the environment variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
				Type: types.SetType{
					ElemType: types.StringType,
				},
			},
			"key": {
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Description:   "The name of the environment variable.",
				Type:          types.StringType,
			},
			"value": {
				Required:    true,
				Description: "The value of the environment variable.",
				Type:        types.StringType,
			},
			"git_branch": {
				Optional:    true,
				Description: "The git branch of the environment variable.",
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
				Description:   "The ID of the Vercel team.",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Type:          types.StringType,
			},
			"id": {
				Description:   "The ID of the environment variable.",
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown()},
				Computed:      true,
			},
		},
	}, nil
}

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceProjectEnvironmentVariableType) NewResource(_ context.Context, p provider.Provider) (resource.Resource, diag.Diagnostics) {
	return resourceProjectEnvironmentVariable{
		p: *(p.(*vercelProvider)),
	}, nil
}

type resourceProjectEnvironmentVariable struct {
	p vercelProvider
}

// Create will create a new project environment variable for a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r resourceProjectEnvironmentVariable) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var plan ProjectEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.p.client.GetProject(ctx, plan.ProjectID.Value, plan.TeamID.Value, false)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		resp.Diagnostics.AddError(
			"Error creating project environment variable",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	response, err := r.p.client.CreateEnvironmentVariable(ctx, plan.toCreateEnvironmentVariableRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment variable",
			"Could not create project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(response, plan)

	tflog.Trace(ctx, "created project environment variable", map[string]interface{}{
		"id":         result.ID.Value,
		"team_id":    result.TeamID.Value,
		"project_id": result.ProjectID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an environment variable of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r resourceProjectEnvironmentVariable) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetEnvironmentVariable(ctx, state.ProjectID.Value, state.TeamID.Value, state.ID.Value)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variable",
			fmt.Sprintf("Could not get project environment variable %s %s %s, unexpected error: %s",
				state.ID.Value,
				state.ProjectID.Value,
				state.TeamID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(out, state)
	tflog.Trace(ctx, "read project environment variable", map[string]interface{}{
		"id":         result.ID.Value,
		"team_id":    result.TeamID.Value,
		"project_id": result.ProjectID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the project environment variable of a Vercel project state.
func (r resourceProjectEnvironmentVariable) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.p.client.UpdateEnvironmentVariable(ctx, plan.toUpdateEnvironmentVariableRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project environment variable",
			"Could not update project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProjectEnvironmentVariable(response, plan)

	tflog.Trace(ctx, "updated project environment variable", map[string]interface{}{
		"id":         result.ID.Value,
		"team_id":    result.TeamID.Value,
		"project_id": result.ProjectID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel project environment variable.
func (r resourceProjectEnvironmentVariable) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.p.client.DeleteEnvironmentVariable(ctx, state.ProjectID.Value, state.TeamID.Value, state.ID.Value)
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project environment variable",
			fmt.Sprintf(
				"Could not delete project environment variable %s, unexpected error: %s",
				state.ID.Value,
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "deleted project environment variable", map[string]interface{}{
		"id":         state.ID.Value,
		"team_id":    state.TeamID.Value,
		"project_id": state.ProjectID.Value,
	})
}
