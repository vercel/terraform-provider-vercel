package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &projectTracingResource{}
	_ resource.ResourceWithConfigure   = &projectTracingResource{}
	_ resource.ResourceWithImportState = &projectTracingResource{}
)

func newProjectTracingResource() resource.Resource {
	return &projectTracingResource{}
}

type projectTracingResource struct {
	client *client.Client
}

func (r *projectTracingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tracing"
}

func (r *projectTracingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectTracingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Tracing resource.

Project Tracing sends OpenTelemetry traces from a project's deployments to Vercel. Ordered sampling rules can limit which traces are retained.

Deleting this resource disables tracing for the project.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Project to configure tracing for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether tracing is enabled for the Project.",
			},
			"sampling_rules": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Ordered head-sampling rules for traces. If omitted, all traces are retained.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(10),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"rate": schema.Float64Attribute{
							Required:    true,
							Description: "Sampling rate from 0 to 1.",
							Validators: []validator.Float64{
								float64validator.Between(0, 1),
							},
						},
						"environment": schema.StringAttribute{
							Optional:    true,
							Description: "Environment to apply this sampling rule to. Can be `production` or `preview`.",
							Validators: []validator.String{
								stringvalidator.OneOf("production", "preview"),
							},
						},
						"request_path": schema.StringAttribute{
							Optional:    true,
							Description: "Request path prefix to apply this sampling rule to.",
						},
					},
				},
			},
		},
	}
}

type ProjectTracing struct {
	ID            types.String `tfsdk:"id"`
	ProjectID     types.String `tfsdk:"project_id"`
	TeamID        types.String `tfsdk:"team_id"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	SamplingRules types.List   `tfsdk:"sampling_rules"`
}

func responseToProjectTracing(ctx context.Context, out client.ProjectTracing, preferredSamplingRules types.List) (ProjectTracing, diag.Diagnostics) {
	samplingRules, diags := traceDrainSamplingRulesFromAPI(ctx, out.SamplingRules, preferredSamplingRules)
	if diags.HasError() {
		return ProjectTracing{}, diags
	}
	return ProjectTracing{
		ID:            types.StringValue(out.ProjectID),
		ProjectID:     types.StringValue(out.ProjectID),
		TeamID:        toTeamID(out.TeamID),
		Enabled:       types.BoolValue(out.Enabled),
		SamplingRules: samplingRules,
	}, diags
}

func (r *projectTracingResource) apply(ctx context.Context, plan ProjectTracing, respDiagnostics *diag.Diagnostics) (ProjectTracing, bool) {
	samplingRules, diags := traceDrainSamplingRulesToClient(ctx, plan.SamplingRules)
	respDiagnostics.Append(diags...)
	if respDiagnostics.HasError() {
		return ProjectTracing{}, false
	}

	out, err := r.client.UpdateProjectTracing(ctx, client.ProjectTracing{
		TeamID:        plan.TeamID.ValueString(),
		ProjectID:     plan.ProjectID.ValueString(),
		Enabled:       plan.Enabled.ValueBool(),
		SamplingRules: samplingRules,
	})
	if err != nil {
		respDiagnostics.AddError(
			"Error updating Project Tracing",
			"Could not update Project Tracing, unexpected error: "+err.Error(),
		)
		return ProjectTracing{}, false
	}

	result, diags := responseToProjectTracing(ctx, out, plan.SamplingRules)
	respDiagnostics.Append(diags...)
	return result, !respDiagnostics.HasError()
}

func (r *projectTracingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectTracing
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, ok := r.apply(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	tflog.Info(ctx, "created project tracing configuration", map[string]any{"team_id": result.TeamID.ValueString(), "project_id": result.ProjectID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectTracingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectTracing
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProjectTracing(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Tracing",
			fmt.Sprintf("Could not get Project Tracing %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	result, diags := responseToProjectTracing(ctx, out, state.SamplingRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read project tracing configuration", map[string]any{"team_id": result.TeamID.ValueString(), "project_id": result.ProjectID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectTracingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectTracing
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, ok := r.apply(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	tflog.Info(ctx, "updated project tracing configuration", map[string]any{"team_id": result.TeamID.ValueString(), "project_id": result.ProjectID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectTracingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectTracing
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProjectTracing(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Project Tracing",
			fmt.Sprintf("Could not delete Project Tracing %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}
	tflog.Info(ctx, "deleted project tracing configuration", map[string]any{"team_id": state.TeamID.ValueString(), "project_id": state.ProjectID.ValueString()})
}

func (r *projectTracingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Project Tracing",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetProjectTracing(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Tracing",
			fmt.Sprintf("Could not get Project Tracing %s %s, unexpected error: %s", teamID, projectID, err),
		)
		return
	}
	result, diags := responseToProjectTracing(ctx, out, types.ListNull(traceDrainSamplingRuleAttrType))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
