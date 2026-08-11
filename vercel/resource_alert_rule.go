package vercel

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                     = &alertRuleResource{}
	_ resource.ResourceWithConfigure        = &alertRuleResource{}
	_ resource.ResourceWithConfigValidators = &alertRuleResource{}
	_ resource.ResourceWithValidateConfig   = &alertRuleResource{}
	_ resource.ResourceWithImportState      = &alertRuleResource{}
)

func newAlertRuleResource() resource.Resource {
	return &alertRuleResource{}
}

type alertRuleResource struct {
	client *client.Client
}

func (r *alertRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *alertRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = configuredClient
}

func (r *alertRuleResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("project_filter"),
			path.MatchRoot("project_id"),
		),
	}
}

// alertRuleCustomAlertRequiresReplace forces replacement when a rule is switched
// between a built-in anomaly rule and a custom alert rule. The Vercel API cannot
// remove a custom alert definition from an existing rule.
func alertRuleCustomAlertRequiresReplace() planmodifier.Object {
	return objectplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
				// Creating or destroying the resource.
				return
			}
			resp.RequiresReplace = req.StateValue.IsNull() != req.ConfigValue.IsNull()
		},
		"Adding or removing `custom_alert` requires the alert rule to be replaced.",
		"Adding or removing `custom_alert` requires the alert rule to be replaced.",
	)
}

func (r *alertRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Alert Rule resource.

Alert Rules decide which anomalies and custom Observability thresholds raise an alert for a team or a set of projects, and who is subscribed to the alerts that are raised.

There are two kinds of alert rule:

* Built-in anomaly rules, configured with the ` + "`usage_anomaly`" + ` and ` + "`error_anomaly`" + ` alert types. These are scoped with the ` + "`project_filter`" + ` OData expression, or left team-wide.
* Custom alert rules, configured with the ` + "`custom_alert`" + ` alert type and a ` + "`custom_alert`" + ` block. These evaluate an Observability query against a threshold or an anomaly z-score, and target a single project via ` + "`project_id`" + `.

Delivery to Slack channels is subscribed per rule outside of Terraform, using the rule ` + "`id`" + `. See the [Alerts documentation](https://vercel.com/docs/alerts/configure-alerts) for more detail.

~> Alert Rules require Observability Plus. Custom alert rules additionally consume your team's custom alert allowance.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Alert Rule.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team the Alert Rule should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "A human-readable name for the Alert Rule.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"project_filter": schema.StringAttribute{
				Description: "An OData expression limiting a built-in anomaly rule to specific projects, such as `projectId in ('prj_123', 'prj_456')`. Omit this to monitor every project in the team. Conflicts with `project_id`, and cannot be used by custom alert rules.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project a custom alert rule monitors. Required when `custom_alert` is set, and unavailable to built-in anomaly rules, which use `project_filter` instead.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"alert_types": schema.ListNestedAttribute{
				Description: "The classes of alert this rule raises.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The class of alert. Can be `usage_anomaly`, `error_anomaly`, or `custom_alert`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									client.AlertRuleTypeUsageAnomaly,
									client.AlertRuleTypeErrorAnomaly,
									client.AlertRuleTypeCustomAlert,
								),
							},
						},
						"filter": schema.StringAttribute{
							Description: "An OData expression narrowing this alert type. `usage_anomaly` supports `metric`, for example `metric eq 'edge_requests'`. `error_anomaly` supports `statusGroup` and `route`, for example `statusGroup eq '5xx' and route eq '/api/checkout'`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
			},
			"sensitivity_level": schema.Int64Attribute{
				Description: "How sensitive built-in anomaly detection should be, from `1` (least sensitive) to `5` (most sensitive). Omit this to use Vercel's default sensitivity.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 5),
				},
			},
			"autosubscribe_owners": schema.BoolAttribute{
				Description: "Whether team owners are automatically subscribed to alerts raised by this rule.",
				Optional:    true,
			},
			"autosubscribe_project_admins": schema.BoolAttribute{
				Description: "Whether project administrators are automatically subscribed to alerts raised by this rule.",
				Optional:    true,
			},
			"custom_alert": schema.SingleNestedAttribute{
				Description:   "An Observability query and trigger for a custom alert rule. Requires `custom_alert` in `alert_types`, and requires `project_id`.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{alertRuleCustomAlertRequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"event": schema.StringAttribute{
						Description: "The Observability event to query, such as `incomingRequest`, `serverlessFunctionInvocation`, `outgoingRequest`, or `sandboxUsage`. Run `vercel metrics schema <metric-or-prefix>` to discover the event and measure names behind a public metric ID.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"rollups": schema.MapNestedAttribute{
						Description: "The named measure aggregations the trigger is evaluated against. A `formula` references these names.",
						Required:    true,
						Validators: []validator.Map{
							mapvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"measure": schema.StringAttribute{
									Description: "The measure to aggregate, such as `count`.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
								"aggregation": schema.StringAttribute{
									Description: "How the measure is aggregated, such as `sum`.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
								"filter": schema.StringAttribute{
									Description: "An OData expression limiting this rollup, such as `httpStatus ge 500`.",
									Optional:    true,
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
							},
						},
					},
					"group_by": schema.ListAttribute{
						Description: "A dimension to evaluate the trigger separately for, such as `route`. At most one dimension is supported.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.SizeAtMost(1),
						},
					},
					"filter": schema.StringAttribute{
						Description: "An OData expression limiting the whole query.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"granularity": schema.StringAttribute{
						Description: "The window the query is bucketed into. Can be `5m`, `1h`, or `1d`. Defaults to `5m`.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("5m", "1h", "1d"),
						},
					},
					"trigger_type": schema.StringAttribute{
						Description: "How `trigger_threshold` is interpreted. `threshold` compares the query result against the value directly. `anomaly` compares the query result's z-score against it.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("threshold", "anomaly"),
						},
					},
					"trigger_operator": schema.StringAttribute{
						Description: "How the query result is compared against `trigger_threshold`. Can be `gt`, `gte`, `lt`, or `lte`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("gt", "gte", "lt", "lte"),
						},
					},
					"trigger_threshold": schema.Float64Attribute{
						Description: "The value the query result is compared against. For a `threshold` trigger this is the metric value, and for an `anomaly` trigger it is a z-score.",
						Required:    true,
					},
					"min_threshold": schema.Float64Attribute{
						Description: "The smallest observed value that can raise an alert, used to suppress alerts on low traffic. Only supported by `anomaly` triggers and by triggers using a `formula`.",
						Optional:    true,
						Validators: []validator.Float64{
							float64validator.AtLeast(0),
						},
					},
					"formula": schema.SingleNestedAttribute{
						Description: "Combines two rollups into a single value before the trigger is evaluated, such as an error rate.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"operator": schema.StringAttribute{
								Description: "How the rollups are combined. Only `divide` is supported.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("divide"),
								},
							},
							"left": schema.StringAttribute{
								Description: "The name of the rollup used as the left-hand side of the formula.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.LengthAtLeast(1),
								},
							},
							"right": schema.StringAttribute{
								Description: "The name of the rollup used as the right-hand side of the formula.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.LengthAtLeast(1),
								},
							},
						},
					},
				},
			},
		},
	}
}

