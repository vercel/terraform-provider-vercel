package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

var (
	_ resource.Resource              = &projectDomainResource{}
	_ resource.ResourceWithConfigure = &projectDomainResource{}
)

func newProjectDomainResource() resource.Resource {
	return &projectDomainResource{}
}

type projectDomainResource struct {
	client *client.Client
}

func (r *projectDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_domain"
}

func (r *projectDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a deployment resource.
func (r *projectDomainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Domain resource.

A Project Domain is used to associate a domain name with a ` + "`vercel_project`." + `

By default, Project Domains will be automatically applied to any ` + "`production` deployments.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description:   "The project ID to add the deployment to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Description:   "The ID of the team the project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Description:   "The domain name to associate with the project.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"redirect": schema.StringAttribute{
				Description: "The domain name that serves as a target destination for redirects.",
				Optional:    true,
			},
			"redirect_status_code": schema.Int64Attribute{
				Description: "The HTTP status code to use when serving as a redirect.",
				Optional:    true,
				Validators: []validator.Int64{
					int64OneOf(301, 302, 307, 308),
				},
			},
			"git_branch": schema.StringAttribute{
				Description: "Git branch to link to the project domain. Deployments from this git branch will be assigned the domain name.",
				Optional:    true,
			},
		},
	}
}

// Create will create a project domain within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *projectDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectDomain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), false)
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project domain",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	out, err := r.client.CreateProjectDomain(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), plan.toCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding domain to project",
			fmt.Sprintf(
				"Could not add domain %s to project %s, unexpected error: %s",
				plan.Domain.ValueString(),
				plan.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDomain(out)
	tflog.Trace(ctx, "added domain to project", map[string]interface{}{
		"project_id": result.ProjectID.ValueString(),
		"domain":     result.Domain.ValueString(),
		"team_id":    result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a project domain from the vercel API and provide terraform with information about it.
// It is called by the provider whenever data source values should be read to update state.
func (r *projectDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectDomain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectDomain(ctx, state.ProjectID.ValueString(), state.Domain.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project domain",
			fmt.Sprintf("Could not get domain %s for project %s, unexpected error: %s",
				state.Domain.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDomain(out)
	tflog.Trace(ctx, "read project domain", map[string]interface{}{
		"project_id": result.ProjectID.ValueString(),
		"domain":     result.Domain.ValueString(),
		"team_id":    result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update will update a project domain via the vercel API.
func (r *projectDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectDomain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateProjectDomain(
		ctx,
		plan.ProjectID.ValueString(),
		plan.Domain.ValueString(),
		plan.TeamID.ValueString(),
		plan.toUpdateRequest(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project domain",
			fmt.Sprintf("Could not update domain %s for project %s, unexpected error: %s",
				plan.Domain.ValueString(),
				plan.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDomain(out)
	tflog.Trace(ctx, "update project domain", map[string]interface{}{
		"project_id": result.ProjectID.ValueString(),
		"domain":     result.Domain.ValueString(),
		"team_id":    result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete will remove a project domain via the Vercel API.
func (r *projectDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectDomain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProjectDomain(ctx, state.ProjectID.ValueString(), state.Domain.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		// The domain is already gone - do nothing.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			fmt.Sprintf(
				"Could not delete domain %s for project %s, unexpected error: %s",
				state.Domain.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "delete project domain", map[string]interface{}{
		"project_id": state.ProjectID.ValueString(),
		"domain":     state.Domain.ValueString(),
		"team_id":    state.TeamID.ValueString(),
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
func (r *projectDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, domain, ok := splitProjectDomainID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project domain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/domain\" or \"project_id/domain\"", req.ID),
		)
	}

	out, err := r.client.GetProjectDomain(ctx, projectID, domain, teamID)
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

	result := convertResponseToProjectDomain(out)
	tflog.Trace(ctx, "imported project domain", map[string]interface{}{
		"project_id": result.ProjectID.ValueString(),
		"domain":     result.Domain.ValueString(),
		"team_id":    result.TeamID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
