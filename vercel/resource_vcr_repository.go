package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &vcrRepositoryResource{}
	_ resource.ResourceWithConfigure   = &vcrRepositoryResource{}
	_ resource.ResourceWithImportState = &vcrRepositoryResource{}
)

func newVCRRepositoryResource() resource.Resource {
	return &vcrRepositoryResource{}
}

type vcrRepositoryResource struct {
	client *client.Client
}

func (r *vcrRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcr_repository"
}

func (r *vcrRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vcrRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel Container Registry (VCR) Repository resource.

A VCR Repository belongs to a Vercel Project and stores container images that can be
used by Vercel Functions and Vercel Sandbox. Images are pushed to and pulled from
` + "`vcr.vercel.com/team-slug/project-slug/repository-name`" + ` using Docker-compatible tooling.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the VCR Repository.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Description:   "The ID of the team the repository should be created under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the existing Vercel Project the repository belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description:   "The name of the repository. Can contain lowercase letters, numbers, periods, underscores, and dashes, but cannot start or end with a period, underscore, or dash.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`),
						"The name of a VCR Repository can only contain lowercase letters, numbers, periods, underscores, and dashes, and cannot start or end with a period, underscore, or dash",
					),
				},
			},
		},
	}
}

type VCRRepository struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
}

func convertResponseToVCRRepository(res client.VCRRepository) VCRRepository {
	id := res.ID
	if id == "" {
		// The repository name is unique within a project, so fall back to it
		// if the API response does not include an ID.
		id = res.Name
	}
	return VCRRepository{
		ID:        types.StringValue(id),
		TeamID:    types.StringValue(res.TeamID),
		ProjectID: types.StringValue(res.ProjectID),
		Name:      types.StringValue(res.Name),
	}
}

func (r *vcrRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VCRRepository
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.CreateVCRRepository(ctx, client.CreateVCRRepositoryRequest{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Name:      plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VCR Repository",
			fmt.Sprintf("Could not create VCR Repository, unexpected error: %s", err),
		)
		return
	}

	tflog.Info(ctx, "created vcr repository", map[string]any{
		"team_id":    res.TeamID,
		"project_id": res.ProjectID,
		"name":       res.Name,
	})

	diags = resp.State.Set(ctx, convertResponseToVCRRepository(res))
	resp.Diagnostics.Append(diags...)
}

func (r *vcrRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VCRRepository
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetVCRRepository(ctx, client.GetVCRRepositoryRequest{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		IDOrName:  state.ID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading VCR Repository",
			fmt.Sprintf("Could not read VCR Repository %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.Name.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "read vcr repository", map[string]any{
		"team_id":    res.TeamID,
		"project_id": res.ProjectID,
		"name":       res.Name,
	})

	diags = resp.State.Set(ctx, convertResponseToVCRRepository(res))
	resp.Diagnostics.Append(diags...)
}

// Update is never called as all attributes force a replacement.
func (r *vcrRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Error updating VCR Repository",
		"VCR Repositories cannot be updated. Any change requires the repository to be replaced.",
	)
}

func (r *vcrRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VCRRepository
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteVCRRepository(ctx, client.DeleteVCRRepositoryRequest{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		IDOrName:  state.ID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting VCR Repository",
			fmt.Sprintf("Could not delete VCR Repository %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.Name.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted vcr repository", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
		"name":       state.Name.ValueString(),
	})
}

func (r *vcrRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, name, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing VCR Repository",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/repository_name\" or \"project_id/repository_name\"", req.ID),
		)
		return
	}

	res, err := r.client.GetVCRRepository(ctx, client.GetVCRRepositoryRequest{
		TeamID:    teamID,
		ProjectID: projectID,
		IDOrName:  name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing VCR Repository",
			fmt.Sprintf("Could not import VCR Repository %s %s %s, unexpected error: %s",
				teamID,
				projectID,
				name,
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "imported vcr repository", map[string]any{
		"team_id":    res.TeamID,
		"project_id": res.ProjectID,
		"name":       res.Name,
	})

	diags := resp.State.Set(ctx, convertResponseToVCRRepository(res))
	resp.Diagnostics.Append(diags...)
}