// AlertRule is the Terraform model for a vercel_alert_rule resource.
type AlertRule struct {
	ID                         types.String          `tfsdk:"id"`
	TeamID                     types.String          `tfsdk:"team_id"`
	Name                       types.String          `tfsdk:"name"`
	ProjectFilter              types.String          `tfsdk:"project_filter"`
	ProjectID                  types.String          `tfsdk:"project_id"`
	AlertTypes                 types.List            `tfsdk:"alert_types"`
	SensitivityLevel           types.Int64           `tfsdk:"sensitivity_level"`
	AutosubscribeOwners        types.Bool            `tfsdk:"autosubscribe_owners"`
	AutosubscribeProjectAdmins types.Bool            `tfsdk:"autosubscribe_project_admins"`
	CustomAlert                *AlertRuleCustomAlert `tfsdk:"custom_alert"`
}

// AlertRuleAlertType is the Terraform model for a single entry of alert_types.
type AlertRuleAlertType struct {
	Type   types.String `tfsdk:"type"`
	Filter types.String `tfsdk:"filter"`
}

// AlertRuleCustomAlert is the Terraform model for the custom_alert block.
type AlertRuleCustomAlert struct {
	Event            types.String                 `tfsdk:"event"`
	Rollups          types.Map                    `tfsdk:"rollups"`
	GroupBy          types.List                   `tfsdk:"group_by"`
	Filter           types.String                 `tfsdk:"filter"`
	Granularity      types.String                 `tfsdk:"granularity"`
	TriggerType      types.String                 `tfsdk:"trigger_type"`
	TriggerOperator  types.String                 `tfsdk:"trigger_operator"`
	TriggerThreshold types.Float64                `tfsdk:"trigger_threshold"`
	MinThreshold     types.Float64                `tfsdk:"min_threshold"`
	Formula          *AlertRuleCustomAlertFormula `tfsdk:"formula"`
}

