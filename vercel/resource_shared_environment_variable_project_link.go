package vercel

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource              = &sharedEnvironmentVariableProjectLinkResource{}
	_ resource.ResourceWithConfigure = &sharedEnvironmentVariableProjectLinkResource{}
)

func newSharedEnvironmentVariableProjectLinkResource() resource.Resource {
	return &sharedEnvironmentVariableProjectLinkResource{}
}

type sharedEnvironmentVariableProjectLinkResource struct {
	client *client.Client
}

func (r *sharedEnvironmentVariableProjectLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_environment_variable_project_link"
}

func (r *sharedEnvironmentVariableProjectLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sharedEnvironmentVariableProjectLinkResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Links a project to a Shared Environment Variable.

~> This resource is intended to be used alongside a vercel_shared_environment_variable Data Source, not the Resource. The Resource also defines which projects to link to the shared environment variable, and using both vercel_shared_environment_variable and vercel_shared_environment_variable_project_link together results in undefined behavior.`,
		Attributes: map[string]schema.Attribute{
			"shared_environment_variable_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The ID of the shared environment variable.",
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type SharedEnvironmentVariableProjectLink struct {
	SharedEnvironmentVariableID types.String `tfsdk:"shared_environment_variable_id"`
	ProjectID                   types.String `tfsdk:"project_id"`
	TeamID                      types.String `tfsdk:"team_id"`
}

func (e *SharedEnvironmentVariableProjectLink) toUpdateSharedEnvironmentVariableRequest(link bool) (req client.UpdateSharedEnvironmentVariableRequest, ok bool) {
	upd := client.UpdateSharedEnvironmentVariableRequestProjectIDUpdates{}

	if link {
		upd.Link = []string{e.ProjectID.ValueString()}
	} else {
		upd.Unlink = []string{e.ProjectID.ValueString()}
	}

	return client.UpdateSharedEnvironmentVariableRequest{
		TeamID:           e.TeamID.ValueString(),
		EnvID:            e.SharedEnvironmentVariableID.ValueString(),
		ProjectIDUpdates: upd,
	}, true
}

func (r *sharedEnvironmentVariableProjectLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SharedEnvironmentVariableProjectLink
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, ok := plan.toUpdateSharedEnvironmentVariableRequest(true)
	if !ok {
		return
	}

	response, err := r.client.UpdateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error linking project to shared environment variable",
			"Could not link project to shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := SharedEnvironmentVariableProjectLink{
		TeamID:                      types.StringValue(response.TeamID),
		SharedEnvironmentVariableID: types.StringValue(response.ID),
		ProjectID:                   plan.ProjectID,
	}

	tflog.Info(ctx, "linked shared environment variable to project", map[string]any{
		"team_id":                        result.TeamID.ValueString(),
		"shared_environment_variable_id": result.SharedEnvironmentVariableID.ValueString(),
		"project_id":                     result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sharedEnvironmentVariableProjectLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SharedEnvironmentVariableProjectLink
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.SharedEnvironmentVariableID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			"Could not read shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	if !slices.Contains(response.ProjectIDs, state.ProjectID.ValueString()) {
		tflog.Info(ctx, "failed to read shared environment variable for linked project", map[string]any{
			"team_id":                        state.TeamID.ValueString(),
			"shared_environment_variable_id": state.SharedEnvironmentVariableID.ValueString(),
			"project_id":                     state.ProjectID.ValueString(),
		})

		// not found, so replace state
		resp.State.RemoveResource(ctx)
		return
	}

	result := SharedEnvironmentVariableProjectLink{
		TeamID:                      types.StringValue(response.TeamID),
		SharedEnvironmentVariableID: types.StringValue(response.ID),
		ProjectID:                   state.ProjectID,
	}
	tflog.Info(ctx, "read shared environment variable for linked project", map[string]any{
		"team_id":                        result.TeamID.ValueString(),
		"shared_environment_variable_id": result.SharedEnvironmentVariableID.ValueString(),
		"project_id":                     result.ProjectID.ValueString(),
	})
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *sharedEnvironmentVariableProjectLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Linking should always be recreated", "Something incorrectly caused an Update, this should always be recreated instead of updated.")
}

func (r *sharedEnvironmentVariableProjectLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan SharedEnvironmentVariableProjectLink
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, ok := plan.toUpdateSharedEnvironmentVariableRequest(false)
	if !ok {
		return
	}

	response, err := r.client.UpdateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error unlinking project from shared environment variable",
			"Could not unlink project from shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := SharedEnvironmentVariableProjectLink{
		TeamID:                      types.StringValue(response.TeamID),
		SharedEnvironmentVariableID: types.StringValue(response.ID),
		ProjectID:                   plan.ProjectID,
	}

	tflog.Info(ctx, "project unlinked from shared environment", map[string]any{
		"team_id":                        result.TeamID.ValueString(),
		"shared_environment_variable_id": result.SharedEnvironmentVariableID.ValueString(),
		"project_id":                     result.ProjectID.ValueString(),
	})
}
