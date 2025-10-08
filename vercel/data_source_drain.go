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
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &drainDataSource{}
	_ datasource.DataSourceWithConfigure = &drainDataSource{}
)

func newDrainDataSource() datasource.DataSource {
	return &drainDataSource{}
}

type drainDataSource struct {
	client *client.Client
}

func (d *drainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_drain"
}

func (d *drainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *drainDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Drain.

Drains collect various types of data including logs, traces, analytics, and speed insights from your Vercel projects.
This is a more generic version of log drains that supports multiple data types and delivery methods.

Teams on Pro and Enterprise plans can create configurable drains from the Vercel dashboard.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Drain.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Drain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"name": schema.StringAttribute{
				Description: "The name of the Drain.",
				Computed:    true,
			},
			"projects": schema.StringAttribute{
				Description: "Whether to include all projects or a specific set. Valid values are `all` or `some`.",
				Computed:    true,
			},
			"project_ids": schema.SetAttribute{
				Description: "A list of project IDs that the drain should be associated with. Only valid when `projects` is set to `some`.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"filter": schema.StringAttribute{
				Description: "A filter expression applied to incoming data.",
				Computed:    true,
			},
			"schemas": schema.MapAttribute{
				Description: "A map of schema configurations. Keys can be `log`, `trace`, `analytics`, or `speed_insights`.",
				Computed:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"version": types.StringType,
					},
				},
			},
			"delivery": schema.SingleNestedAttribute{
				Description: "Configuration for how data should be delivered.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The delivery type. Valid values are `http` or `otlphttp`.",
						Computed:    true,
					},
					"endpoint": schema.SingleNestedAttribute{
						Description: "Endpoint configuration. Contains `url` for HTTP or `traces` for OTLP.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Description: "The endpoint URL for HTTP delivery type.",
								Computed:    true,
							},
							"traces": schema.StringAttribute{
								Description: "The traces endpoint URL for OTLP delivery type.",
								Computed:    true,
							},
						},
					},
					"encoding": schema.StringAttribute{
						Description: "The encoding format. Valid values are `json`, `ndjson` (for HTTP) or `proto` (for OTLP).",
						Computed:    true,
					},
					"compression": schema.StringAttribute{
						Description: "The compression method. Valid values are `gzip` or `none`. Only applicable for HTTP delivery.",
						Computed:    true,
					},
					"headers": schema.MapAttribute{
						Description: "Custom headers to include in HTTP requests.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"sampling": schema.SetNestedAttribute{
				Description: "Sampling configuration for the drain.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The sampling type. Only `head_sampling` is supported.",
							Computed:    true,
						},
						"rate": schema.Float64Attribute{
							Description: "The sampling rate from 0 to 1 (e.g., 0.1 for 10%).",
							Computed:    true,
						},
						"environment": schema.StringAttribute{
							Description: "The environment to apply sampling to. Valid values are `production` or `preview`.",
							Computed:    true,
						},
						"request_path": schema.StringAttribute{
							Description: "Request path prefix to apply the sampling rule to.",
							Computed:    true,
						},
					},
				},
			},
			"transforms": schema.SetNestedAttribute{
				Description: "Transform configurations for the drain.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The transform ID.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the drain.",
				Computed:    true,
			},
		},
	}
}

type DrainDataSource struct {
	ID         types.String `tfsdk:"id"`
	TeamID     types.String `tfsdk:"team_id"`
	Name       types.String `tfsdk:"name"`
	Projects   types.String `tfsdk:"projects"`
	ProjectIds types.Set    `tfsdk:"project_ids"`
	Filter     types.String `tfsdk:"filter"`
	Schemas    types.Map    `tfsdk:"schemas"`
	Delivery   types.Object `tfsdk:"delivery"`
	Sampling   types.Set    `tfsdk:"sampling"`
	Transforms types.Set    `tfsdk:"transforms"`
	Status     types.String `tfsdk:"status"`
}

type DeliveryDataSourceModel struct {
	Type        types.String `tfsdk:"type"`
	Endpoint    types.Object `tfsdk:"endpoint"`
	Encoding    types.String `tfsdk:"encoding"`
	Compression types.String `tfsdk:"compression"`
	Headers     types.Map    `tfsdk:"headers"`
}

