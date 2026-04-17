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
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &projectDelegatedProtectionResource{}
	_ resource.ResourceWithConfigure   = &projectDelegatedProtectionResource{}
	_ resource.ResourceWithImportState = &projectDelegatedProtectionResource{}
)

func newProjectDelegatedProtectionResource() resource.Resource {
	return &projectDelegatedProtectionResource{}
}

type projectDelegatedProtectionResource struct {
	client *client.Client
}

// ProjectDelegatedProtection reflects Terraform state for delegated protection.
type ProjectDelegatedProtection struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	TeamID         types.String `tfsdk:"team_id"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	CookieName     types.String `tfsdk:"cookie_name"`
	DeploymentType types.String `tfsdk:"deployment_type"`
	Issuer         types.String `tfsdk:"issuer"`
}

func (r *projectDelegatedProtectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_delegated_protection"
}

func (r *projectDelegatedProtectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cli, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = cli
}

func (r *projectDelegatedProtectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a Project Delegated Protection resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project to configure delegated protection for.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The OAuth client ID used for delegated protection.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The OAuth client secret used for delegated protection. This value is persisted in Terraform state.",
			},
			"cookie_name": schema.StringAttribute{
				Optional:    true,
				Description: "The cookie name used for delegated protection. Unset this attribute to remove a configured cookie name.",
			},
			"deployment_type": schema.StringAttribute{
				Required:    true,
				Description: "The deployment environment to protect. Must be one of `standard_protection_new`, `standard_protection`, `all_deployments`, or `only_preview_deployments`.",
				Validators: []validator.String{
					stringvalidator.OneOf("standard_protection_new", "standard_protection", "all_deployments", "only_preview_deployments"),
				},
			},
			"issuer": schema.StringAttribute{
				Required:    true,
				Description: "The issuer URL of the OIDC provider used for delegated protection.",
			},
		},
	}
}

func projectDelegatedProtectionFromResponse(response client.DelegatedProtection, secret types.String) ProjectDelegatedProtection {
	cookieName := types.StringNull()
	if response.CookieName != nil {
		cookieName = types.StringValue(*response.CookieName)
	}

	return ProjectDelegatedProtection{
		ID:             types.StringValue(response.ProjectID),
		ProjectID:      types.StringValue(response.ProjectID),
		TeamID:         toTeamID(response.TeamID),
		ClientID:       types.StringValue(response.ClientID),
		ClientSecret:   secret,
		CookieName:     cookieName,
		DeploymentType: fromApiDeploymentProtectionType(response.DeploymentType),
		Issuer:         types.StringValue(response.Issuer),
	}
}

func (r *projectDelegatedProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectDelegatedProtection
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ClientSecret.IsNull() || plan.ClientSecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Missing required client_secret",
			"client_secret must be set when creating delegated protection.",
		)
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project delegated protection",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to configure.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project delegated protection",
			"Error reading project information, unexpected error: "+err.Error(),
		)
		return
	}

	response, err := r.client.CreateDelegatedProtection(ctx, client.CreateDelegatedProtectionRequest{
		ProjectID:      plan.ProjectID.ValueString(),
		TeamID:         plan.TeamID.ValueString(),
		ClientID:       plan.ClientID.ValueString(),
		ClientSecret:   plan.ClientSecret.ValueString(),
		CookieName:     plan.CookieName.ValueStringPointer(),
		DeploymentType: toApiDeploymentProtectionType(plan.DeploymentType),
		Issuer:         plan.Issuer.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project delegated protection",
			"Could not create project delegated protection, unexpected error: "+err.Error(),
		)
		return
	}

	result := projectDelegatedProtectionFromResponse(response, plan.ClientSecret)
	tflog.Info(ctx, "created project delegated protection", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *projectDelegatedProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectDelegatedProtection
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectDelegatedProtection(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project delegated protection",
			fmt.Sprintf("Could not get project delegated protection %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	result := projectDelegatedProtectionFromResponse(out, state.ClientSecret)
	tflog.Info(ctx, "read project delegated protection", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *projectDelegatedProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectDelegatedProtection
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectDelegatedProtection
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientID := plan.ClientID.ValueString()
	deploymentType := toApiDeploymentProtectionType(plan.DeploymentType)
	issuer := plan.Issuer.ValueString()
	updateRequest := client.UpdateDelegatedProtectionRequest{
		ProjectID:      plan.ProjectID.ValueString(),
		TeamID:         plan.TeamID.ValueString(),
		ClientID:       &clientID,
		DeploymentType: &deploymentType,
		Issuer:         &issuer,
	}

	if plan.CookieName.IsNull() {
		// The API requires tri-state PATCH semantics for cookieName:
		// - omitted field => leave existing value unchanged
		// - string value  => set/update cookieName
		// - null          => clear cookieName
		//
		// We model "clear" by sending an empty string pointer to the client layer,
		// where it is translated into JSON null for the PATCH payload.
		emptyString := ""
		updateRequest.CookieName = &emptyString
	} else {
		updateRequest.CookieName = plan.CookieName.ValueStringPointer()
	}

	if !plan.ClientSecret.IsNull() && !plan.ClientSecret.IsUnknown() && plan.ClientSecret.ValueString() != state.ClientSecret.ValueString() {
		clientSecret := plan.ClientSecret.ValueString()
		updateRequest.ClientSecret = &clientSecret
	}

	out, err := r.client.UpdateDelegatedProtection(ctx, updateRequest)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project delegated protection",
			fmt.Sprintf("Could not update project delegated protection %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), err),
		)
		return
	}

	secret := state.ClientSecret
	if !plan.ClientSecret.IsNull() && !plan.ClientSecret.IsUnknown() {
		secret = plan.ClientSecret
	}

	result := projectDelegatedProtectionFromResponse(out, secret)
	tflog.Info(ctx, "updated project delegated protection", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *projectDelegatedProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectDelegatedProtection
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDelegatedProtection(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) || client.DelegatedProtectionNotEnabled(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project delegated protection",
			fmt.Sprintf("Could not delete project delegated protection %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted project delegated protection", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

func (r *projectDelegatedProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project delegated protection",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetProjectDelegatedProtection(ctx, projectID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project delegated protection",
			fmt.Sprintf("Could not get project delegated protection %s %s, unexpected error: %s", teamID, projectID, err),
		)
		return
	}

	result := projectDelegatedProtectionFromResponse(out, types.StringNull())
	tflog.Info(ctx, "imported project delegated protection", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
