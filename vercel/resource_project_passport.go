package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &projectPassportResource{}
	_ resource.ResourceWithConfigure   = &projectPassportResource{}
	_ resource.ResourceWithImportState = &projectPassportResource{}
)

type projectPassportResource struct{ client *client.Client }

type projectPassportModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	TeamID         types.String `tfsdk:"team_id"`
	ConnectorID    types.String `tfsdk:"connector_id"`
	DeploymentType types.String `tfsdk:"deployment_type"`
}

func newProjectPassportResource() resource.Resource { return &projectPassportResource{} }

func (r *projectPassportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_passport"
}

func (r *projectPassportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
	}
}

func (r *projectPassportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages Passport as a standalone setting on an existing Vercel project. It does not create, import, or delete the project itself, and does not manage builds, Git settings, other protection methods, bypasses, or exceptions. Requires an eligible Enterprise team and Owner permissions.

Creating this resource enables Passport. Destroying it disables Passport, leaving the project, Connect application, and application connections intact. To stop managing Passport while keeping it enabled, use a Terraform removed block with lifecycle { destroy = false }.

If the project is also managed by vercel_project, set manage_passport = false on that resource and remove its inline passport attribute before using this resource. Do not give multiple resources ownership of Passport for the same project. Import adopts the existing Passport configuration without changing it; Passport must already be enabled.`,
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true, Description: "The project ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"project_id":      schema.StringAttribute{Required: true, Description: "The ID of the existing project. This resource never creates a project.", Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"team_id":         schema.StringAttribute{Optional: true, Computed: true, Description: "The team ID. Defaults to the provider team.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()}},
			"connector_id":    schema.StringAttribute{Required: true, Description: "The stable ID of an existing Vercel Connect OAuth application, available from the vercel_connect_application data source.", Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
			"deployment_type": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("all"), Description: "Deployments to protect: all, preview, prod_deployment_urls_and_all_previews, or all_except_custom_domains. Defaults to all.", Validators: []validator.String{stringvalidator.OneOf("all", "preview", "prod_deployment_urls_and_all_previews", "all_except_custom_domains")}},
		},
	}
}

func (m *projectPassportModel) read(out client.ProjectResponse) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ID)
	m.TeamID = toTeamID(out.TeamID)
	m.ConnectorID = types.StringValue(out.Passport.ConnectorID)
	m.DeploymentType = types.StringValue(out.Passport.DeploymentType)
}

func (r *projectPassportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectPassportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateProjectPassport(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), &client.Passport{ConnectorID: plan.ConnectorID.ValueString(), DeploymentType: plan.DeploymentType.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error enabling project Passport", err.Error())
		return
	}
	if out.Passport == nil {
		resp.Diagnostics.AddError("Passport was not enabled", "The API returned no Passport configuration after the update.")
		return
	}
	plan.read(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectPassportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectPassportModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetProject(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading project Passport", err.Error())
		return
	}
	if out.Passport == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.read(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectPassportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectPassportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateProjectPassport(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), &client.Passport{ConnectorID: plan.ConnectorID.ValueString(), DeploymentType: plan.DeploymentType.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error updating project Passport", err.Error())
		return
	}
	if out.Passport == nil {
		resp.Diagnostics.AddError("Passport was not enabled", "The API returned no Passport configuration after the update.")
		return
	}
	plan.read(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectPassportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectPassportModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.UpdateProjectPassport(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), nil)
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error disabling project Passport", err.Error())
	}
}

func (r *projectPassportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "Use project_id or team_id/project_id.")
		return
	}
	out, err := r.client.GetProject(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing project Passport", err.Error())
		return
	}
	if out.Passport == nil {
		resp.Diagnostics.AddError("Passport is not enabled", "Enable Passport before importing, or create this settings resource to enable it on the existing project.")
		return
	}
	var state projectPassportModel
	state.read(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
