package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
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
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
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
					int64validator.OneOf(301, 302, 307, 308),
				},
			},
			"git_branch": schema.StringAttribute{
				Description: "Git branch to link to the project domain. Deployments from this git branch will be assigned the domain name.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("custom_environment_id"),
						path.MatchRoot("git_branch"),
					),
				},
			},
			"custom_environment_id": schema.StringAttribute{
				Description: "The name of the Custom Environment to link to the Project Domain. Deployments from this custom environment will be assigned the domain name.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("custom_environment_id"),
						path.MatchRoot("git_branch"),
					),
				},
			},
		},
	}
}

// ProjectDomain reflects the state terraform stores internally for a project domain.
type ProjectDomain struct {
	Domain              types.String `tfsdk:"domain"`
	GitBranch           types.String `tfsdk:"git_branch"`
	CustomEnvironmentID types.String `tfsdk:"custom_environment_id"`
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	Redirect            types.String `tfsdk:"redirect"`
	RedirectStatusCode  types.Int64  `tfsdk:"redirect_status_code"`
	TeamID              types.String `tfsdk:"team_id"`
}

func convertResponseToProjectDomain(response client.ProjectDomainResponse) ProjectDomain {
	return ProjectDomain{
		Domain:              types.StringValue(response.Name),
		GitBranch:           types.StringPointerValue(response.GitBranch),
		CustomEnvironmentID: types.StringPointerValue(response.CustomEnvironmentID),
		ID:                  types.StringValue(response.Name),
		ProjectID:           types.StringValue(response.ProjectID),
		Redirect:            types.StringPointerValue(response.Redirect),
		RedirectStatusCode:  types.Int64PointerValue(response.RedirectStatusCode),
		TeamID:              toTeamID(response.TeamID),
	}
}

func (p *ProjectDomain) toCreateRequest() client.CreateProjectDomainRequest {
	return client.CreateProjectDomainRequest{
		GitBranch:           p.GitBranch.ValueString(),
		CustomEnvironmentID: p.CustomEnvironmentID.ValueString(),
		Name:                p.Domain.ValueString(),
		Redirect:            p.Redirect.ValueString(),
		RedirectStatusCode:  p.RedirectStatusCode.ValueInt64(),
	}
}

func (p *ProjectDomain) toUpdateRequest() client.UpdateProjectDomainRequest {
	return client.UpdateProjectDomainRequest{
		GitBranch:           p.GitBranch.ValueStringPointer(),
		CustomEnvironmentID: p.CustomEnvironmentID.ValueStringPointer(),
		Redirect:            p.Redirect.ValueStringPointer(),
		RedirectStatusCode:  p.RedirectStatusCode.ValueInt64Pointer(),
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

	project, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project domain",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	// Crazy condition to add an error if the git_branch is the production branch.
	if plan.GitBranch.ValueString() != "" && project.Link != nil && project.Link.ProductionBranch != nil && *project.Link.ProductionBranch == plan.GitBranch.ValueString() {
		resp.Diagnostics.AddError(
			"Error adding domain to project",
			fmt.Sprintf(
				"Could not add domain %s to project %s, the git_branch specified is the production branch. If you want to use this domain as a production domain, please omit the git_branch field.",
				plan.Domain.ValueString(),
				plan.ProjectID.ValueString(),
			),
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
	tflog.Info(ctx, "added domain to project", map[string]any{
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
	tflog.Info(ctx, "read project domain", map[string]any{
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
	tflog.Info(ctx, "update project domain", map[string]any{
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

	tflog.Info(ctx, "delete project domain", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"domain":     state.Domain.ValueString(),
		"team_id":    state.TeamID.ValueString(),
	})
}

// ImportState takes an identifier and reads all the project domain information from the Vercel API.
// Note that environment variables are also read. The results are then stored in terraform state.
func (r *projectDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, domain, ok := splitInto2Or3(req.ID)
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
	tflog.Info(ctx, "imported project domain", map[string]any{
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
