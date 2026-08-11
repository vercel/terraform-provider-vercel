package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &alertRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &alertRuleDataSource{}
)

func newAlertRuleDataSource() datasource.DataSource {
	return &alertRuleDataSource{}
}

type alertRuleDataSource struct {
	client *client.Client
}

func (d *alertRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (d *alertRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = configuredClient
}

func (d *alertRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Alert Rule.

Alert Rules decide which anomalies and custom Observability thresholds raise an alert for a team or a set of projects, and who is subscribed to the alerts that are raised.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Alert Rule.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the team the Alert Rule exists under. Required when reading a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The human-readable name of the Alert Rule.",
				Computed:    true,
			},
			"project_filter": schema.StringAttribute{
				Description: "The OData expression limiting a built-in anomaly rule to specific projects. Null for team-wide rules and for custom alert rules.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project a custom alert rule monitors. Null for built-in anomaly rules.",
				Computed:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether this is the team's Vercel-managed default Alert Rule.",
				Computed:    true,
			},
			"alert_types": schema.ListNestedAttribute{
				Description: "The classes of alert this rule raises.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The class of alert. Can be `usage_anomaly`, `error_anomaly`, or `custom_alert`.",
							Computed:    true,
						},
						"filter": schema.StringAttribute{
							Description: "The OData expression narrowing this alert type.",
							Computed:    true,
						},
					},
				},
			},
			"sensitivity_level": schema.Int64Attribute{
				Description: "How sensitive built-in anomaly detection is, from `1` (least sensitive) to `5` (most sensitive).",
				Computed:    true,
			},
			"autosubscribe_owners": schema.BoolAttribute{
				Description: "Whether team owners are automatically subscribed to alerts raised by this rule.",
				Computed:    true,
			},
			"autosubscribe_project_admins": schema.BoolAttribute{
				Description: "Whether project administrators are automatically subscribed to alerts raised by this rule.",
				Computed:    true,
			},
			"custom_alert": schema.SingleNestedAttribute{
				Description: "The Observability query and trigger of a custom alert rule.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"event": schema.StringAttribute{
						Description: "The Observability event the query reads.",
						Computed:    true,
					},
					"rollups": schema.MapNestedAttribute{
						Description: "The named measure aggregations the trigger is evaluated against.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"measure": schema.StringAttribute{
									Description: "The measure that is aggregated.",
									Computed:    true,
								},
								"aggregation": schema.StringAttribute{
									Description: "How the measure is aggregated.",
									Computed:    true,
								},
								"filter": schema.StringAttribute{
									Description: "The OData expression limiting this rollup.",
									Computed:    true,
								},
							},
						},
					},
					"group_by": schema.ListAttribute{
						Description: "The dimension the trigger is evaluated separately for.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"filter": schema.StringAttribute{
						Description: "The OData expression limiting the whole query.",
						Computed:    true,
					},
					"granularity": schema.StringAttribute{
						Description: "The window the query is bucketed into.",
						Computed:    true,
					},
					"trigger_type": schema.StringAttribute{
						Description: "How `trigger_threshold` is interpreted. Can be `threshold` or `anomaly`.",
						Computed:    true,
					},
					"trigger_operator": schema.StringAttribute{
						Description: "How the query result is compared against `trigger_threshold`.",
						Computed:    true,
					},
					"trigger_threshold": schema.Float64Attribute{
						Description: "The value the query result is compared against.",
						Computed:    true,
					},
					"min_threshold": schema.Float64Attribute{
						Description: "The smallest observed value that can raise an alert.",
						Computed:    true,
					},
					"formula": schema.SingleNestedAttribute{
						Description: "Combines two rollups into a single value before the trigger is evaluated.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"operator": schema.StringAttribute{
								Description: "How the rollups are combined.",
								Computed:    true,
							},
							"left": schema.StringAttribute{
								Description: "The name of the rollup used as the left-hand side of the formula.",
								Computed:    true,
							},
							"right": schema.StringAttribute{
								Description: "The name of the rollup used as the right-hand side of the formula.",
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

// AlertRuleDataSourceModel mirrors AlertRule, with the additional is_default
// attribute that is only meaningful when reading an existing rule.
type AlertRuleDataSourceModel struct {
	ID                         types.String          `tfsdk:"id"`
	TeamID                     types.String          `tfsdk:"team_id"`
	Name                       types.String          `tfsdk:"name"`
	ProjectFilter              types.String          `tfsdk:"project_filter"`
	ProjectID                  types.String          `tfsdk:"project_id"`
	IsDefault                  types.Bool            `tfsdk:"is_default"`
	AlertTypes                 types.List            `tfsdk:"alert_types"`
	SensitivityLevel           types.Int64           `tfsdk:"sensitivity_level"`
	AutosubscribeOwners        types.Bool            `tfsdk:"autosubscribe_owners"`
	AutosubscribeProjectAdmins types.Bool            `tfsdk:"autosubscribe_project_admins"`
	CustomAlert                *AlertRuleCustomAlert `tfsdk:"custom_alert"`
}

func (d *alertRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AlertRuleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetAlertRule(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Alert Rule",
			fmt.Sprintf("Could not get Alert Rule %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	alertRule, diags := alertRuleFromAPI(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result := AlertRuleDataSourceModel{
		ID:                         alertRule.ID,
		TeamID:                     alertRule.TeamID,
		Name:                       alertRule.Name,
		ProjectFilter:              alertRule.ProjectFilter,
		ProjectID:                  alertRule.ProjectID,
		IsDefault:                  types.BoolValue(out.IsDefault),
		AlertTypes:                 alertRule.AlertTypes,
		SensitivityLevel:           alertRule.SensitivityLevel,
		AutosubscribeOwners:        alertRule.AutosubscribeOwners,
		AutosubscribeProjectAdmins: alertRule.AutosubscribeProjectAdmins,
		CustomAlert:                alertRule.CustomAlert,
	}

	tflog.Info(ctx, "read Alert Rule", map[string]any{
		"team_id":       result.TeamID.ValueString(),
		"alert_rule_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
