package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &traceDrainDataSource{}
	_ datasource.DataSourceWithConfigure = &traceDrainDataSource{}
)

func newTraceDrainDataSource() datasource.DataSource {
	return &traceDrainDataSource{}
}

type traceDrainDataSource struct {
	client *client.Client
}

func (d *traceDrainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trace_drain"
}

func (d *traceDrainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *traceDrainDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Trace Drain.

Trace Drains forward OpenTelemetry trace data from your deployments to an OTLP/HTTP compatible endpoint.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Trace Drain.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Trace Drain exists under. Required when reading a team resource if a default team has not been set in the provider.",
			},
			"delivery_format": schema.StringAttribute{
				Description: "The OTLP/HTTP format trace data is delivered in. Can be `json` or `proto`.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The human-readable name of the Trace Drain.",
				Computed:    true,
			},
			"headers": schema.MapAttribute{
				Description: "Custom headers included in requests to the trace drain endpoint.",
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
			},
			"project_ids": schema.SetAttribute{
				Description: "A list of project IDs that the trace drain is associated with. If omitted, traces are sent for all projects.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"sampling_rules": schema.ListNestedAttribute{
				Description: "Ordered sampling rules for traces sent to this drain.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"rate": schema.Float64Attribute{
							Description: "Sampling rate from 0 to 1.",
							Computed:    true,
						},
						"environment": schema.StringAttribute{
							Description: "Environment this sampling rule applies to.",
							Computed:    true,
						},
						"request_path": schema.StringAttribute{
							Description: "Request path prefix this sampling rule applies to.",
							Computed:    true,
						},
					},
				},
			},
			"endpoint": schema.StringAttribute{
				Description: "The OTLP/HTTP traces endpoint.",
				Computed:    true,
			},
		},
	}
}

type TraceDrainWithoutSecret struct {
	ID             types.String `tfsdk:"id"`
	TeamID         types.String `tfsdk:"team_id"`
	Name           types.String `tfsdk:"name"`
	DeliveryFormat types.String `tfsdk:"delivery_format"`
	Headers        types.Map    `tfsdk:"headers"`
	ProjectIDs     types.Set    `tfsdk:"project_ids"`
	SamplingRules  types.List   `tfsdk:"sampling_rules"`
	Endpoint       types.String `tfsdk:"endpoint"`
}

func responseToTraceDrainWithoutSecret(ctx context.Context, out client.TraceDrain) (l TraceDrainWithoutSecret, diags diag.Diagnostics) {
	projectIDs, diags := types.SetValueFrom(ctx, types.StringType, out.ProjectIDs)
	if diags.HasError() {
		return l, diags
	}

	headers, diags := types.MapValueFrom(ctx, types.StringType, out.Headers)
	if diags.HasError() {
		return l, diags
	}

	samplingRules, diags := traceDrainSamplingRulesFromAPI(ctx, out.SamplingRules, types.ListValueMust(traceDrainSamplingRuleAttrType, []attr.Value{}))
	if diags.HasError() {
		return l, diags
	}

	return TraceDrainWithoutSecret{
		ID:             types.StringValue(out.ID),
		TeamID:         toTeamID(out.TeamID),
		Name:           types.StringValue(out.Name),
		DeliveryFormat: types.StringValue(out.DeliveryFormat),
		Headers:        headers,
		ProjectIDs:     projectIDs,
		SamplingRules:  samplingRules,
		Endpoint:       types.StringValue(out.Endpoint),
	}, nil
}

func (d *traceDrainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TraceDrainWithoutSecret
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetTraceDrain(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Trace Drain",
			fmt.Sprintf("Could not get Trace Drain %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToTraceDrainWithoutSecret(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read trace drain", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"trace_drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
