package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &deploymentProtectionExceptionResource{}
	_ resource.ResourceWithConfigure   = &deploymentProtectionExceptionResource{}
	_ resource.ResourceWithImportState = &deploymentProtectionExceptionResource{}
)

func newDeploymentProtectionExceptionResource() resource.Resource {
	return &deploymentProtectionExceptionResource{}
}

type deploymentProtectionExceptionResource struct {
	client *client.Client
}

func (r *deploymentProtectionExceptionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_protection_exception"
}

func (r *deploymentProtectionExceptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *deploymentProtectionExceptionResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Deployment Protection Exception resource.

A Deployment Protection Exception makes a preview alias or deployment URL publicly accessible by bypassing Deployment Protection for that URL.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project the exception belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"alias": schema.StringAttribute{
				Description:   "The preview alias or deployment URL to add to Deployment Protection Exceptions.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "The unix timestamp in milliseconds at which the exception was created.",
			},
			"created_by": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the user who created the exception.",
			},
			"scope": schema.StringAttribute{
				Computed:    true,
				Description: "The scope of the exception. Always `alias-protection-override` for Deployment Protection Exceptions.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type DeploymentProtectionException struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	Alias     types.String `tfsdk:"alias"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	CreatedBy types.String `tfsdk:"created_by"`
	Scope     types.String `tfsdk:"scope"`
}

func protectionOverride(response client.AliasResponse) (client.ProtectionBypass, bool) {
	override, ok := response.ProtectionBypass["*"]
	if !ok || override.Scope != "alias-protection-override" {
		return client.ProtectionBypass{}, false
	}
	return override, true
}

func responseToDeploymentProtectionException(response client.AliasResponse, state DeploymentProtectionException) (DeploymentProtectionException, bool) {
	override, ok := protectionOverride(response)
	if !ok {
		return DeploymentProtectionException{}, false
	}

	projectID := state.ProjectID.ValueString()
	alias := state.Alias.ValueString()
	if response.Alias != "" {
		alias = response.Alias
	}

	return DeploymentProtectionException{
		ID:        types.StringValue(fmt.Sprintf("%s/%s", projectID, alias)),
		ProjectID: types.StringValue(projectID),
		TeamID:    toTeamID(response.TeamID),
		Alias:     types.StringValue(alias),
		CreatedAt: types.Int64Value(override.CreatedAt),
		CreatedBy: types.StringValue(override.CreatedBy),
		Scope:     types.StringValue(override.Scope),
	}, true
}

func aliasProjectMatches(response client.AliasResponse, projectID string) bool {
	return response.ProjectID == "" || response.ProjectID == projectID
}

func (r *deploymentProtectionExceptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeploymentProtectionException
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAlias(ctx, plan.Alias.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Alias not found",
			fmt.Sprintf("Could not create deployment protection exception because alias %s was not found.", plan.Alias.ValueString()),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading alias before creating deployment protection exception",
			fmt.Sprintf("Could not read alias %s before creating deployment protection exception: %s", plan.Alias.ValueString(), err),
		)
		return
	}
	if !aliasProjectMatches(out, plan.ProjectID.ValueString()) {
		resp.Diagnostics.AddError(
			"Alias belongs to a different project",
			fmt.Sprintf("Alias %s belongs to project %s, not configured project %s.", plan.Alias.ValueString(), out.ProjectID, plan.ProjectID.ValueString()),
		)
		return
	}

	_, err = r.client.UpdateDeploymentProtectionException(ctx, client.UpdateDeploymentProtectionExceptionRequest{
		Alias:  plan.Alias.ValueString(),
		TeamID: plan.TeamID.ValueString(),
		Action: "create",
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment protection exception",
			fmt.Sprintf("Could not create deployment protection exception for alias %s: %s", plan.Alias.ValueString(), err),
		)
		return
	}

	out, err = r.client.GetAlias(ctx, plan.Alias.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment protection exception after creation",
			fmt.Sprintf("Could not read alias %s after creating deployment protection exception: %s", plan.Alias.ValueString(), err),
		)
		return
	}
	if !aliasProjectMatches(out, plan.ProjectID.ValueString()) {
		resp.Diagnostics.AddError(
			"Alias belongs to a different project",
			fmt.Sprintf("Alias %s belongs to project %s, not configured project %s.", plan.Alias.ValueString(), out.ProjectID, plan.ProjectID.ValueString()),
		)
		return
	}

	result, ok := responseToDeploymentProtectionException(out, plan)
	if !ok {
		resp.Diagnostics.AddError(
			"Deployment protection exception missing after creation",
			fmt.Sprintf("Alias %s did not contain a Deployment Protection Exception after creation.", plan.Alias.ValueString()),
		)
		return
	}

	tflog.Info(ctx, "created deployment protection exception", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"alias":      result.Alias.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *deploymentProtectionExceptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeploymentProtectionException
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAlias(ctx, state.Alias.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment protection exception",
			fmt.Sprintf("Could not read alias %s: %s", state.Alias.ValueString(), err),
		)
		return
	}
	if !aliasProjectMatches(out, state.ProjectID.ValueString()) {
		resp.Diagnostics.AddError(
			"Alias belongs to a different project",
			fmt.Sprintf("Alias %s belongs to project %s, not configured project %s.", state.Alias.ValueString(), out.ProjectID, state.ProjectID.ValueString()),
		)
		return
	}

	result, ok := responseToDeploymentProtectionException(out, state)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "read deployment protection exception", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"alias":      result.Alias.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *deploymentProtectionExceptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentProtectionException
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *deploymentProtectionExceptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DeploymentProtectionException
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateDeploymentProtectionException(ctx, client.UpdateDeploymentProtectionExceptionRequest{
		Alias:  state.Alias.ValueString(),
		TeamID: state.TeamID.ValueString(),
		Action: "revoke",
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting deployment protection exception",
			fmt.Sprintf("Could not delete deployment protection exception for alias %s: %s", state.Alias.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted deployment protection exception", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
		"alias":      state.Alias.ValueString(),
	})
}

func (r *deploymentProtectionExceptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, alias, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing deployment protection exception",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/alias\" or \"project_id/alias\"", req.ID),
		)
		return
	}

	out, err := r.client.GetAlias(ctx, alias, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing deployment protection exception",
			fmt.Sprintf("Could not read alias %s: %s", alias, err),
		)
		return
	}
	if !aliasProjectMatches(out, projectID) {
		resp.Diagnostics.AddError(
			"Error importing deployment protection exception",
			fmt.Sprintf("Alias %s belongs to project %s, not imported project %s.", alias, out.ProjectID, projectID),
		)
		return
	}

	result, ok := responseToDeploymentProtectionException(out, DeploymentProtectionException{
		ProjectID: types.StringValue(projectID),
		TeamID:    toTeamID(teamID),
		Alias:     types.StringValue(alias),
	})
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing deployment protection exception",
			fmt.Sprintf("Alias %s does not have a Deployment Protection Exception.", alias),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