type EndpointDataSourceModel struct {
	URL    types.String `tfsdk:"url"`
	Traces types.String `tfsdk:"traces"`
}

func responseToDrainDataSource(ctx context.Context, out client.Drain) (DrainDataSource, diag.Diagnostics) {
	var diags diag.Diagnostics

	projectIds, d := types.SetValueFrom(ctx, types.StringType, out.ProjectIds)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	schemasMap := make(map[string]SchemaVersionModel)
	for k, v := range out.Schemas {
		if schemaMap, ok := v.(map[string]any); ok {
			if version, exists := schemaMap["version"]; exists {
				if versionStr, ok := version.(string); ok {
					schemasMap[k] = SchemaVersionModel{
						Version: types.StringValue(versionStr),
					}
				}
			}
		}
	}

	schemas, d := types.MapValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"version": types.StringType,
		},
	}, schemasMap)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	deliveryHeaders, d := types.MapValueFrom(ctx, types.StringType, out.Delivery.Headers)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	deliveryModel := DeliveryDataSourceModel{
		Type:        types.StringValue(out.Delivery.Type),
		Encoding:    types.StringValue(out.Delivery.Encoding),
		Compression: types.StringPointerValue(out.Delivery.Compression),
		Headers:     deliveryHeaders,
	}

	var endpointModel EndpointDataSourceModel
	if endpoint, ok := out.Delivery.Endpoint.(string); ok {
		endpointModel = EndpointDataSourceModel{
			URL:    types.StringValue(endpoint),
			Traces: types.StringNull(),
		}
	} else if otlpEndpoint, ok := out.Delivery.Endpoint.(map[string]any); ok {
		if traces, exists := otlpEndpoint["traces"]; exists {
			if tracesStr, ok := traces.(string); ok {
				endpointModel = EndpointDataSourceModel{
					URL:    types.StringNull(),
					Traces: types.StringValue(tracesStr),
				}
			}
		}
	}

	endpoint, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"url":    types.StringType,
		"traces": types.StringType,
	}, endpointModel)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	deliveryModel.Endpoint = endpoint

	delivery, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"type":        types.StringType,
		"endpoint":    types.ObjectType{AttrTypes: map[string]attr.Type{"url": types.StringType, "traces": types.StringType}},
		"encoding":    types.StringType,
		"compression": types.StringType,
		"headers":     types.MapType{ElemType: types.StringType},
	}, deliveryModel)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	samplingModels := make([]SamplingModel, len(out.Sampling))
	for i, s := range out.Sampling {
		samplingModels[i] = SamplingModel{
			Type:        types.StringValue(s.Type),
			Rate:        types.Float64Value(s.Rate),
			Environment: types.StringPointerValue(s.Env),
			RequestPath: types.StringPointerValue(s.RequestPath),
		}
	}

	sampling, d := types.SetValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"rate":         types.Float64Type,
			"environment":  types.StringType,
			"request_path": types.StringType,
		},
	}, samplingModels)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	transformModels := make([]TransformModel, len(out.Transforms))
	for i, t := range out.Transforms {
		transformModels[i] = TransformModel{
			ID: types.StringValue(t.ID),
		}
	}

	transforms, d := types.SetValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id": types.StringType,
		},
	}, transformModels)
	diags.Append(d...)
	if diags.HasError() {
		return DrainDataSource{}, diags
	}

	return DrainDataSource{
		ID:         types.StringValue(out.ID),
		TeamID:     toTeamID(out.TeamID),
		Name:       types.StringValue(out.Name),
		Projects:   types.StringValue(out.Projects),
		ProjectIds: projectIds,
		Filter:     types.StringPointerValue(out.Filter),
		Schemas:    schemas,
		Delivery:   delivery,
		Sampling:   sampling,
		Transforms: transforms,
		Status:     types.StringValue(out.Status),
	}, diags
}

// Read will read the drain information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *drainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DrainDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetDrain(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Drain",
			fmt.Sprintf("Could not get Drain %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToDrainDataSource(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read drain", map[string]any{
		"team_id":  result.TeamID.ValueString(),
		"drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
