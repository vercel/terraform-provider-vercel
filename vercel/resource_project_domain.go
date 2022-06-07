package vercel

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

type resourceProjectDomainType struct{}

// GetSchema returns the schema information for a deployment resource.
func (r resourceProjectDomainType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a Project Domain resource.

A Project Domain is used to associate a domain name with a ` + "`vercel_project`." + `

By default, Project Domains will be automatically applied to any ` + "`production` deployments.",
		Attributes: map[string]tfsdk.Attribute{
			"project_id": {
				Description:   "The project ID to add the deployment to.",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"team_id": {
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
				Description:   "The ID of the team the project exists under.",
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"domain": {
				Description:   "The domain name to associate with the project.",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"redirect": {
				Description: "The domain name that serves as a target destination for redirects.",
				Optional:    true,
				Type:        types.StringType,
			},
			"redirect_status_code": {
				Description: "The HTTP status code to use when serving as a redirect.",
				Optional:    true,
				Type:        types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					int64OneOf(301, 302, 307, 308),
				},
			},
			"git_branch": {
				Description: "Git branch to link to the project domain. Deployments from this git branch will be assigned the domain name.",
				Optional:    true,
				Type:        types.StringType,
			},
		},
	}, nil
}

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceProjectDomainType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceProjectDomain{
		p: *(p.(*provider)),
	}, nil
}

type resourceProjectDomain struct {
	p provider
}

// Create will create a project domain within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r resourceProjectDomain) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var plan ProjectDomain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.p.client.GetProject(ctx, plan.ProjectID.Value, plan.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		resp.Diagnostics.AddError(
			"Error creating project domain",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	out, err := r.p.client.CreateProjectDomain(ctx, plan.ProjectID.Value, plan.TeamID.Value, plan.toCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding domain to project",
			fmt.Sprintf(
				"Could not add domain %s to project %s, unexpected error: %s",
				plan.Domain.Value,
				plan.ProjectID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDomain(out, plan.TeamID)
	tflog.Trace(ctx, "added domain to project", map[string]interface{}{
		"project_id": result.ProjectID.Value,
		"domain":     result.Domain.Value,
		"team_id":    result.TeamID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a project domain from the vercel API and provide terraform with information about it.
// It is called by the provider whenever data source values should be read to update state.
func (r resourceProjectDomain) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state ProjectDomain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetProjectDomain(ctx, state.ProjectID.Value, state.Domain.Value, state.TeamID.Value)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project domain",
			fmt.Sprintf("Could not get domain %s for project %s, unexpected error: %s",
				state.Domain.Value,
				state.ProjectID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDomain(out, state.TeamID)
	tflog.Trace(ctx, "read project domain", map[string]interface{}{
		"project_id": result.ProjectID.Value,
		"domain":     result.Domain.Value,
		"team_id":    result.TeamID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update will update a project domain via the vercel API.
func (r resourceProjectDomain) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan ProjectDomain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.UpdateProjectDomain(
		ctx,
		plan.ProjectID.Value,
		plan.Domain.Value,
		plan.TeamID.Value,
		plan.toUpdateRequest(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project domain",
			fmt.Sprintf("Could not update domain %s for project %s, unexpected error: %s",
				plan.Domain.Value,
				plan.ProjectID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDomain(out, plan.TeamID)
	tflog.Trace(ctx, "update project domain", map[string]interface{}{
		"project_id": result.ProjectID.Value,
		"domain":     result.Domain.Value,
		"team_id":    result.TeamID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete will remove a project domain via the Vercel API.
func (r resourceProjectDomain) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state ProjectDomain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.p.client.DeleteProjectDomain(ctx, state.ProjectID.Value, state.Domain.Value, state.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		// The domain is already gone - do nothing.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			fmt.Sprintf(
				"Could not delete domain %s for project %s, unexpected error: %s",
				state.Domain.Value,
				state.ProjectID.Value,
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "delete project domain", map[string]interface{}{
		"project_id": state.ProjectID.Value,
		"domain":     state.Domain.Value,
		"team_id":    state.TeamID.Value,
	})
}

// splitProjectDomainID is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitProjectDomainID(id string) (teamID, projectID, domain string, ok bool) {
	attributes := strings.Split(id, "/")
	if len(attributes) == 2 {
		// we have project_id/domain
		return "", attributes[0], attributes[1], true
	}
	if len(attributes) == 3 {
		// we have team_id/project_id/domain
		return attributes[0], attributes[1], attributes[2], true
	}
	return "", "", "", false
}

// ImportState takes an identifier and reads all the project domain information from the Vercel API.
// Note that environment variables are also read. The results are then stored in terraform state.
func (r resourceProjectDomain) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	teamID, projectID, domain, ok := splitProjectDomainID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project domain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/domain\" or \"project_id/domain\"", req.ID),
		)
	}

	out, err := r.p.client.GetProjectDomain(ctx, projectID, domain, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project domain",
			fmt.Sprintf("Could not get domain %s for project %s, unexpected error: %s",
				domain,
				projectID,
				err,
			),
		)
		return
	}

	stringTypeTeamID := types.String{Value: teamID}
	if teamID == "" {
		stringTypeTeamID.Null = true
	}
	result := convertResponseToProjectDomain(out, stringTypeTeamID)
	tflog.Trace(ctx, "imported project domain", map[string]interface{}{
		"project_id": result.ProjectID.Value,
		"domain":     result.Domain.Value,
		"team_id":    result.TeamID.Value,
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