// AlertRuleCustomAlertRollup is the Terraform model for a single custom_alert rollup.
type AlertRuleCustomAlertRollup struct {
	Measure     types.String `tfsdk:"measure"`
	Aggregation types.String `tfsdk:"aggregation"`
	Filter      types.String `tfsdk:"filter"`
}

// AlertRuleCustomAlertFormula is the Terraform model for the custom_alert formula.
type AlertRuleCustomAlertFormula struct {
	Operator types.String `tfsdk:"operator"`
	Left     types.String `tfsdk:"left"`
	Right    types.String `tfsdk:"right"`
}

var alertRuleAlertTypeAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":   types.StringType,
		"filter": types.StringType,
	},
}

var alertRuleRollupAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"measure":     types.StringType,
		"aggregation": types.StringType,
		"filter":      types.StringType,
	},
}

// alertRuleGranularityToClient converts a granularity such as `1h` into the
// object shape the Observability query uses.
func alertRuleGranularityToClient(value types.String) (*client.CustomAlertGranularity, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil, nil
	}

	raw := value.ValueString()
	amount, err := strconv.ParseInt(raw[:len(raw)-1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid granularity %q: must be a number followed by `m`, `h`, or `d`", raw)
	}

	switch raw[len(raw)-1] {
	case 'm':
		return &client.CustomAlertGranularity{Minutes: &amount}, nil
	case 'h':
		return &client.CustomAlertGranularity{Hours: &amount}, nil
	case 'd':
		return &client.CustomAlertGranularity{Days: &amount}, nil
	default:
		return nil, fmt.Errorf("invalid granularity %q: must be a number followed by `m`, `h`, or `d`", raw)
	}
}

func alertRuleGranularityFromClient(granularity *client.CustomAlertGranularity) types.String {
	if granularity == nil {
		return types.StringNull()
	}
	switch {
	case granularity.Minutes != nil:
		return types.StringValue(fmt.Sprintf("%dm", *granularity.Minutes))
	case granularity.Hours != nil:
		return types.StringValue(fmt.Sprintf("%dh", *granularity.Hours))
	case granularity.Days != nil:
		return types.StringValue(fmt.Sprintf("%dd", *granularity.Days))
	default:
		return types.StringNull()
	}
}

func alertRuleAlertTypesToClient(ctx context.Context, list types.List) ([]client.AlertRuleType, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var models []AlertRuleAlertType
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	alertTypes := make([]client.AlertRuleType, 0, len(models))
	for _, model := range models {
		alertTypes = append(alertTypes, client.AlertRuleType{
			Type:   model.Type.ValueString(),
			Filter: optionalString(model.Filter),
		})
	}
	return alertTypes, nil
}

func alertRuleAlertTypesFromClient(ctx context.Context, alertTypes []client.AlertRuleType) (types.List, diag.Diagnostics) {
	models := make([]AlertRuleAlertType, 0, len(alertTypes))
	for _, alertType := range alertTypes {
		filter := types.StringNull()
		if alertType.Filter != nil {
			filter = types.StringValue(*alertType.Filter)
		}
		models = append(models, AlertRuleAlertType{
			Type:   types.StringValue(alertType.Type),
			Filter: filter,
		})
	}
	return types.ListValueFrom(ctx, alertRuleAlertTypeAttrType, models)
}

