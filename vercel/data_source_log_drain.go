package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &logDrainDataSource{}
	_ datasource.DataSourceWithConfigure = &logDrainDataSource{}
)

func newLogDrainDataSource() datasource.DataSource {
	return &logDrainDataSource{}
}

type logDrainDataSource struct {
	client *client.Client
}

func (d *logDrainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_drain"
}

func (d *logDrainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Schema returns the schema information for an logDrain data source
func (r *logDrainDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Log Drain.

Log Drains collect all of your logs using a service specializing in storing app logs.

Teams on Pro and Enterprise plans can subscribe to log drains that are generic and configurable from the Vercel dashboard without creating an integration. This allows you to use a HTTP service to receive logs through Vercel's log drains.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Log Drain.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Log Drain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"delivery_format": schema.StringAttribute{
				Description: "The format log data should be delivered in. Can be `json` or `ndjson`.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The human-readable name of the Log Drain.",
				Computed:    true,
			},
			"environments": schema.SetAttribute{
				Description: "Logs from the selected environments will be forwarded to your webhook. At least one must be present.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"headers": schema.MapAttribute{
				Description: "Custom headers to include in requests to the log drain endpoint.",
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
			},
			"project_ids": schema.SetAttribute{
				Description: "A list of project IDs that the log drain should be associated with. Logs from these projects will be sent log events to the specified endpoint. If omitted, logs will be sent for all projects.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"sampling_rate": schema.Float64Attribute{
				Description: "A ratio of logs matching the sampling rate will be sent to your log drain. Should be a value between 0 and 1. If unspecified, all logs are sent.",
				Computed:    true,
			},
			"sources": schema.SetAttribute{
				Description: "A set of sources that the log drain should send logs for. Valid values are `static`, `edge`, `external`, `build`, `lambda`, `firewall`, and `redirect`.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"endpoint": schema.StringAttribute{
				Description: "Logs will be sent as POST requests to this URL. The endpoint will be verified, and must return a `200` status code and an `x-vercel-verify` header taken from the endpoint_verification data source. The value the `x-vercel-verify` header should be can be read from the `vercel_endpoint_verification_code` data source.",
				Required:    true,
			},
		},
	}
}

type LogDrainWithoutSecret struct {
	ID             types.String  `tfsdk:"id"`
	TeamID         types.String  `tfsdk:"team_id"`
	Name           types.String  `tfsdk:"name"`
	DeliveryFormat types.String  `tfsdk:"delivery_format"`
	Environments   types.Set     `tfsdk:"environments"`
	Headers        types.Map     `tfsdk:"headers"`
	ProjectIDs     types.Set     `tfsdk:"project_ids"`
	SamplingRate   types.Float64 `tfsdk:"sampling_rate"`
	Sources        types.Set     `tfsdk:"sources"`
	Endpoint       types.String  `tfsdk:"endpoint"`
}

func responseToLogDrainWithoutSecret(ctx context.Context, out client.LogDrain) (l LogDrainWithoutSecret, diags diag.Diagnostics) {
	projectIDs, diags := types.SetValueFrom(ctx, types.StringType, out.ProjectIDs)
	if diags.HasError() {
		return l, diags
	}

	environments, diags := types.SetValueFrom(ctx, types.StringType, out.Environments)
	if diags.HasError() {
		return l, diags
	}

	sources, diags := types.SetValueFrom(ctx, types.StringType, out.Sources)
	if diags.HasError() {
		return l, diags
	}

	headers, diags := types.MapValueFrom(ctx, types.StringType, out.Headers)
	if diags.HasError() {
		return l, diags
	}

	return LogDrainWithoutSecret{
		ID:             types.StringValue(out.ID),
		TeamID:         toTeamID(out.TeamID),
		Name:           types.StringValue(out.Name),
		DeliveryFormat: types.StringValue(out.DeliveryFormat),
		SamplingRate:   types.Float64PointerValue(out.SamplingRate),
		Endpoint:       types.StringValue(out.Endpoint),
		Environments:   environments,
		Headers:        headers,
		Sources:        sources,
		ProjectIDs:     projectIDs,
	}, nil
}

// Read will read the logDrain information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *logDrainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config LogDrainWithoutSecret
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetLogDrain(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Log Drain",
			fmt.Sprintf("Could not get Log Drain %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToLogDrainWithoutSecret(ctx, out)
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
