package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &projectProtectionBypassResource{}
	_ resource.ResourceWithConfigure   = &projectProtectionBypassResource{}
	_ resource.ResourceWithImportState = &projectProtectionBypassResource{}
)

func newProjectProtectionBypassResource() resource.Resource {
	return &projectProtectionBypassResource{}
}

type projectProtectionBypassResource struct {
	client *client.Client
}

func (r *projectProtectionBypassResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protection_bypass"
}

func (r *projectProtectionBypassResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectProtectionBypassResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Protection Bypass resource.

A Project Protection Bypass is an automation bypass token that allows automation services
to bypass Deployment Protection on a ` + "`vercel_project`" + ` using the ` + "`x-vercel-protection-bypass`" + ` HTTP header.

Multiple bypasses can be created per project. Exactly one bypass per project may have
` + "`is_env_var = true`" + `; that bypass is exposed as the ` + "`VERCEL_AUTOMATION_BYPASS_SECRET`" + ` environment variable on deployments.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project the bypass belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
				Description:   "The ID of the team the project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"secret": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Description:   "The 32-character alphanumeric secret used as the value of the `x-vercel-protection-bypass` header. If omitted, Vercel generates one.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]{32}$`),
						"secret must be a 32 character alphanumeric string.",
					),
				},
			},
			"note": schema.StringAttribute{
				Optional:    true,
				Description: "An optional note shown in the Vercel UI for this bypass. Maximum 100 characters.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(100),
				},
			},
			"is_env_var": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Whether this bypass is exposed as the `VERCEL_AUTOMATION_BYPASS_SECRET` environment variable on deployments. Exactly one bypass per project may have this set to true; promoting a different bypass automatically demotes the previous one.",
			},
			"scope": schema.StringAttribute{
				Computed:    true,
				Description: "The scope of the bypass. Always `automation-bypass` for bypasses managed by this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "The unix timestamp in milliseconds at which the bypass was created.",
			},
			"created_by": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the user who created the bypass.",
			},
		},
	}
}

type ProjectProtectionBypass struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	Secret    types.String `tfsdk:"secret"`
	Note      types.String `tfsdk:"note"`
	IsEnvVar  types.Bool   `tfsdk:"is_env_var"`
	Scope     types.String `tfsdk:"scope"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	CreatedBy types.String `tfsdk:"created_by"`
}

func convertBypass(projectID, teamID, secret string, bypass client.ProtectionBypass) ProjectProtectionBypass {
	return ProjectProtectionBypass{
		ID:        types.StringValue(fmt.Sprintf("%s/%s", projectID, secret)),
		ProjectID: types.StringValue(projectID),
		TeamID:    toTeamID(teamID),
		Secret:    types.StringValue(secret),
		Note:      types.StringPointerValue(bypass.Note),
		IsEnvVar:  types.BoolPointerValue(bypass.IsEnvVar),
		Scope:     types.StringValue(bypass.Scope),
		CreatedAt: types.Int64Value(bypass.CreatedAt),
		CreatedBy: types.StringValue(bypass.CreatedBy),
	}
}

func (r *projectProtectionBypassResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectProtectionBypass
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, bypass, err := r.client.CreateProtectionBypass(ctx, client.CreateProtectionBypassRequest{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Secret:    plan.Secret.ValueString(),
		Note:      plan.Note.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project protection bypass",
			fmt.Sprintf("Could not create protection bypass for project %s: %s", plan.ProjectID.ValueString(), err),
		)
		return
	}

	// Promote this bypass to the env-var default if the user asked for it and the API
	// didn't already make it the default (i.e. another bypass still holds that slot).
	needsPromotion := !plan.IsEnvVar.IsNull() && !plan.IsEnvVar.IsUnknown() && plan.IsEnvVar.ValueBool()
	actuallyDefault := bypass.IsEnvVar != nil && *bypass.IsEnvVar
	if needsPromotion && !actuallyDefault {
		isEnvVar := true
		updated, err := r.client.UpdateProtectionBypass(ctx, client.UpdateProtectionBypassRequest{
			TeamID:    plan.TeamID.ValueString(),
			ProjectID: plan.ProjectID.ValueString(),
			Secret:    secret,
			IsEnvVar:  &isEnvVar,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error promoting project protection bypass to env var default",
				fmt.Sprintf("Created bypass but could not set is_env_var=true: %s", err),
			)
			return
		}
		bypass = updated
	}

	result := convertBypass(plan.ProjectID.ValueString(), plan.TeamID.ValueString(), secret, bypass)
	tflog.Info(ctx, "created project protection bypass", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectProtectionBypassResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectProtectionBypass
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bypass, err := r.client.GetProtectionBypass(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), state.Secret.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project protection bypass",
			fmt.Sprintf("Could not read protection bypass for project %s: %s", state.ProjectID.ValueString(), err),
		)
		return
	}

	result := convertBypass(state.ProjectID.ValueString(), state.TeamID.ValueString(), state.Secret.ValueString(), bypass)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectProtectionBypassResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectProtectionBypass
	var state ProjectProtectionBypass
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateProtectionBypassRequest{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Secret:    state.Secret.ValueString(),
	}
	if !plan.Note.Equal(state.Note) {
		note := plan.Note.ValueString()
		updateReq.Note = &note
	}
	// Only send IsEnvVar when promoting to true. Demotion to false is handled
	// atomically by the API when some other bypass is promoted — sending an
	// explicit false here would fail the "one default must exist" invariant
	// when two sibling resources are updated in the same plan.
	if !plan.IsEnvVar.Equal(state.IsEnvVar) && !plan.IsEnvVar.IsNull() && !plan.IsEnvVar.IsUnknown() && plan.IsEnvVar.ValueBool() {
		isEnvVar := true
		updateReq.IsEnvVar = &isEnvVar
	}

	bypass, err := r.client.UpdateProtectionBypass(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project protection bypass",
			fmt.Sprintf("Could not update protection bypass for project %s: %s", plan.ProjectID.ValueString(), err),
		)
		return
	}

	result := convertBypass(plan.ProjectID.ValueString(), plan.TeamID.ValueString(), state.Secret.ValueString(), bypass)
	// When the plan demotes is_env_var to false we skip the API call (a sibling
	// bypass is being promoted in the same apply and will trigger the atomic swap).
	// Mirror that in state so Terraform sees a consistent result — the actual
	// demotion has either already happened or is about to in this apply.
	if !plan.IsEnvVar.IsNull() && !plan.IsEnvVar.IsUnknown() && !plan.IsEnvVar.ValueBool() {
		result.IsEnvVar = types.BoolValue(false)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectProtectionBypassResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectProtectionBypass
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProtectionBypass(ctx, client.DeleteProtectionBypassRequest{
		TeamID:                      state.TeamID.ValueString(),
		ProjectID:                   state.ProjectID.ValueString(),
		Secret:                      state.Secret.ValueString(),
		PromoteReplacementIfDefault: state.IsEnvVar.ValueBool(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project protection bypass",
			fmt.Sprintf("Could not delete protection bypass for project %s: %s", state.ProjectID.ValueString(), err),
		)
		return
	}
}

func (r *projectProtectionBypassResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, secret, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project protection bypass",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/secret\" or \"project_id/secret\"", req.ID),
		)
		return
	}

	bypass, err := r.client.GetProtectionBypass(ctx, projectID, teamID, secret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing project protection bypass",
			fmt.Sprintf("Could not read protection bypass for project %s: %s", projectID, err),
		)
		return
	}

	result := convertBypass(projectID, teamID, secret, bypass)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
