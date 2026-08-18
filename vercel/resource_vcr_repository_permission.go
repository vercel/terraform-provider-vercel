package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &vcrRepositoryPermissionResource{}
	_ resource.ResourceWithConfigure   = &vcrRepositoryPermissionResource{}
	_ resource.ResourceWithImportState = &vcrRepositoryPermissionResource{}
)

func newVCRRepositoryPermissionResource() resource.Resource {
	return &vcrRepositoryPermissionResource{}
}

type vcrRepositoryPermissionResource struct {
	client *client.Client
}

func (r *vcrRepositoryPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcr_repository_permission"
}

func (r *vcrRepositoryPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vcrRepositoryPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel Container Registry (VCR) Repository Permission resource.

A VCR Repository Permission shares a VCR Repository with another Vercel Team, granting
it read (pull) access to the repository's images. One resource manages a single grant,
identified by the repository and the team the permission is granted to.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of this resource. Format: `repository_id/granted_team_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Description:   "The ID of the team that owns the repository. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Vercel Project the repository belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository": schema.StringAttribute{
				Description:   "The ID or name of the VCR Repository to share.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the VCR Repository the permission is granted on.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"granted_team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team to grant pull access to. Must specify one of granted_team_id or granted_team_slug.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("granted_team_id"),
						path.MatchRoot("granted_team_slug"),
					),
				},
			},
			"granted_team_slug": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The slug of the team to grant pull access to. Must specify one of granted_team_id or granted_team_slug.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("granted_team_id"),
						path.MatchRoot("granted_team_slug"),
					),
				},
			},
		},
	}
}

type VCRRepositoryPermission struct {
	ID              types.String `tfsdk:"id"`
	TeamID          types.String `tfsdk:"team_id"`
	ProjectID       types.String `tfsdk:"project_id"`
	Repository      types.String `tfsdk:"repository"`
	RepositoryID    types.String `tfsdk:"repository_id"`
	GrantedTeamID   types.String `tfsdk:"granted_team_id"`
	GrantedTeamSlug types.String `tfsdk:"granted_team_slug"`
}

func convertResponseToVCRRepositoryPermission(res client.VCRRepositoryPermission, projectID, repository string) VCRRepositoryPermission {
	repositoryID := res.RepositoryID
	if repositoryID == "" {
		repositoryID = repository
	}
	return VCRRepositoryPermission{
		ID:              types.StringValue(repositoryID + "/" + res.GrantedTeamID),
		TeamID:          types.StringValue(res.TeamID),
		ProjectID:       types.StringValue(projectID),
		Repository:      types.StringValue(repository),
		RepositoryID:    types.StringValue(repositoryID),
		GrantedTeamID:   types.StringValue(res.GrantedTeamID),
		GrantedTeamSlug: types.StringValue(res.GrantedTeamSlug),
	}
}

func (p VCRRepositoryPermission) repositoryIDOrName() string {
	if p.RepositoryID.ValueString() != "" {
		return p.RepositoryID.ValueString()
	}
	return p.Repository.ValueString()
}

func (r *vcrRepositoryPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VCRRepositoryPermission
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.CreateVCRRepositoryPermission(ctx, client.CreateVCRRepositoryPermissionRequest{
		TeamID:          plan.TeamID.ValueString(),
		ProjectID:       plan.ProjectID.ValueString(),
		IDOrName:        plan.Repository.ValueString(),
		GrantedTeamID:   plan.GrantedTeamID.ValueString(),
		GrantedTeamSlug: plan.GrantedTeamSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VCR Repository Permission",
			fmt.Sprintf("Could not create VCR Repository Permission, unexpected error: %s", err),
		)
		return
	}

	tflog.Info(ctx, "created vcr repository permission", map[string]any{
		"team_id":         res.TeamID,
		"project_id":      plan.ProjectID.ValueString(),
		"repository_id":   res.RepositoryID,
		"granted_team_id": res.GrantedTeamID,
	})

	diags = resp.State.Set(ctx, convertResponseToVCRRepositoryPermission(res, plan.ProjectID.ValueString(), plan.Repository.ValueString()))
	resp.Diagnostics.Append(diags...)
}

func (r *vcrRepositoryPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VCRRepositoryPermission
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetVCRRepositoryPermission(ctx, client.GetVCRRepositoryPermissionRequest{
		TeamID:          state.TeamID.ValueString(),
		ProjectID:       state.ProjectID.ValueString(),
		IDOrName:        state.repositoryIDOrName(),
		GrantedTeamID:   state.GrantedTeamID.ValueString(),
		GrantedTeamSlug: state.GrantedTeamSlug.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading VCR Repository Permission",
			fmt.Sprintf("Could not read VCR Repository Permission %s %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.Repository.ValueString(),
				state.GrantedTeamID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "read vcr repository permission", map[string]any{
		"team_id":         res.TeamID,
		"project_id":      state.ProjectID.ValueString(),
		"repository_id":   res.RepositoryID,
		"granted_team_id": res.GrantedTeamID,
	})

	diags = resp.State.Set(ctx, convertResponseToVCRRepositoryPermission(res, state.ProjectID.ValueString(), state.Repository.ValueString()))
	resp.Diagnostics.Append(diags...)
}

// Update is never called as all attributes force a replacement.
func (r *vcrRepositoryPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Error updating VCR Repository Permission",
		"VCR Repository Permissions cannot be updated. Any change requires the permission to be replaced.",
	)
}

func (r *vcrRepositoryPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VCRRepositoryPermission
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API accepts exactly one of the granted team ID or slug.
	request := client.DeleteVCRRepositoryPermissionRequest{
		TeamID:        state.TeamID.ValueString(),
		ProjectID:     state.ProjectID.ValueString(),
		IDOrName:      state.repositoryIDOrName(),
		GrantedTeamID: state.GrantedTeamID.ValueString(),
	}
	if request.GrantedTeamID == "" {
		request.GrantedTeamSlug = state.GrantedTeamSlug.ValueString()
	}

	err := r.client.DeleteVCRRepositoryPermission(ctx, request)
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting VCR Repository Permission",
			fmt.Sprintf("Could not delete VCR Repository Permission %s %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.Repository.ValueString(),
				state.GrantedTeamID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted vcr repository permission", map[string]any{
		"team_id":         state.TeamID.ValueString(),
		"project_id":      state.ProjectID.ValueString(),
		"repository":      state.Repository.ValueString(),
		"granted_team_id": state.GrantedTeamID.ValueString(),
	})
}

func (r *vcrRepositoryPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, repository, grantedTeam, ok := splitInto3Or4(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing VCR Repository Permission",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/repository/granted_team_id\" or \"project_id/repository/granted_team_id\"", req.ID),
		)
		return
	}

	res, err := r.client.GetVCRRepositoryPermission(ctx, client.GetVCRRepositoryPermissionRequest{
		TeamID:          teamID,
		ProjectID:       projectID,
		IDOrName:        repository,
		GrantedTeamID:   grantedTeam,
		GrantedTeamSlug: grantedTeam,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing VCR Repository Permission",
			fmt.Sprintf("Could not import VCR Repository Permission %s %s %s %s, unexpected error: %s",
				teamID,
				projectID,
				repository,
				grantedTeam,
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "imported vcr repository permission", map[string]any{
		"team_id":         res.TeamID,
		"project_id":      projectID,
		"repository_id":   res.RepositoryID,
		"granted_team_id": res.GrantedTeamID,
	})

	diags := resp.State.Set(ctx, convertResponseToVCRRepositoryPermission(res, projectID, repository))
	resp.Diagnostics.Append(diags...)
}
