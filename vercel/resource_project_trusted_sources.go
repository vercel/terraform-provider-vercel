package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &projectTrustedSourcesResource{}
	_ resource.ResourceWithConfigure   = &projectTrustedSourcesResource{}
	_ resource.ResourceWithImportState = &projectTrustedSourcesResource{}
)

type projectTrustedSourcesResource struct{ client *client.Client }

type projectTrustedSourcesModel struct {
	ID                           types.String `tfsdk:"id"`
	ProjectID                    types.String `tfsdk:"project_id"`
	TeamID                       types.String `tfsdk:"team_id"`
	Projects                     types.Set    `tfsdk:"projects"`
	ExternalSources              types.Set    `tfsdk:"external_sources"`
	EnableVercelCiSameRepository types.Bool   `tfsdk:"enable_vercel_ci_same_repository"`
}

func newProjectTrustedSourcesResource() resource.Resource {
	return &projectTrustedSourcesResource{}
}

func (r *projectTrustedSourcesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_trusted_sources"
}

func (r *projectTrustedSourcesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
	}
}

func (r *projectTrustedSourcesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	projects := trustedSourcesProjectsResourceSchema()
	projects.Description = "Vercel projects in the same team that can reach this project's protected deployments. Omit to trust no other projects; removing every entry clears the list."
	projects.Validators = append(projects.Validators, setvalidator.SizeAtLeast(1))
	externalSources := trustedSourcesExternalSourcesResourceSchema()
	externalSources.Description = "External sources that can reach this project's protected deployments using short-lived OIDC tokens. Omit to trust no external sources; removing every entry clears the list."
	externalSources.Validators = append(externalSources.Validators, setvalidator.SizeAtLeast(1))

	resp.Schema = schema.Schema{
		Description: `Manages Deployment Protection Trusted Sources as a standalone setting on an existing Vercel project. It never creates or deletes the project, and writes only the project's trustedSources setting.

This resource owns the project's complete list of trusted projects and external sources: applying it replaces any entries that are not in the configuration. Destroying it clears Trusted Sources, leaving the project intact. To stop managing Trusted Sources while keeping them, use a removed block with lifecycle { destroy = false }.

If vercel_project manages the same project, set manage_trusted_sources = false there and remove its inline trusted_sources attribute. Only one resource may own Trusted Sources per project. Import adopts the existing configuration without changing it.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The project ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the existing project. No project is created by this resource.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The team ID. Defaults to the provider team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"projects":         projects,
			"external_sources": externalSources,
			"enable_vercel_ci_same_repository": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Allow same-team Vercel CI runs to access preview and production deployments built from the same repository as the CI run. The API enables this when no value is stored. When omitted, the stored value is left unchanged, including on destroy.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// read replaces the model with the live configuration. Empty lists are null
// because the schema requires at least one entry when a list is configured.
func (m *projectTrustedSourcesModel) read(out client.ProjectResponse) {
	m.ID, m.ProjectID, m.TeamID = types.StringValue(out.ID), types.StringValue(out.ID), toTeamID(out.TeamID)
	m.Projects = types.SetNull(trustedSourcesProjectAttrType)
	m.ExternalSources = types.SetNull(trustedSourcesExternalSourceAttrType)
	m.EnableVercelCiSameRepository = types.BoolValue(true)
	if out.TrustedSources == nil {
		return
	}
	value := trustedSourcesObjectFromResponse(*out.TrustedSources).Attributes()
	if len(out.TrustedSources.Projects) > 0 {
		m.Projects = value["projects"].(types.Set)
	}
	if len(out.TrustedSources.OIDCProviders) > 0 {
		m.ExternalSources = value["external_sources"].(types.Set)
	}
	if out.TrustedSources.EnableVercelCiSameRepository != nil {
		m.EnableVercelCiSameRepository = types.BoolValue(*out.TrustedSources.EnableVercelCiSameRepository)
	}
}

func (m projectTrustedSourcesModel) request(ctx context.Context, config projectTrustedSourcesModel) (*client.ProjectTrustedSources, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.Projects.IsUnknown() || m.ExternalSources.IsUnknown() {
		diags.AddError("Unknown Trusted Sources", "Trusted projects and external sources must be known at apply.")
		return nil, diags
	}
	trusted, d := (&TrustedSources{Projects: m.Projects, ExternalSources: m.ExternalSources}).toUpdateProjectRequest(ctx)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	request := &client.ProjectTrustedSources{Projects: trusted.Projects, OIDCProviders: trusted.OIDCProviders}
	// Write the flag only when configured, so omitting it preserves the stored value.
	if !config.EnableVercelCiSameRepository.IsNull() {
		if m.EnableVercelCiSameRepository.IsUnknown() {
			diags.AddError("Unknown Vercel CI setting", "enable_vercel_ci_same_repository must be known at apply.")
			return nil, diags
		}
		request.EnableVercelCiSameRepository = m.EnableVercelCiSameRepository.ValueBoolPointer()
	}
	return request, diags
}

func (r *projectTrustedSourcesResource) apply(ctx context.Context, plan *projectTrustedSourcesModel, config projectTrustedSourcesModel) diag.Diagnostics {
	request, diags := plan.request(ctx, config)
	if diags.HasError() {
		return diags
	}
	out, err := r.client.UpdateProjectTrustedSources(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), request)
	if err != nil {
		diags.AddError("Error updating project Trusted Sources", err.Error())
		return diags
	}
	plan.read(out)
	return diags
}

func (r *projectTrustedSourcesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config projectTrustedSourcesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan, config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *projectTrustedSourcesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectTrustedSourcesModel
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
		resp.Diagnostics.AddError("Error reading project Trusted Sources", err.Error())
		return
	}
	state.read(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectTrustedSourcesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config projectTrustedSourcesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan, config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *projectTrustedSourcesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectTrustedSourcesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.UpdateProjectTrustedSources(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), nil)
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error clearing project Trusted Sources", err.Error())
	}
}

func (r *projectTrustedSourcesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "Use project_id or team_id/project_id.")
		return
	}
	out, err := r.client.GetProject(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing project Trusted Sources", err.Error())
		return
	}
	var state projectTrustedSourcesModel
	state.read(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
