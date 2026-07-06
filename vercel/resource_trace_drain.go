package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &traceDrainResource{}
	_ resource.ResourceWithConfigure   = &traceDrainResource{}
	_ resource.ResourceWithImportState = &traceDrainResource{}
)

func newTraceDrainResource() resource.Resource {
	return &traceDrainResource{}
}

type traceDrainResource struct {
	client *client.Client
}

func (r *traceDrainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trace_drain"
}

func (r *traceDrainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *traceDrainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a configurable Trace Drain resource.

Trace Drains forward OpenTelemetry trace data from your deployments to an OTLP/HTTP compatible endpoint.

~> Only Pro and Enterprise teams can create configurable Trace Drains.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Trace Drain.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Trace Drain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"delivery_format": schema.StringAttribute{
				Description:   "The OTLP/HTTP format trace data should be delivered in. Can be `json` or `proto`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("json", "proto"),
				},
			},
			"name": schema.StringAttribute{
				Description:   "A human-readable name for the trace drain.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"headers": schema.MapAttribute{
				Description:   "Custom headers to include in requests to the trace drain endpoint.",
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
				Optional:      true,
				Sensitive:     true,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(5),
				},
			},
			"project_ids": schema.SetAttribute{
				Description:   "A list of project IDs that the trace drain should be associated with. If omitted, traces will be sent for all projects.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"sampling_rules": schema.ListNestedAttribute{
				Description:   "Ordered sampling rules for traces sent to this drain. If omitted, all traces are sent.",
				Optional:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"rate": schema.Float64Attribute{
							Description: "Sampling rate from 0 to 1.",
							Required:    true,
							Validators: []validator.Float64{
								float64validator.AtLeast(0),
								float64validator.AtMost(1),
							},
						},
						"environment": schema.StringAttribute{
							Description: "Environment to apply this sampling rule to. Can be `production` or `preview`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("production", "preview"),
							},
						},
						"request_path": schema.StringAttribute{
							Description: "Request path prefix to apply this sampling rule to.",
							Optional:    true,
						},
					},
				},
			},
			"secret": schema.StringAttribute{
				Description:   "A custom secret used to sign trace drain events. If omitted, Vercel generates one.",
				Optional:      true,
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(32),
				},
			},
			"endpoint": schema.StringAttribute{
				Description:   "The OTLP/HTTP traces endpoint. This should be the full traces endpoint URL, commonly ending in `/v1/traces`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type TraceDrain struct {
	ID             types.String `tfsdk:"id"`
	TeamID         types.String `tfsdk:"team_id"`
	Name           types.String `tfsdk:"name"`
	DeliveryFormat types.String `tfsdk:"delivery_format"`
	Headers        types.Map    `tfsdk:"headers"`
	ProjectIDs     types.Set    `tfsdk:"project_ids"`
	SamplingRules  types.List   `tfsdk:"sampling_rules"`
	Secret         types.String `tfsdk:"secret"`
	Endpoint       types.String `tfsdk:"endpoint"`
}

func responseToTraceDrain(ctx context.Context, out client.TraceDrain, secret types.String, preferredSamplingRules types.List) (TraceDrain, diag.Diagnostics) {
	projectIDs, diags := types.SetValueFrom(ctx, types.StringType, out.ProjectIDs)
	if diags.HasError() {
		return TraceDrain{}, diags
	}

	headers, diags := types.MapValueFrom(ctx, types.StringType, out.Headers)
	if diags.HasError() {
		return TraceDrain{}, diags
	}

	samplingRules, diags := traceDrainSamplingRulesFromAPI(ctx, out.SamplingRules, preferredSamplingRules)
	if diags.HasError() {
		return TraceDrain{}, diags
	}

	if secret.IsNull() || secret.IsUnknown() {
		secret = types.StringValue(out.Secret)
	}

	return TraceDrain{
		ID:             types.StringValue(out.ID),
		TeamID:         toTeamID(out.TeamID),
		Name:           types.StringValue(out.Name),
		DeliveryFormat: types.StringValue(out.DeliveryFormat),
		Headers:        headers,
		ProjectIDs:     projectIDs,
		SamplingRules:  samplingRules,
		Secret:         secret,
		Endpoint:       types.StringValue(out.Endpoint),
	}, nil
}

func (r *traceDrainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TraceDrain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var headers map[string]string
	diags = plan.Headers.ElementsAs(ctx, &headers, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectIDs []string
	diags = plan.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	samplingRules, diags := traceDrainSamplingRulesToClient(ctx, plan.SamplingRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateTraceDrain(ctx, client.CreateTraceDrainRequest{
		TeamID:         plan.TeamID.ValueString(),
		Name:           plan.Name.ValueString(),
		DeliveryFormat: plan.DeliveryFormat.ValueString(),
		Headers:        headers,
		ProjectIDs:     projectIDs,
		SamplingRules:  samplingRules,
		Secret:         plan.Secret.ValueString(),
		Endpoint:       plan.Endpoint.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Trace Drain",
			"Could not create Trace Drain, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToTraceDrain(ctx, out, plan.Secret, plan.SamplingRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Headers.IsNull() || plan.Headers.IsUnknown() {
		result.Headers = types.MapNull(types.StringType)
	}

	tflog.Info(ctx, "created Trace Drain", map[string]any{
		"team_id":        plan.TeamID.ValueString(),
		"trace_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *traceDrainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TraceDrain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetTraceDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Trace Drain",
			fmt.Sprintf("Could not get Trace Drain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToTraceDrain(ctx, out, state.Secret, state.SamplingRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.Headers.IsNull() || state.Headers.IsUnknown() {
		result.Headers = types.MapNull(types.StringType)
	}

	tflog.Info(ctx, "read trace drain", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"trace_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *traceDrainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a Trace Drain is not supported",
		"Updating a Trace Drain is not supported",
	)
}

func (r *traceDrainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TraceDrain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTraceDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting trace drain",
			fmt.Sprintf(
				"Could not delete Trace Drain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Trace Drain", map[string]any{
		"team_id":        state.TeamID.ValueString(),
		"trace_drain_id": state.ID.ValueString(),
	})
}

func (r *traceDrainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Trace Drain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/trace_drain_id\" or \"trace_drain_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetTraceDrain(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Trace Drain",
			fmt.Sprintf("Could not get Trace Drain %s %s, unexpected error: %s",
				teamID,
				id,
				err,
			),
		)
		return
	}

	result, diags := responseToTraceDrain(ctx, out, types.StringNull(), types.ListNull(traceDrainSamplingRuleAttrType))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "import trace drain", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"trace_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