func alertRuleCustomAlertToClient(ctx context.Context, customAlert *AlertRuleCustomAlert) (*client.CustomAlert, diag.Diagnostics) {
	var diags diag.Diagnostics
	if customAlert == nil {
		return nil, diags
	}

	var rollupModels map[string]AlertRuleCustomAlertRollup
	if !customAlert.Rollups.IsNull() && !customAlert.Rollups.IsUnknown() {
		diags.Append(customAlert.Rollups.ElementsAs(ctx, &rollupModels, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	rollups := make(map[string]client.CustomAlertRollup, len(rollupModels))
	for name, rollup := range rollupModels {
		rollups[name] = client.CustomAlertRollup{
			Measure:     rollup.Measure.ValueString(),
			Aggregation: rollup.Aggregation.ValueString(),
			Filter:      optionalString(rollup.Filter),
		}
	}

	var groupBy []string
	if !customAlert.GroupBy.IsNull() && !customAlert.GroupBy.IsUnknown() {
		diags.Append(customAlert.GroupBy.ElementsAs(ctx, &groupBy, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	granularity, err := alertRuleGranularityToClient(customAlert.Granularity)
	if err != nil {
		diags.AddError("Invalid Alert Rule custom alert granularity", err.Error())
		return nil, diags
	}

	var formula *client.CustomAlertFormula
	if customAlert.Formula != nil {
		formula = &client.CustomAlertFormula{
			Operator: customAlert.Formula.Operator.ValueString(),
			Left:     customAlert.Formula.Left.ValueString(),
			Right:    customAlert.Formula.Right.ValueString(),
		}
	}

	var minThreshold *float64
	if !customAlert.MinThreshold.IsNull() && !customAlert.MinThreshold.IsUnknown() {
		value := customAlert.MinThreshold.ValueFloat64()
		minThreshold = &value
	}

	return &client.CustomAlert{
		Query: client.CustomAlertQuery{
			Event:       customAlert.Event.ValueString(),
			Rollups:     rollups,
			GroupBy:     groupBy,
			Filter:      optionalString(customAlert.Filter),
			Granularity: granularity,
		},
		TriggerType:      customAlert.TriggerType.ValueString(),
		TriggerOperator:  customAlert.TriggerOperator.ValueString(),
		TriggerThreshold: customAlert.TriggerThreshold.ValueFloat64(),
		MinThreshold:     minThreshold,
		Formula:          formula,
	}, diags
}

func alertRuleCustomAlertFromClient(ctx context.Context, customAlert *client.CustomAlert) (*AlertRuleCustomAlert, diag.Diagnostics) {
	var diags diag.Diagnostics
	if customAlert == nil {
		return nil, diags
	}

	rollupModels := make(map[string]AlertRuleCustomAlertRollup, len(customAlert.Query.Rollups))
	for name, rollup := range customAlert.Query.Rollups {
		filter := types.StringNull()
		if rollup.Filter != nil {
			filter = types.StringValue(*rollup.Filter)
		}
		rollupModels[name] = AlertRuleCustomAlertRollup{
			Measure:     types.StringValue(rollup.Measure),
			Aggregation: types.StringValue(rollup.Aggregation),
			Filter:      filter,
		}
	}
	rollups, rollupDiags := types.MapValueFrom(ctx, alertRuleRollupAttrType, rollupModels)
	diags.Append(rollupDiags...)
	if diags.HasError() {
		return nil, diags
	}

	groupBy := types.ListNull(types.StringType)
	if len(customAlert.Query.GroupBy) > 0 {
		converted, groupByDiags := types.ListValueFrom(ctx, types.StringType, customAlert.Query.GroupBy)
		diags.Append(groupByDiags...)
		if diags.HasError() {
			return nil, diags
		}
		groupBy = converted
	}

	filter := types.StringNull()
	if customAlert.Query.Filter != nil {
		filter = types.StringValue(*customAlert.Query.Filter)
	}

	minThreshold := types.Float64Null()
	if customAlert.MinThreshold != nil {
		minThreshold = types.Float64Value(*customAlert.MinThreshold)
	}

	var formula *AlertRuleCustomAlertFormula
	if customAlert.Formula != nil {
		formula = &AlertRuleCustomAlertFormula{
			Operator: types.StringValue(customAlert.Formula.Operator),
			Left:     types.StringValue(customAlert.Formula.Left),
			Right:    types.StringValue(customAlert.Formula.Right),
		}
	}

	return &AlertRuleCustomAlert{
		Event:            types.StringValue(customAlert.Query.Event),
		Rollups:          rollups,
		GroupBy:          groupBy,
		Filter:           filter,
		Granularity:      alertRuleGranularityFromClient(customAlert.Query.Granularity),
		TriggerType:      types.StringValue(customAlert.TriggerType),
		TriggerOperator:  types.StringValue(customAlert.TriggerOperator),
		TriggerThreshold: types.Float64Value(customAlert.TriggerThreshold),
		MinThreshold:     minThreshold,
		Formula:          formula,
	}, diags
}

// alertRuleFromAPI converts an API response into the Terraform model. The API
// stores a single projectId field that holds an OData project filter for
// built-in anomaly rules and a raw project ID for custom alert rules, so the
// value is mapped onto whichever attribute matches the kind of rule.
func alertRuleFromAPI(ctx context.Context, out client.AlertRule) (AlertRule, diag.Diagnostics) {
	alertTypes, diags := alertRuleAlertTypesFromClient(ctx, out.AlertTypes)
	if diags.HasError() {
		return AlertRule{}, diags
	}

	customAlert, customAlertDiags := alertRuleCustomAlertFromClient(ctx, out.CustomAlert)
	diags.Append(customAlertDiags...)
	if diags.HasError() {
		return AlertRule{}, diags
	}

	projectFilter := types.StringNull()
	projectID := types.StringNull()
	if out.ProjectID != nil && *out.ProjectID != "" {
		if out.IsCustomAlert() {
			projectID = types.StringValue(*out.ProjectID)
		} else {
			projectFilter = types.StringValue(*out.ProjectID)
		}
	}

	sensitivityLevel := types.Int64Null()
	if out.SensitivityLevel != nil {
		sensitivityLevel = types.Int64Value(*out.SensitivityLevel)
	}

	autosubscribeOwners := types.BoolNull()
	if out.AutosubscribeOwners != nil {
		autosubscribeOwners = types.BoolValue(*out.AutosubscribeOwners)
	}

	autosubscribeProjectAdmins := types.BoolNull()
	if out.AutosubscribeProjectAdmins != nil {
		autosubscribeProjectAdmins = types.BoolValue(*out.AutosubscribeProjectAdmins)
	}

	return AlertRule{
		ID:                         types.StringValue(out.ID),
		TeamID:                     toTeamID(out.TeamID),
		Name:                       types.StringValue(out.Name),
		ProjectFilter:              projectFilter,
		ProjectID:                  projectID,
		AlertTypes:                 alertTypes,
		SensitivityLevel:           sensitivityLevel,
		AutosubscribeOwners:        autosubscribeOwners,
		AutosubscribeProjectAdmins: autosubscribeProjectAdmins,
		CustomAlert:                customAlert,
	}, diags
}

// projectTarget returns the value for the API's projectId field, which is an
// OData filter for built-in rules and a raw project ID for custom alert rules.
func (a AlertRule) projectTarget() *string {
	if target := optionalString(a.ProjectID); target != nil {
		return target
	}
	return optionalString(a.ProjectFilter)
}

func alertRuleHasCustomAlertType(ctx context.Context, list types.List) (bool, diag.Diagnostics) {
	alertTypes, diags := alertRuleAlertTypesToClient(ctx, list)
	if diags.HasError() {
		return false, diags
	}
	for _, alertType := range alertTypes {
		if alertType.Type == client.AlertRuleTypeCustomAlert {
			return true, diags
		}
	}
	return false, diags
}

// ValidateConfig checks the combinations of attributes that depend on the kind of
// alert rule being configured.
func (r *alertRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var alertTypesList types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("alert_types"), &alertTypesList)...)
	var customAlertObject types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("custom_alert"), &customAlertObject)...)
	var projectFilter types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_filter"), &projectFilter)...)
	var projectID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasCustomAlertType, diags := alertRuleHasCustomAlertType(ctx, alertTypesList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasCustomAlertBlock := !customAlertObject.IsNull() && !customAlertObject.IsUnknown()
	alertTypesKnown := !alertTypesList.IsNull() && !alertTypesList.IsUnknown()

	if hasCustomAlertBlock && alertTypesKnown && !hasCustomAlertType {
		resp.Diagnostics.AddAttributeError(
			path.Root("alert_types"),
			"Alert Rule Invalid",
			"The `custom_alert` attribute requires an `alert_types` entry with a `type` of `custom_alert`.",
		)
	}

	if hasCustomAlertType && customAlertObject.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("custom_alert"),
			"Alert Rule Invalid",
			"An `alert_types` entry with a `type` of `custom_alert` requires the `custom_alert` attribute to be set.",
		)
	}

	if hasCustomAlertType && !projectFilter.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_filter"),
			"Alert Rule Invalid",
			"Custom alert rules target a single project. Use `project_id` instead of `project_filter`.",
		)
	}

	if hasCustomAlertType && projectID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"),
			"Alert Rule Invalid",
			"Custom alert rules must set `project_id` to the project they monitor.",
		)
	}

	if alertTypesKnown && !hasCustomAlertType && !projectID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"),
			"Alert Rule Invalid",
			"Built-in anomaly rules are scoped with the `project_filter` OData expression rather than `project_id`.",
		)
	}

	if !hasCustomAlertBlock {
		return
	}

	var customAlert *AlertRuleCustomAlert
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("custom_alert"), &customAlert)...)
	if resp.Diagnostics.HasError() || customAlert == nil {
		return
	}

	if !customAlert.MinThreshold.IsNull() &&
		customAlert.TriggerType.ValueString() == "threshold" &&
		customAlert.Formula == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("custom_alert").AtName("min_threshold"),
			"Alert Rule Invalid",
			"`min_threshold` is only supported by `anomaly` triggers and by triggers using a `formula`.",
		)
	}

	if customAlert.Formula == nil || customAlert.Rollups.IsNull() || customAlert.Rollups.IsUnknown() {
		return
	}

	var rollups map[string]AlertRuleCustomAlertRollup
	resp.Diagnostics.Append(customAlert.Rollups.ElementsAs(ctx, &rollups, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, side := range map[string]types.String{
		"left":  customAlert.Formula.Left,
		"right": customAlert.Formula.Right,
	} {
		if side.IsNull() || side.IsUnknown() {
			continue
		}
		if _, ok := rollups[side.ValueString()]; !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("custom_alert").AtName("formula").AtName(name),
				"Alert Rule Invalid",
				fmt.Sprintf(
					"`formula.%s` must reference a key of `rollups`. %q is not one of: %s.",
					name,
					side.ValueString(),
					strings.Join(slices.Sorted(maps.Keys(rollups)), ", "),
				),
			)
		}
	}
}

