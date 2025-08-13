package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &drainResource{}
	_ resource.ResourceWithConfigure   = &drainResource{}
	_ resource.ResourceWithImportState = &drainResource{}
)

func newDrainResource() resource.Resource {
	return &drainResource{}
}

type drainResource struct {
	client *client.Client
}

func (r *drainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_drain"
}

func (r *drainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *drainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Configurable Drain resource.

Drains collect various types of data including logs, traces, analytics, and speed insights from your Vercel projects.
This is a more generic version of log drains that supports multiple data types and delivery methods.

Teams on Pro and Enterprise plans can create configurable drains from the Vercel dashboard.

~> Only Pro and Enterprise teams can create Configurable Drains.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Drain.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Drain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Drain.",
				Required:    true,
			},
			"projects": schema.StringAttribute{
				Description: "Whether to include all projects or a specific set. Valid values are `all` or `some`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "some"),
				},
			},
			"project_ids": schema.SetAttribute{
				Description: "A list of project IDs that the drain should be associated with. Required when `projects` is `some`.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"filter": schema.StringAttribute{
				Description: "A filter expression to apply to incoming data.",
				Optional:    true,
			},
			"schemas": schema.MapAttribute{
				Description: "A map of schema configurations. Keys can be `log`, `trace`, `analytics`, or `speed_insights`.",
				Required:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"version": types.StringType,
					},
				},
			},
			"delivery": schema.SingleNestedAttribute{
				Description: "Configuration for how data should be delivered.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The delivery type. Valid values are `http` or `otlphttp`",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("http", "otlphttp"),
						},
					},
					"endpoint": schema.SingleNestedAttribute{
						Description: "Endpoint configuration. Use `url` for HTTP or `traces` for OTLP.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Description: "The endpoint URL for HTTP delivery type.",
								Optional:    true,
							},
							"traces": schema.StringAttribute{
								Description: "The traces endpoint URL for OTLP delivery type.",
								Optional:    true,
							},
						},
					},
					"encoding": schema.StringAttribute{
						Description: "The encoding format. Valid values are `json`, `ndjson`, or `proto` (for OTLP).",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("json", "ndjson", "proto"),
						},
					},
					"compression": schema.StringAttribute{
						Description: "The compression method. Valid values are `gzip` or `none`.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("gzip", "none"),
						},
					},
					"headers": schema.MapAttribute{
						Description: "Custom headers to include in HTTP requests.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.Map{
							mapvalidator.SizeAtMost(10),
						},
					},
					"secret": schema.StringAttribute{
						Description: "A secret for signing requests.",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
			"sampling": schema.SetNestedAttribute{
				Description: "Sampling configuration for the drain.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The sampling type. Only `head_sampling` is supported.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("head_sampling"),
							},
						},
						"rate": schema.Float64Attribute{
							Description: "The sampling rate from 0 to 1 (e.g., 0.1 for 10%).",
							Required:    true,
							Validators: []validator.Float64{
								float64validator.AtLeast(0),
								float64validator.AtMost(1),
							},
						},
						"environment": schema.StringAttribute{
							Description: "The environment to apply sampling to. Valid values are `production` or `preview`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("production", "preview"),
							},
						},
						"request_path": schema.StringAttribute{
							Description: "Request path prefix to apply the sampling rule to.",
							Optional:    true,
						},
					},
				},
			},
			"transforms": schema.SetNestedAttribute{
				Description: "Transform configurations for the drain.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The transform ID.",
							Required:    true,
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

type Drain struct {
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

type DeliveryModel struct {
	Type        types.String `tfsdk:"type"`
	Endpoint    types.Object `tfsdk:"endpoint"`
	Encoding    types.String `tfsdk:"encoding"`
	Compression types.String `tfsdk:"compression"`
	Headers     types.Map    `tfsdk:"headers"`
	Secret      types.String `tfsdk:"secret"`
}

type EndpointModel struct {
	URL    types.String `tfsdk:"url"`
	Traces types.String `tfsdk:"traces"`
}

type SamplingModel struct {
	Type        types.String  `tfsdk:"type"`
	Rate        types.Float64 `tfsdk:"rate"`
	Environment types.String  `tfsdk:"environment"`
	RequestPath types.String  `tfsdk:"request_path"`
}

type TransformModel struct {
	ID types.String `tfsdk:"id"`
}

type SchemaVersionModel struct {
	Version types.String `tfsdk:"version"`
}

func responseToDrain(ctx context.Context, out client.Drain) (Drain, diag.Diagnostics) {
	var diags diag.Diagnostics

	projectIds, d := types.SetValueFrom(ctx, types.StringType, out.ProjectIds)
	diags.Append(d...)
	if diags.HasError() {
		return Drain{}, diags
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
		return Drain{}, diags
	}

	deliveryHeaders, d := types.MapValueFrom(ctx, types.StringType, out.Delivery.Headers)
	diags.Append(d...)
	if diags.HasError() {
		return Drain{}, diags
	}

	deliveryModel := DeliveryModel{
		Type:        types.StringValue(out.Delivery.Type),
		Encoding:    types.StringValue(out.Delivery.Encoding),
		Compression: types.StringPointerValue(out.Delivery.Compression),
		Headers:     deliveryHeaders,
		Secret:      types.StringPointerValue(out.Delivery.Secret),
	}

	var endpointModel EndpointModel
	if endpoint, ok := out.Delivery.Endpoint.(string); ok {
		endpointModel = EndpointModel{
			URL:    types.StringValue(endpoint),
			Traces: types.StringNull(),
		}
	} else if otlpEndpoint, ok := out.Delivery.Endpoint.(map[string]any); ok {
		if traces, exists := otlpEndpoint["traces"]; exists {
			if tracesStr, ok := traces.(string); ok {
				endpointModel = EndpointModel{
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
		return Drain{}, diags
	}
	deliveryModel.Endpoint = endpoint

	delivery, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"type":        types.StringType,
		"endpoint":    types.ObjectType{AttrTypes: map[string]attr.Type{"url": types.StringType, "traces": types.StringType}},
		"encoding":    types.StringType,
		"compression": types.StringType,
		"headers":     types.MapType{ElemType: types.StringType},
		"secret":      types.StringType,
	}, deliveryModel)
	diags.Append(d...)
	if diags.HasError() {
		return Drain{}, diags
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
		return Drain{}, diags
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
		return Drain{}, diags
	}

	return Drain{
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

func (r *drainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Drain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest, d := planToCreateRequest(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateDrain(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Drain",
			"Could not create Drain, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToDrain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created Drain", map[string]any{
		"team_id":  plan.TeamID.ValueString(),
		"drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func planToCreateRequest(ctx context.Context, plan Drain) (client.CreateDrainRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	var projectIds []string
	if !plan.ProjectIds.IsNull() && !plan.ProjectIds.IsUnknown() {
		d := plan.ProjectIds.ElementsAs(ctx, &projectIds, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.CreateDrainRequest{}, diags
		}
	}

	schemas := make(map[string]client.SchemaConfig)
	var schemasMap map[string]SchemaVersionModel
	d := plan.Schemas.ElementsAs(ctx, &schemasMap, false)
	diags.Append(d...)
	if diags.HasError() {
		return client.CreateDrainRequest{}, diags
	}

	for k, v := range schemasMap {
		schemas[k] = client.SchemaConfig{
			Version: v.Version.ValueString(),
		}
	}

	var deliveryModel DeliveryModel
	d = plan.Delivery.As(ctx, &deliveryModel, basetypes.ObjectAsOptions{})
	diags.Append(d...)
	if diags.HasError() {
		return client.CreateDrainRequest{}, diags
	}

	var headers map[string]string
	if !deliveryModel.Headers.IsNull() && !deliveryModel.Headers.IsUnknown() {
		d = deliveryModel.Headers.ElementsAs(ctx, &headers, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.CreateDrainRequest{}, diags
		}
	}

	delivery := client.DeliveryConfig{
		Type:     deliveryModel.Type.ValueString(),
		Encoding: deliveryModel.Encoding.ValueString(),
		Headers:  headers,
	}

	if !deliveryModel.Endpoint.IsNull() && !deliveryModel.Endpoint.IsUnknown() {
		var endpointModel EndpointModel
		d = deliveryModel.Endpoint.As(ctx, &endpointModel, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !endpointModel.Traces.IsNull() && !endpointModel.Traces.IsUnknown() {
				delivery.Endpoint = map[string]string{
					"traces": endpointModel.Traces.ValueString(),
				}
			} else if !endpointModel.URL.IsNull() && !endpointModel.URL.IsUnknown() {
				delivery.Endpoint = endpointModel.URL.ValueString()
			}
		}
	}
	if !deliveryModel.Compression.IsNull() && !deliveryModel.Compression.IsUnknown() {
		compression := deliveryModel.Compression.ValueString()
		delivery.Compression = &compression
	}
	if !deliveryModel.Secret.IsNull() && !deliveryModel.Secret.IsUnknown() {
		secret := deliveryModel.Secret.ValueString()
		delivery.Secret = &secret
	}

	var sampling []client.SamplingConfig
	if !plan.Sampling.IsNull() && !plan.Sampling.IsUnknown() {
		var samplingModels []SamplingModel
		d = plan.Sampling.ElementsAs(ctx, &samplingModels, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.CreateDrainRequest{}, diags
		}

		for _, s := range samplingModels {
			samplingConfig := client.SamplingConfig{
				Type: s.Type.ValueString(),
				Rate: s.Rate.ValueFloat64(),
			}
			if !s.Environment.IsNull() && !s.Environment.IsUnknown() {
				env := s.Environment.ValueString()
				samplingConfig.Env = &env
			}
			if !s.RequestPath.IsNull() && !s.RequestPath.IsUnknown() {
				path := s.RequestPath.ValueString()
				samplingConfig.RequestPath = &path
			}
			sampling = append(sampling, samplingConfig)
		}
	}

	var transforms []client.TransformConfig
	if !plan.Transforms.IsNull() && !plan.Transforms.IsUnknown() {
		var transformModels []TransformModel
		d = plan.Transforms.ElementsAs(ctx, &transformModels, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.CreateDrainRequest{}, diags
		}

		for _, t := range transformModels {
			transforms = append(transforms, client.TransformConfig{
				ID: t.ID.ValueString(),
			})
		}
	}

	var filter *string
	if !plan.Filter.IsNull() && !plan.Filter.IsUnknown() {
		f := plan.Filter.ValueString()
		filter = &f
	}

	return client.CreateDrainRequest{
		TeamID:     plan.TeamID.ValueString(),
		Name:       plan.Name.ValueString(),
		Projects:   plan.Projects.ValueString(),
		ProjectIds: projectIds,
		Filter:     filter,
		Schemas:    schemas,
		Delivery:   delivery,
		Sampling:   sampling,
		Transforms: transforms,
	}, diags
}

func (r *drainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Drain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Drain",
			fmt.Sprintf("Could not get Drain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToDrain(ctx, out)
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
}

func (r *drainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state Drain

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest, d := planToUpdateRequest(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateDrain(ctx, state.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Drain",
			"Could not update Drain, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToDrain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updated Drain", map[string]any{
		"team_id":  plan.TeamID.ValueString(),
		"drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func planToUpdateRequest(ctx context.Context, plan Drain) (client.UpdateDrainRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	var projectIds []string
	if !plan.ProjectIds.IsNull() && !plan.ProjectIds.IsUnknown() {
		d := plan.ProjectIds.ElementsAs(ctx, &projectIds, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.UpdateDrainRequest{}, diags
		}
	}

	schemas := make(map[string]client.SchemaConfig)
	var schemasMap map[string]SchemaVersionModel
	d := plan.Schemas.ElementsAs(ctx, &schemasMap, false)
	diags.Append(d...)
	if diags.HasError() {
		return client.UpdateDrainRequest{}, diags
	}

	for k, v := range schemasMap {
		schemas[k] = client.SchemaConfig{
			Version: v.Version.ValueString(),
		}
	}

	var deliveryModel DeliveryModel
	d = plan.Delivery.As(ctx, &deliveryModel, basetypes.ObjectAsOptions{})
	diags.Append(d...)
	if diags.HasError() {
		return client.UpdateDrainRequest{}, diags
	}

	var headers map[string]string
	if !deliveryModel.Headers.IsNull() && !deliveryModel.Headers.IsUnknown() {
		d = deliveryModel.Headers.ElementsAs(ctx, &headers, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.UpdateDrainRequest{}, diags
		}
	}

	delivery := &client.DeliveryConfig{
		Type:     deliveryModel.Type.ValueString(),
		Encoding: deliveryModel.Encoding.ValueString(),
		Headers:  headers,
	}

	if !deliveryModel.Endpoint.IsNull() && !deliveryModel.Endpoint.IsUnknown() {
		var endpointModel EndpointModel
		d = deliveryModel.Endpoint.As(ctx, &endpointModel, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !endpointModel.Traces.IsNull() && !endpointModel.Traces.IsUnknown() {
				delivery.Endpoint = map[string]string{
					"traces": endpointModel.Traces.ValueString(),
				}
			} else if !endpointModel.URL.IsNull() && !endpointModel.URL.IsUnknown() {
				delivery.Endpoint = endpointModel.URL.ValueString()
			}
		}
	}

	if !deliveryModel.Compression.IsNull() && !deliveryModel.Compression.IsUnknown() {
		compression := deliveryModel.Compression.ValueString()
		delivery.Compression = &compression
	}

	if !deliveryModel.Secret.IsNull() && !deliveryModel.Secret.IsUnknown() {
		secret := deliveryModel.Secret.ValueString()
		delivery.Secret = &secret
	}

	var sampling []client.SamplingConfig
	if !plan.Sampling.IsNull() && !plan.Sampling.IsUnknown() {
		var samplingModels []SamplingModel
		d = plan.Sampling.ElementsAs(ctx, &samplingModels, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.UpdateDrainRequest{}, diags
		}

		for _, s := range samplingModels {
			samplingConfig := client.SamplingConfig{
				Type: s.Type.ValueString(),
				Rate: s.Rate.ValueFloat64(),
			}
			if !s.Environment.IsNull() && !s.Environment.IsUnknown() {
				env := s.Environment.ValueString()
				samplingConfig.Env = &env
			}
			if !s.RequestPath.IsNull() && !s.RequestPath.IsUnknown() {
				path := s.RequestPath.ValueString()
				samplingConfig.RequestPath = &path
			}
			sampling = append(sampling, samplingConfig)
		}
	}

	var transforms []client.TransformConfig
	if !plan.Transforms.IsNull() && !plan.Transforms.IsUnknown() {
		var transformModels []TransformModel
		d = plan.Transforms.ElementsAs(ctx, &transformModels, false)
		diags.Append(d...)
		if diags.HasError() {
			return client.UpdateDrainRequest{}, diags
		}

		for _, t := range transformModels {
			transforms = append(transforms, client.TransformConfig{
				ID: t.ID.ValueString(),
			})
		}
	}

	var filter *string
	if !plan.Filter.IsNull() && !plan.Filter.IsUnknown() {
		f := plan.Filter.ValueString()
		filter = &f
	}

	name := plan.Name.ValueString()
	projects := plan.Projects.ValueString()

	return client.UpdateDrainRequest{
		TeamID:     plan.TeamID.ValueString(),
		Name:       &name,
		Projects:   &projects,
		ProjectIds: projectIds,
		Filter:     filter,
		Schemas:    schemas,
		Delivery:   delivery,
		Sampling:   sampling,
		Transforms: transforms,
	}, diags
}

func (r *drainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Drain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting drain",
			fmt.Sprintf(
				"Could not delete Drain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Drain", map[string]any{
		"team_id":  state.TeamID.ValueString(),
		"drain_id": state.ID.ValueString(),
	})
}

func (r *drainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Drain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/drain_id\" or \"drain_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetDrain(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Drain",
			fmt.Sprintf("Could not get Drain %s %s, unexpected error: %s",
				teamID,
				id,
				err,
			),
		)
		return
	}

	result, diags := responseToDrain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "import drain", map[string]any{
		"team_id":  result.TeamID.ValueString(),
		"drain_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
