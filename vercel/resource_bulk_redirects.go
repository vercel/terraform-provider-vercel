package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &bulkRedirectsResource{}
	_ resource.ResourceWithConfigure   = &bulkRedirectsResource{}
	_ resource.ResourceWithImportState = &bulkRedirectsResource{}
)

func newBulkRedirectsResource() resource.Resource {
	return &bulkRedirectsResource{}
}

type bulkRedirectsResource struct {
	client *client.Client
}

type bulkRedirectsResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	VersionID types.String `tfsdk:"version_id"`
	Redirects types.Set    `tfsdk:"redirects"`
}

func (r *bulkRedirectsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bulk_redirects"
}

func (r *bulkRedirectsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bulkRedirectsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Bulk Redirects resource.

This resource manages the live project-level bulk redirects for a Vercel project.
Each apply stages the configured redirect set and promotes that version to production.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Vercel project to manage redirects for.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"version_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the live bulk redirects version managed by this resource.",
			},
			"redirects": schema.SetNestedAttribute{
				Description: "The full set of live bulk redirects for the project.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Description: "The source pathname to match.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(2048),
							},
						},
						"destination": schema.StringAttribute{
							Description: "The destination pathname or URL to redirect to.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(2048),
							},
						},
						"status_code": schema.Int64Attribute{
							Description: "The HTTP status code for the redirect.",
							Required:    true,
						},
						"case_sensitive": schema.BoolAttribute{
							Description: "Whether the source match is case-sensitive.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"query": schema.BoolAttribute{
							Description: "Whether query parameters are considered when matching the redirect.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func bulkRedirectsResourceStateFromResponse(response client.BulkRedirects) bulkRedirectsResourceModel {
	return bulkRedirectsResourceModel{
		ID:        types.StringValue(response.ProjectID),
		ProjectID: types.StringValue(response.ProjectID),
		TeamID:    toTeamID(response.TeamID),
		VersionID: bulkRedirectVersionID(response.Version),
		Redirects: flattenBulkRedirects(response.Redirects),
	}
}

func (r *bulkRedirectsResource) applyBulkRedirects(ctx context.Context, projectID, teamID string, redirects []client.BulkRedirect) (client.BulkRedirects, error) {
	stagedVersion, err := r.client.StageBulkRedirects(ctx, client.StageBulkRedirectsRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		Overwrite: true,
		Redirects: redirects,
	})
	if err != nil {
		return client.BulkRedirects{}, err
	}

	liveVersion, err := r.client.UpdateBulkRedirectVersion(ctx, client.UpdateBulkRedirectVersionRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		VersionID: stagedVersion.ID,
		Action:    "promote",
	})
	if err != nil {
		return client.BulkRedirects{}, err
	}

	response, err := r.client.GetBulkRedirects(ctx, client.GetBulkRedirectsRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		VersionID: liveVersion.ID,
	})
	if err != nil {
		return client.BulkRedirects{}, err
	}

	if response.Version == nil {
		response.Version = &liveVersion
	}

	return response, nil
}

func (r *bulkRedirectsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bulkRedirectsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating bulk redirects",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to configure.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating bulk redirects",
			"Error reading project information, unexpected error: "+err.Error(),
		)
		return
	}

	redirects, diags := expandBulkRedirects(ctx, plan.Redirects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.applyBulkRedirects(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), redirects)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating bulk redirects",
			"Could not create bulk redirects, unexpected error: "+err.Error(),
		)
		return
	}

	result := bulkRedirectsResourceStateFromResponse(out)
	tflog.Info(ctx, "created bulk redirects", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"version_id": result.VersionID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *bulkRedirectsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bulkRedirectsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, ok, err := readLiveBulkRedirects(ctx, r.client, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bulk redirects",
			fmt.Sprintf("Could not get bulk redirects %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	result := bulkRedirectsResourceStateFromResponse(out)
	tflog.Info(ctx, "read bulk redirects", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"version_id": result.VersionID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *bulkRedirectsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bulkRedirectsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	redirects, diags := expandBulkRedirects(ctx, plan.Redirects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.applyBulkRedirects(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), redirects)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating bulk redirects",
			fmt.Sprintf("Could not update bulk redirects %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), err),
		)
		return
	}

	result := bulkRedirectsResourceStateFromResponse(out)
	tflog.Info(ctx, "updated bulk redirects", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"version_id": result.VersionID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *bulkRedirectsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bulkRedirectsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.applyBulkRedirects(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), []client.BulkRedirect{})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting bulk redirects",
			fmt.Sprintf("Could not delete bulk redirects %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted bulk redirects", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
	})
}

func (r *bulkRedirectsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing bulk redirects",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
		return
	}

	out, live, err := readLiveBulkRedirects(ctx, r.client, projectID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bulk redirects",
			fmt.Sprintf("Could not get bulk redirects %s %s, unexpected error: %s", teamID, projectID, err),
		)
		return
	}
	if !live {
		resp.State.RemoveResource(ctx)
		return
	}

	result := bulkRedirectsResourceStateFromResponse(out)
	tflog.Info(ctx, "imported bulk redirects", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"version_id": result.VersionID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