func (r *alertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alertTypes, diags := alertRuleAlertTypesToClient(ctx, plan.AlertTypes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	customAlert, diags := alertRuleCustomAlertToClient(ctx, plan.CustomAlert)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateAlertRule(ctx, client.CreateAlertRuleRequest{
		TeamID:                     plan.TeamID.ValueString(),
		Name:                       plan.Name.ValueString(),
		ProjectID:                  plan.projectTarget(),
		AlertTypes:                 alertTypes,
		SensitivityLevel:           optionalInt64(plan.SensitivityLevel),
		AutosubscribeOwners:        optionalBool(plan.AutosubscribeOwners),
		AutosubscribeProjectAdmins: optionalBool(plan.AutosubscribeProjectAdmins),
		CustomAlert:                customAlert,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Alert Rule",
			"Could not create Alert Rule, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := alertRuleFromAPI(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created Alert Rule", map[string]any{
		"team_id":       result.TeamID.ValueString(),
		"alert_rule_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *alertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAlertRule(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Alert Rule",
			fmt.Sprintf("Could not get Alert Rule %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := alertRuleFromAPI(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "read Alert Rule", map[string]any{
		"team_id":       result.TeamID.ValueString(),
		"alert_rule_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alertTypes, diags := alertRuleAlertTypesToClient(ctx, plan.AlertTypes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	customAlert, diags := alertRuleCustomAlertToClient(ctx, plan.CustomAlert)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateAlertRule(ctx, client.UpdateAlertRuleRequest{
		TeamID:                     plan.TeamID.ValueString(),
		ID:                         state.ID.ValueString(),
		Name:                       plan.Name.ValueString(),
		ProjectID:                  plan.projectTarget(),
		AlertTypes:                 alertTypes,
		SensitivityLevel:           optionalInt64(plan.SensitivityLevel),
		AutosubscribeOwners:        optionalBool(plan.AutosubscribeOwners),
		AutosubscribeProjectAdmins: optionalBool(plan.AutosubscribeProjectAdmins),
		CustomAlert:                customAlert,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Alert Rule",
			fmt.Sprintf("Could not update Alert Rule %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := alertRuleFromAPI(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updated Alert Rule", map[string]any{
		"team_id":       result.TeamID.ValueString(),
		"alert_rule_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *alertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAlertRule(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Alert Rule",
			fmt.Sprintf("Could not delete Alert Rule %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Alert Rule", map[string]any{
		"team_id":       state.TeamID.ValueString(),
		"alert_rule_id": state.ID.ValueString(),
	})
}

func (r *alertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Alert Rule",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/alert_rule_id\" or \"alert_rule_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetAlertRule(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Alert Rule",
			fmt.Sprintf("Could not get Alert Rule %s %s, unexpected error: %s", teamID, id, err),
		)
		return
	}

	result, diags := alertRuleFromAPI(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "import Alert Rule", map[string]any{
		"team_id":       result.TeamID.ValueString(),
		"alert_rule_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
