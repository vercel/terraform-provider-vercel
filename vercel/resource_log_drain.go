package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &logDrainResource{}
	_ resource.ResourceWithConfigure   = &logDrainResource{}
	_ resource.ResourceWithImportState = &logDrainResource{}
)

func newLogDrainResource() resource.Resource {
	return &logDrainResource{}
}

type logDrainResource struct {
	client *client.Client
}

func (r *logDrainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_drain"
}

func (r *logDrainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *logDrainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Configurable Log Drain resource.

~> For Log Drain integrations, please see the [Integration Log Drain docs](https://vercel.com/docs/observability/log-drains#log-drains-integration).

Log Drains collect all of your logs using a service specializing in storing app logs.

Teams on Pro and Enterprise plans can subscribe to log drains that are generic and configurable from the Vercel dashboard without creating an integration. This allows you to use a HTTP service to receive logs through Vercel's log drains.

~> Only Pro and Enterprise teams can create Configurable Log Drains.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Log Drain.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Log Drain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"delivery_format": schema.StringAttribute{
				Description:   "The format log data should be delivered in. Can be `json` or `ndjson`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("json", "ndjson"),
				},
			},
			"environments": schema.SetAttribute{
				Description:   "Logs from the selected environments will be forwarded to your webhook. At least one must be present.",
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
				Required:      true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf("production", "preview")),
					setvalidator.SizeAtLeast(1),
				},
			},
			"headers": schema.MapAttribute{
				Description:   "Custom headers to include in requests to the log drain endpoint.",
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
				Optional:      true,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(5),
				},
			},
			"project_ids": schema.SetAttribute{
				Description:   "A list of project IDs that the log drain should be associated with. Logs from these projects will be sent log events to the specified endpoint. If omitted, logs will be sent for all projects.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"sampling_rate": schema.Float64Attribute{
				Description:   "A ratio of logs matching the sampling rate will be sent to your log drain. Should be a value between 0 and 1. If unspecified, all logs are sent.",
				Optional:      true,
				PlanModifiers: []planmodifier.Float64{float64planmodifier.RequiresReplace()},
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
					float64validator.AtMost(1),
				},
			},
			"secret": schema.StringAttribute{
				Description:   "A custom secret to be used for signing log events. You can use this secret to verify that log events are coming from Vercel and are not tampered with. See https://vercel.com/docs/observability/log-drains/log-drains-reference#secure-log-drains for full info.",
				Optional:      true,
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(32),
				},
			},
			"sources": schema.SetAttribute{
				Description:   "A set of sources that the log drain should send logs for. Valid values are `static`, `edge`, `external`, `build`, `lambda` and `firewall`.",
				Required:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf("static", "edge", "external", "build", "lambda", "firewall")),
					setvalidator.SizeAtLeast(1),
				},
			},
			"endpoint": schema.StringAttribute{
				Description:   "Logs will be sent as POST requests to this URL. The endpoint will be verified, and must return a `200` status code and an `x-vercel-verify` header taken from the endpoint_verification data source. The value the `x-vercel-verify` header should be can be read from the `vercel_endpoint_verification_code` data source.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type LogDrain struct {
	ID             types.String  `tfsdk:"id"`
	TeamID         types.String  `tfsdk:"team_id"`
	DeliveryFormat types.String  `tfsdk:"delivery_format"`
	Environments   types.Set     `tfsdk:"environments"`
	Headers        types.Map     `tfsdk:"headers"`
	ProjectIDs     types.Set     `tfsdk:"project_ids"`
	SamplingRate   types.Float64 `tfsdk:"sampling_rate"`
	Secret         types.String  `tfsdk:"secret"`
	Sources        types.Set     `tfsdk:"sources"`
	Endpoint       types.String  `tfsdk:"endpoint"`
}

func responseToLogDrain(ctx context.Context, out client.LogDrain, secret types.String) (LogDrain, diag.Diagnostics) {
	projectIDs, diags := types.SetValueFrom(ctx, types.StringType, out.ProjectIDs)
	if diags.HasError() {
		return LogDrain{}, diags
	}

	environments, diags := types.SetValueFrom(ctx, types.StringType, out.Environments)
	if diags.HasError() {
		return LogDrain{}, diags
	}

	sources, diags := types.SetValueFrom(ctx, types.StringType, out.Sources)
	if diags.HasError() {
		return LogDrain{}, diags
	}

	headers, diags := types.MapValueFrom(ctx, types.StringType, out.Headers)
	if diags.HasError() {
		return LogDrain{}, diags
	}

	if secret.IsNull() || secret.IsUnknown() {
		secret = types.StringValue(out.Secret)
	}

	return LogDrain{
		ID:             types.StringValue(out.ID),
		TeamID:         toTeamID(out.TeamID),
		DeliveryFormat: types.StringValue(out.DeliveryFormat),
		SamplingRate:   types.Float64PointerValue(out.SamplingRate),
		Secret:         secret,
		Endpoint:       types.StringValue(out.Endpoint),
		Environments:   environments,
		Headers:        headers,
		Sources:        sources,
		ProjectIDs:     projectIDs,
	}, nil
}

func (r *logDrainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LogDrain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environments []string
	diags = plan.Environments.ElementsAs(ctx, &environments, false)
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

	var sources []string
	diags = plan.Sources.ElementsAs(ctx, &sources, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateLogDrain(ctx, client.CreateLogDrainRequest{
		TeamID:         plan.TeamID.ValueString(),
		DeliveryFormat: plan.DeliveryFormat.ValueString(),
		Environments:   environments,
		Headers:        headers,
		ProjectIDs:     projectIDs,
		SamplingRate:   plan.SamplingRate.ValueFloat64(),
		Secret:         plan.Secret.ValueString(),
		Sources:        sources,
		Endpoint:       plan.Endpoint.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Log Drain",
			"Could not create Log Drain, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToLogDrain(ctx, out, plan.Secret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created Log Drain", map[string]any{
		"team_id":      plan.TeamID.ValueString(),
		"log_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *logDrainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LogDrain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetLogDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Log Drain",
			fmt.Sprintf("Could not get Log Drain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToLogDrain(ctx, out, state.Secret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read log drain", map[string]any{
		"team_id":      result.TeamID.ValueString(),
		"log_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *logDrainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a Log Drain is not supported",
		"Updating a Log Drain is not supported",
	)
}

func (r *logDrainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LogDrain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteLogDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting log drain",
			fmt.Sprintf(
				"Could not delete Log Drain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Log Drain", map[string]any{
		"team_id":      state.TeamID.ValueString(),
		"log_drain_id": state.ID.ValueString(),
	})
}

func (r *logDrainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Log Drain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/log_drain_id\" or \"log_drain_id\"", req.ID),
		)
	}

	out, err := r.client.GetLogDrain(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Log Drain",
			fmt.Sprintf("Could not get Log Drain %s %s, unexpected error: %s",
				teamID,
				id,
				err,
			),
		)
		return
	}

	result, diags := responseToLogDrain(ctx, out, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "import log drain", map[string]any{
		"team_id":      result.TeamID.ValueString(),
		"log_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
