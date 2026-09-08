package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource                   = &alertRuleResource{}
	_ resource.ResourceWithConfigure      = &alertRuleResource{}
	_ resource.ResourceWithImportState    = &alertRuleResource{}
	_ resource.ResourceWithValidateConfig = &alertRuleResource{}
)

func newAlertRuleResource() resource.Resource {
	return &alertRuleResource{}
}

type alertRuleResource struct {
	client *client.Client
}

func (r *alertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *alertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *alertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a Vercel alert rule using the Alerts v3 API. Built-in rules select anomaly triggers across a team scope; custom rules evaluate a metric query for one project. Configure shared custom-query filters on each metric because the API expands its query-level filter shorthand during reads. Notification channel links are managed separately from this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the alert rule.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the team that owns the alert rule. Required if a default team is not configured in the provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseNonNullStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The alert rule type. Either `built-in` or `custom`.",
				Validators: []validator.String{
					stringvalidator.OneOf(client.AlertRuleTypeBuiltIn, client.AlertRuleTypeCustom),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A human-readable name for the alert rule.",
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"rule_scope": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The projects affected by the rule. Built-in rules use `all`, `include`, or `exclude`; custom rules use `project`.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("all", "include", "exclude", "project"),
						},
					},
					"project_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The single project ID for a custom rule.",
						Validators:          []validator.String{stringvalidator.LengthBetween(1, 256)},
					},
					"project_ids": schema.SetAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "The project IDs included in or excluded from a built-in rule.",
						Validators: []validator.Set{
							setvalidator.SizeBetween(1, 100),
							setvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 256)),
						},
					},
				},
			},
			"triggers": schema.SetNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The built-in anomaly triggers enabled for a built-in rule.",
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf(client.AlertRuleBuiltInTriggerTypes...)},
						},
						"filter": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A KQL filter. It is required for `error_anomaly`, optional for `usage_anomaly`, and unavailable for other trigger types.",
							Validators:          []validator.String{stringvalidator.LengthBetween(1, 2048)},
						},
					},
				},
			},
			"match_minimum_severity_level": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The minimum severity matched by a built-in rule.",
				Validators: []validator.String{
					stringvalidator.OneOf("low", "medium", "high", "critical"),
				},
			},
			"severity": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The severity assigned by a custom rule.",
				Validators:          []validator.String{stringvalidator.OneOf("low", "medium", "high")},
			},
			"evaluation": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The metric query evaluated by a custom rule.",
				Attributes: map[string]schema.Attribute{
					"window": schema.StringAttribute{
						Required:   true,
						Validators: []validator.String{stringvalidator.OneOf("5m", "1h", "1d")},
					},
					"query": schema.SingleNestedAttribute{
						Required: true,
						Attributes: map[string]schema.Attribute{
							"group_by": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
								Validators:  []validator.List{listvalidator.SizeBetween(1, 1)},
							},
							"metrics": schema.MapNestedAttribute{
								Required:            true,
								MarkdownDescription: "Metric selections keyed by caller-chosen aliases. Use one metric without a formula, or two metrics with a division formula.",
								Validators: []validator.Map{
									mapvalidator.SizeAtLeast(1),
									mapvalidator.SizeAtMost(2),
								},
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"metric": schema.StringAttribute{Required: true},
										"aggregation": schema.StringAttribute{
											Required: true,
											Validators: []validator.String{stringvalidator.OneOf(
												"count", "sum", "avg", "min", "max", "p50", "p75", "p90", "p95", "p99", "stddev", "unique",
											)},
										},
										"per": schema.StringAttribute{
											Optional:   true,
											Validators: []validator.String{stringvalidator.OneOf("second")},
										},
										"normalize": schema.StringAttribute{
											Optional:   true,
											Validators: []validator.String{stringvalidator.OneOf("percent")},
										},
										"dimensions": schema.SetAttribute{
											Optional:    true,
											ElementType: types.StringType,
										},
										"filter": schema.StringAttribute{
											Optional:   true,
											Validators: []validator.String{stringvalidator.LengthBetween(1, 2048)},
										},
									},
								},
							},
							"formulas": schema.MapAttribute{
								Optional:            true,
								ElementType:         types.StringType,
								MarkdownDescription: "A single division formula under the `formula` key, for example `errors / requests`.",
								Validators:          []validator.Map{mapvalidator.SizeBetween(1, 1)},
							},
							"outputs": schema.ListAttribute{
								Required:    true,
								ElementType: types.StringType,
								Validators:  []validator.List{listvalidator.SizeBetween(1, 1)},
							},
						},
					},
				},
			},
			"trigger": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The condition that causes a custom rule to fire.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:   true,
						Validators: []validator.String{stringvalidator.OneOf("threshold", "anomaly")},
					},
					"output": schema.StringAttribute{Required: true},
					"operator": schema.StringAttribute{
						Optional:   true,
						Validators: []validator.String{stringvalidator.OneOf("gt", "gte", "lt", "lte")},
					},
					"threshold": schema.Float64Attribute{Optional: true},
					"standard_deviations": schema.Float64Attribute{
						Optional:   true,
						Validators: []validator.Float64{float64validator.AtLeast(0.1)},
					},
					"minimum": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"output":    schema.StringAttribute{Required: true},
							"threshold": schema.Float64Attribute{Required: true, Validators: []validator.Float64{float64validator.AtLeast(0)}},
						},
					},
				},
			},
			"notification_settings": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Notification delivery settings stored on the rule. Notification channel links are managed separately.",
				Attributes: map[string]schema.Attribute{
					"enable_team_owner_notifications": schema.BoolAttribute{
						Optional: true,
						Computed: true,
					},
					"incident_io_routing_key": schema.StringAttribute{
						Optional:   true,
						Sensitive:  true,
						Validators: []validator.String{stringvalidator.LengthBetween(1, 256)},
					},
				},
			},
			"is_default": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the immutable team default rule. Default rules cannot be managed by this resource.",
			},
			"query_supported": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether a custom rule's stored query can be represented by the v3 API.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Creation time as a Unix epoch timestamp in milliseconds.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Last update time as a Unix epoch timestamp in milliseconds.",
			},
		},
	}
}

type AlertRule struct {
	ID                        types.String                   `tfsdk:"id"`
	TeamID                    types.String                   `tfsdk:"team_id"`
	Type                      types.String                   `tfsdk:"type"`
	Name                      types.String                   `tfsdk:"name"`
	RuleScope                 *AlertRuleScope                `tfsdk:"rule_scope"`
	Triggers                  types.Set                      `tfsdk:"triggers"`
	MatchMinimumSeverityLevel types.String                   `tfsdk:"match_minimum_severity_level"`
	Severity                  types.String                   `tfsdk:"severity"`
	Evaluation                *AlertRuleEvaluation           `tfsdk:"evaluation"`
	Trigger                   *AlertRuleCustomTrigger        `tfsdk:"trigger"`
	NotificationSettings      *AlertRuleNotificationSettings `tfsdk:"notification_settings"`
	IsDefault                 types.Bool                     `tfsdk:"is_default"`
	QuerySupported            types.Bool                     `tfsdk:"query_supported"`
	CreatedAt                 types.Int64                    `tfsdk:"created_at"`
	UpdatedAt                 types.Int64                    `tfsdk:"updated_at"`
}

type AlertRuleScope struct {
	Type       types.String `tfsdk:"type"`
	ProjectID  types.String `tfsdk:"project_id"`
	ProjectIDs types.Set    `tfsdk:"project_ids"`
}

type AlertRuleTrigger struct {
	Type   types.String `tfsdk:"type"`
	Filter types.String `tfsdk:"filter"`
}

type AlertRuleEvaluation struct {
	Window types.String          `tfsdk:"window"`
	Query  *AlertRuleCustomQuery `tfsdk:"query"`
}

type AlertRuleCustomQuery struct {
	GroupBy  types.List `tfsdk:"group_by"`
	Metrics  types.Map  `tfsdk:"metrics"`
	Formulas types.Map  `tfsdk:"formulas"`
	Outputs  types.List `tfsdk:"outputs"`
}

type AlertRuleMetricSelection struct {
	Metric      types.String `tfsdk:"metric"`
	Aggregation types.String `tfsdk:"aggregation"`
	Per         types.String `tfsdk:"per"`
	Normalize   types.String `tfsdk:"normalize"`
	Dimensions  types.Set    `tfsdk:"dimensions"`
	Filter      types.String `tfsdk:"filter"`
}

type AlertRuleCustomTrigger struct {
	Type               types.String             `tfsdk:"type"`
	Output             types.String             `tfsdk:"output"`
	Operator           types.String             `tfsdk:"operator"`
	Threshold          types.Float64            `tfsdk:"threshold"`
	StandardDeviations types.Float64            `tfsdk:"standard_deviations"`
	Minimum            *AlertRuleTriggerMinimum `tfsdk:"minimum"`
}

type AlertRuleTriggerMinimum struct {
	Output    types.String  `tfsdk:"output"`
	Threshold types.Float64 `tfsdk:"threshold"`
}

type AlertRuleNotificationSettings struct {
	EnableTeamOwnerNotifications types.Bool   `tfsdk:"enable_team_owner_notifications"`
	IncidentIORoutingKey         types.String `tfsdk:"incident_io_routing_key"`
}

var alertRuleTriggerAttrType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType, "filter": types.StringType,
}}

var alertRuleMetricAttrType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"metric": types.StringType, "aggregation": types.StringType, "per": types.StringType,
	"normalize": types.StringType, "dimensions": types.SetType{ElemType: types.StringType}, "filter": types.StringType,
}}

func alertRuleScopeToClient(ctx context.Context, scope *AlertRuleScope) (client.AlertRuleScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	if scope == nil {
		return client.AlertRuleScope{}, diags
	}

	var projectIDs []string
	if !scope.ProjectIDs.IsNull() && !scope.ProjectIDs.IsUnknown() {
		diags.Append(scope.ProjectIDs.ElementsAs(ctx, &projectIDs, false)...)
	}
	return client.AlertRuleScope{
		Type:       scope.Type.ValueString(),
		ProjectID:  optionalString(scope.ProjectID),
		ProjectIDs: projectIDs,
	}, diags
}

func alertRuleTriggersToClient(ctx context.Context, value types.Set) (*client.AlertRuleTriggers, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	var models []AlertRuleTrigger
	diags.Append(value.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	items := make([]client.AlertRuleTrigger, 0, len(models))
	for _, model := range models {
		items = append(items, client.AlertRuleTrigger{Type: model.Type.ValueString(), Filter: optionalString(model.Filter)})
	}
	return &client.AlertRuleTriggers{Mode: "selected", Items: items}, diags
}

func alertRuleEvaluationToClient(ctx context.Context, evaluation *AlertRuleEvaluation) (*client.AlertRuleEvaluation, diag.Diagnostics) {
	var diags diag.Diagnostics
	if evaluation == nil || evaluation.Query == nil {
		return nil, diags
	}

	var metricModels map[string]AlertRuleMetricSelection
	diags.Append(evaluation.Query.Metrics.ElementsAs(ctx, &metricModels, false)...)
	if diags.HasError() {
		return nil, diags
	}
	metrics := make(map[string]client.AlertRuleMetricSelection, len(metricModels))
	for alias, model := range metricModels {
		var dimensions []string
		if !model.Dimensions.IsNull() && !model.Dimensions.IsUnknown() {
			diags.Append(model.Dimensions.ElementsAs(ctx, &dimensions, false)...)
		}
		metrics[alias] = client.AlertRuleMetricSelection{
			Metric:      model.Metric.ValueString(),
			Aggregation: model.Aggregation.ValueString(),
			Per:         optionalString(model.Per),
			Normalize:   optionalString(model.Normalize),
			Dimensions:  dimensions,
			Filter:      optionalString(model.Filter),
		}
	}
	if diags.HasError() {
		return nil, diags
	}

	var groupBy []string
	if !evaluation.Query.GroupBy.IsNull() && !evaluation.Query.GroupBy.IsUnknown() {
		diags.Append(evaluation.Query.GroupBy.ElementsAs(ctx, &groupBy, false)...)
	}
	var formulas map[string]string
	if !evaluation.Query.Formulas.IsNull() && !evaluation.Query.Formulas.IsUnknown() {
		diags.Append(evaluation.Query.Formulas.ElementsAs(ctx, &formulas, false)...)
	}
	var outputs []string
	diags.Append(evaluation.Query.Outputs.ElementsAs(ctx, &outputs, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return &client.AlertRuleEvaluation{
		Window: evaluation.Window.ValueString(),
		Query: client.AlertRuleCustomQuery{
			GroupBy: groupBy, Metrics: metrics, Formulas: formulas, Outputs: outputs,
		},
	}, diags
}

func alertRuleCustomTriggerToClient(trigger *AlertRuleCustomTrigger) *client.AlertRuleCustomTrigger {
	if trigger == nil {
		return nil
	}
	result := &client.AlertRuleCustomTrigger{
		Type:     trigger.Type.ValueString(),
		Output:   trigger.Output.ValueString(),
		Operator: optionalString(trigger.Operator),
	}
	if !trigger.Threshold.IsNull() && !trigger.Threshold.IsUnknown() {
		value := trigger.Threshold.ValueFloat64()
		result.Threshold = &value
	}
	if !trigger.StandardDeviations.IsNull() && !trigger.StandardDeviations.IsUnknown() {
		value := trigger.StandardDeviations.ValueFloat64()
		result.StandardDeviations = &value
	}
	if trigger.Minimum != nil {
		result.Minimum = &client.AlertRuleTriggerMinimum{
			Output: trigger.Minimum.Output.ValueString(), Threshold: trigger.Minimum.Threshold.ValueFloat64(),
		}
	}
	return result
}

func alertRuleNotificationSettingsToClient(settings *AlertRuleNotificationSettings) *client.AlertRuleNotificationSettings {
	if settings == nil {
		return nil
	}
	enableTeamOwnerNotifications := true
	if !settings.EnableTeamOwnerNotifications.IsNull() && !settings.EnableTeamOwnerNotifications.IsUnknown() {
		enableTeamOwnerNotifications = settings.EnableTeamOwnerNotifications.ValueBool()
	}
	return &client.AlertRuleNotificationSettings{
		EnableTeamOwnerNotifications: enableTeamOwnerNotifications,
		IncidentIORoutingKey:         optionalString(settings.IncidentIORoutingKey),
	}
}

func (model AlertRule) toCreateRequest(ctx context.Context) (client.AlertRuleCreate, diag.Diagnostics) {
	scope, diags := alertRuleScopeToClient(ctx, model.RuleScope)
	triggers, triggerDiags := alertRuleTriggersToClient(ctx, model.Triggers)
	diags.Append(triggerDiags...)
	evaluation, evaluationDiags := alertRuleEvaluationToClient(ctx, model.Evaluation)
	diags.Append(evaluationDiags...)
	if diags.HasError() {
		return client.AlertRuleCreate{}, diags
	}

	return client.AlertRuleCreate{
		Type:                      model.Type.ValueString(),
		Name:                      model.Name.ValueString(),
		RuleScope:                 scope,
		Triggers:                  triggers,
		MatchMinimumSeverityLevel: optionalString(model.MatchMinimumSeverityLevel),
		Severity:                  optionalString(model.Severity),
		Evaluation:                evaluation,
		Trigger:                   alertRuleCustomTriggerToClient(model.Trigger),
		NotificationSettings:      alertRuleNotificationSettingsToClient(model.NotificationSettings),
	}, diags
}

func alertRuleScopeFromClient(ctx context.Context, scope client.AlertRuleScope) (*AlertRuleScope, diag.Diagnostics) {
	projectIDs := types.SetNull(types.StringType)
	var diags diag.Diagnostics
	if len(scope.ProjectIDs) > 0 {
		var converted diag.Diagnostics
		projectIDs, converted = types.SetValueFrom(ctx, types.StringType, scope.ProjectIDs)
		diags.Append(converted...)
	}
	return &AlertRuleScope{
		Type: types.StringValue(scope.Type), ProjectID: stringValue(scope.ProjectID), ProjectIDs: projectIDs,
	}, diags
}

func alertRuleTriggersFromClient(ctx context.Context, triggers *client.AlertRuleTriggers) (types.Set, diag.Diagnostics) {
	if triggers == nil {
		return types.SetNull(alertRuleTriggerAttrType), nil
	}
	items := triggers.Items
	if triggers.Mode == "all" {
		items = make([]client.AlertRuleTrigger, 0, len(client.AlertRuleBuiltInTriggerTypes))
		for _, triggerType := range client.AlertRuleBuiltInTriggerTypes {
			items = append(items, client.AlertRuleTrigger{Type: triggerType})
		}
	}
	models := make([]AlertRuleTrigger, 0, len(items))
	for _, item := range items {
		models = append(models, AlertRuleTrigger{Type: types.StringValue(item.Type), Filter: stringValue(item.Filter)})
	}
	return types.SetValueFrom(ctx, alertRuleTriggerAttrType, models)
}

func alertRuleEvaluationFromClient(ctx context.Context, evaluation *client.AlertRuleEvaluation) (*AlertRuleEvaluation, diag.Diagnostics) {
	var diags diag.Diagnostics
	if evaluation == nil {
		return nil, diags
	}

	metricModels := make(map[string]AlertRuleMetricSelection, len(evaluation.Query.Metrics))
	for alias, metric := range evaluation.Query.Metrics {
		dimensions := types.SetNull(types.StringType)
		if len(metric.Dimensions) > 0 {
			converted, dimensionDiags := types.SetValueFrom(ctx, types.StringType, metric.Dimensions)
			diags.Append(dimensionDiags...)
			dimensions = converted
		}
		metricModels[alias] = AlertRuleMetricSelection{
			Metric: types.StringValue(metric.Metric), Aggregation: types.StringValue(metric.Aggregation),
			Per: stringValue(metric.Per), Normalize: stringValue(metric.Normalize), Dimensions: dimensions, Filter: stringValue(metric.Filter),
		}
	}
	metrics, metricDiags := types.MapValueFrom(ctx, alertRuleMetricAttrType, metricModels)
	diags.Append(metricDiags...)

	groupBy := types.ListNull(types.StringType)
	if len(evaluation.Query.GroupBy) > 0 {
		converted, groupByDiags := types.ListValueFrom(ctx, types.StringType, evaluation.Query.GroupBy)
		diags.Append(groupByDiags...)
		groupBy = converted
	}
	formulas := types.MapNull(types.StringType)
	if len(evaluation.Query.Formulas) > 0 {
		converted, formulaDiags := types.MapValueFrom(ctx, types.StringType, evaluation.Query.Formulas)
		diags.Append(formulaDiags...)
		formulas = converted
	}
	outputs, outputDiags := types.ListValueFrom(ctx, types.StringType, evaluation.Query.Outputs)
	diags.Append(outputDiags...)

	return &AlertRuleEvaluation{
		Window: types.StringValue(evaluation.Window),
		Query:  &AlertRuleCustomQuery{GroupBy: groupBy, Metrics: metrics, Formulas: formulas, Outputs: outputs},
	}, diags
}

func alertRuleCustomTriggerFromClient(trigger *client.AlertRuleCustomTrigger) *AlertRuleCustomTrigger {
	if trigger == nil {
		return nil
	}
	result := &AlertRuleCustomTrigger{
		Type: types.StringValue(trigger.Type), Output: types.StringValue(trigger.Output), Operator: stringValue(trigger.Operator),
		Threshold: float64Value(trigger.Threshold), StandardDeviations: float64Value(trigger.StandardDeviations),
	}
	if trigger.Minimum != nil {
		result.Minimum = &AlertRuleTriggerMinimum{
			Output: types.StringValue(trigger.Minimum.Output), Threshold: types.Float64Value(trigger.Minimum.Threshold),
		}
	}
	return result
}

func alertRuleFromAPI(ctx context.Context, out client.AlertRule, teamID types.String) (AlertRule, diag.Diagnostics) {
	scope, diags := alertRuleScopeFromClient(ctx, out.RuleScope)
	triggers, triggerDiags := alertRuleTriggersFromClient(ctx, out.Triggers)
	diags.Append(triggerDiags...)
	evaluation, evaluationDiags := alertRuleEvaluationFromClient(ctx, out.Evaluation)
	diags.Append(evaluationDiags...)

	return AlertRule{
		ID: types.StringValue(out.ID), TeamID: teamID, Type: types.StringValue(out.Type), Name: types.StringValue(out.Name),
		RuleScope: scope, Triggers: triggers, MatchMinimumSeverityLevel: stringValue(out.MatchMinimumSeverityLevel),
		Severity: stringValue(out.Severity), Evaluation: evaluation, Trigger: alertRuleCustomTriggerFromClient(out.Trigger),
		NotificationSettings: &AlertRuleNotificationSettings{
			EnableTeamOwnerNotifications: types.BoolValue(out.NotificationSettings.EnableTeamOwnerNotifications),
			IncidentIORoutingKey:         stringValue(out.NotificationSettings.IncidentIORoutingKey),
		},
		IsDefault: types.BoolValue(out.IsDefault), QuerySupported: boolValue(out.QuerySupported),
		CreatedAt: int64Value(out.CreatedAt), UpdatedAt: int64Value(out.UpdatedAt),
	}, diags
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func float64Value(value *float64) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*value)
}

func boolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func int64Value(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

func (r *alertRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AlertRule
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Type.IsNull() || config.Type.IsUnknown() {
		return
	}

	ruleType := config.Type.ValueString()
	if config.RuleScope != nil && !config.RuleScope.Type.IsNull() && !config.RuleScope.Type.IsUnknown() {
		scopeType := config.RuleScope.Type.ValueString()
		switch ruleType {
		case client.AlertRuleTypeBuiltIn:
			if scopeType != "all" && scopeType != "include" && scopeType != "exclude" {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("type"), "Invalid built-in alert rule scope", "Built-in alert rules must use an `all`, `include`, or `exclude` scope.")
			}
			if scopeType == "all" && (!config.RuleScope.ProjectID.IsNull() || !config.RuleScope.ProjectIDs.IsNull()) {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope"), "Invalid built-in alert rule scope", "An `all` scope cannot set `project_id` or `project_ids`.")
			}
			if (scopeType == "include" || scopeType == "exclude") && config.RuleScope.ProjectIDs.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("project_ids"), "Invalid built-in alert rule scope", "An `include` or `exclude` scope must set `project_ids`.")
			}
			if !config.RuleScope.ProjectID.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("project_id"), "Invalid built-in alert rule scope", "Built-in alert rules cannot set `project_id`.")
			}
		case client.AlertRuleTypeCustom:
			if scopeType != "project" {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("type"), "Invalid custom alert rule scope", "Custom alert rules must use a `project` scope.")
			}
			if config.RuleScope.ProjectID.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("project_id"), "Invalid custom alert rule scope", "A custom alert rule must set `project_id`.")
			}
			if !config.RuleScope.ProjectIDs.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("project_ids"), "Invalid custom alert rule scope", "Custom alert rules cannot set `project_ids`.")
			}
		}
	}

	if ruleType == client.AlertRuleTypeBuiltIn {
		if config.Triggers.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("triggers"), "Invalid built-in alert rule", "A built-in alert rule must configure at least one trigger.")
		}
		if config.MatchMinimumSeverityLevel.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("match_minimum_severity_level"), "Invalid built-in alert rule", "A built-in alert rule must set `match_minimum_severity_level`.")
		}
		if !config.Severity.IsNull() || config.Evaluation != nil || config.Trigger != nil {
			resp.Diagnostics.AddAttributeError(path.Root("type"), "Invalid built-in alert rule", "Built-in alert rules cannot set `severity`, `evaluation`, or `trigger`.")
		}
		validateBuiltInTriggers(ctx, config.Triggers, resp)
		return
	}

	if !config.Triggers.IsNull() || !config.MatchMinimumSeverityLevel.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("type"), "Invalid custom alert rule", "Custom alert rules cannot set `triggers` or `match_minimum_severity_level`.")
	}
	if config.Severity.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("severity"), "Invalid custom alert rule", "A custom alert rule must set `severity`.")
	}
	if config.Evaluation == nil {
		resp.Diagnostics.AddAttributeError(path.Root("evaluation"), "Invalid custom alert rule", "A custom alert rule must set `evaluation`.")
	} else {
		validateCustomQuery(ctx, config.Evaluation, resp)
	}
	if config.Trigger == nil {
		resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Invalid custom alert rule", "A custom alert rule must set `trigger`.")
	} else {
		validateCustomTrigger(config.Trigger, resp)
	}
}

func validateBuiltInTriggers(ctx context.Context, value types.Set, resp *resource.ValidateConfigResponse) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	var triggers []AlertRuleTrigger
	resp.Diagnostics.Append(value.ElementsAs(ctx, &triggers, false)...)
	seen := map[string]bool{}
	for _, trigger := range triggers {
		if trigger.Type.IsUnknown() {
			continue
		}
		triggerType := trigger.Type.ValueString()
		if seen[triggerType] {
			resp.Diagnostics.AddAttributeError(path.Root("triggers"), "Duplicate built-in trigger", fmt.Sprintf("The trigger type %q can be configured only once.", triggerType))
		}
		seen[triggerType] = true
		if triggerType == "error_anomaly" && trigger.Filter.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("triggers"), "Missing error anomaly filter", "The `error_anomaly` trigger requires a KQL `filter` containing a status group.")
		}
		if triggerType != "error_anomaly" && triggerType != "usage_anomaly" && !trigger.Filter.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("triggers"), "Unsupported built-in trigger filter", fmt.Sprintf("The %q trigger does not support a filter.", triggerType))
		}
	}
}

func validateCustomQuery(ctx context.Context, evaluation *AlertRuleEvaluation, resp *resource.ValidateConfigResponse) {
	if evaluation.Query == nil || evaluation.Query.Metrics.IsUnknown() || evaluation.Query.Outputs.IsUnknown() {
		return
	}
	var metrics map[string]AlertRuleMetricSelection
	resp.Diagnostics.Append(evaluation.Query.Metrics.ElementsAs(ctx, &metrics, false)...)
	var formulas map[string]string
	if !evaluation.Query.Formulas.IsNull() && !evaluation.Query.Formulas.IsUnknown() {
		resp.Diagnostics.Append(evaluation.Query.Formulas.ElementsAs(ctx, &formulas, false)...)
	}
	var outputs []string
	resp.Diagnostics.Append(evaluation.Query.Outputs.ElementsAs(ctx, &outputs, false)...)
	if resp.Diagnostics.HasError() || len(outputs) != 1 {
		return
	}

	if len(metrics) == 1 && len(formulas) == 0 {
		for alias := range metrics {
			if outputs[0] != alias {
				resp.Diagnostics.AddAttributeError(path.Root("evaluation").AtName("query").AtName("outputs"), "Invalid custom alert output", fmt.Sprintf("A single-metric query must use its metric alias %q as the output.", alias))
			}
		}
		return
	}
	if len(metrics) != 2 || len(formulas) != 1 || formulas["formula"] == "" || outputs[0] != "formula" {
		resp.Diagnostics.AddAttributeError(path.Root("evaluation").AtName("query"), "Invalid custom alert query", "A ratio query must define exactly two metrics, one non-empty formula under the `formula` key, and `outputs = [\"formula\"]`.")
	}
}

func validateCustomTrigger(trigger *AlertRuleCustomTrigger, resp *resource.ValidateConfigResponse) {
	if trigger.Type.IsUnknown() {
		return
	}
	triggerType := trigger.Type.ValueString()
	if triggerType == "threshold" {
		if trigger.Operator.IsNull() || trigger.Threshold.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Invalid threshold trigger", "A threshold trigger must set `operator` and `threshold`.")
		}
		if !trigger.StandardDeviations.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger").AtName("standard_deviations"), "Invalid threshold trigger", "A threshold trigger cannot set `standard_deviations`.")
		}
		return
	}
	if !trigger.Operator.IsNull() || !trigger.Threshold.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Invalid anomaly trigger", "An anomaly trigger cannot set `operator` or `threshold`.")
	}
	if trigger.StandardDeviations.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("trigger").AtName("standard_deviations"), "Invalid anomaly trigger", "An anomaly trigger must set `standard_deviations`.")
	}
}

func (r *alertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := plan.toCreateRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateAlertRule(ctx, client.CreateAlertRuleRequest{TeamID: plan.TeamID.ValueString(), AlertRuleCreate: payload})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Alert Rule", fmt.Sprintf("Could not create Alert Rule, unexpected error: %s", err))
		return
	}
	teamID := toTeamID(r.client.TeamID(plan.TeamID.ValueString()))
	result, resultDiags := alertRuleFromAPI(ctx, out, teamID)
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created alert rule", map[string]any{"team_id": teamID.ValueString(), "alert_rule_id": out.ID})
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
		resp.Diagnostics.AddError("Error reading Alert Rule", fmt.Sprintf("Could not get Alert Rule %s, unexpected error: %s", state.ID.ValueString(), err))
		return
	}
	if out.IsDefault {
		resp.Diagnostics.AddError("Unsupported default Alert Rule", "The team default alert rule cannot be managed by `vercel_alert_rule` because the API only permits notification updates for it.")
		return
	}
	if out.Type == client.AlertRuleTypeCustom && out.QuerySupported != nil && !*out.QuerySupported {
		resp.Diagnostics.AddError("Unsupported custom Alert Rule query", "This custom alert rule uses a legacy query that the Alerts v3 API cannot represent. Recreate it with `vercel_alert_rule` before managing it with Terraform.")
		return
	}

	result, diags := alertRuleFromAPI(ctx, out, state.TeamID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read alert rule", map[string]any{"team_id": state.TeamID.ValueString(), "alert_rule_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := plan.toCreateRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateAlertRule(ctx, client.UpdateAlertRuleRequest{
		TeamID: plan.TeamID.ValueString(), ID: plan.ID.ValueString(), Type: &payload.Type, Name: &payload.Name,
		RuleScope: &payload.RuleScope, Triggers: payload.Triggers, MatchMinimumSeverityLevel: payload.MatchMinimumSeverityLevel,
		Severity: payload.Severity, Evaluation: payload.Evaluation, Trigger: payload.Trigger, NotificationSettings: payload.NotificationSettings,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Alert Rule", fmt.Sprintf("Could not update Alert Rule %s, unexpected error: %s", plan.ID.ValueString(), err))
		return
	}
	result, resultDiags := alertRuleFromAPI(ctx, out, plan.TeamID)
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "updated alert rule", map[string]any{"team_id": plan.TeamID.ValueString(), "alert_rule_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *alertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlertRule(ctx, state.ID.ValueString(), state.TeamID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error deleting Alert Rule", fmt.Sprintf("Could not delete Alert Rule %s, unexpected error: %s", state.ID.ValueString(), err))
		return
	}
	tflog.Info(ctx, "deleted alert rule", map[string]any{"team_id": state.TeamID.ValueString(), "alert_rule_id": state.ID.ValueString()})
}

func (r *alertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok || id == "" || (strings.Contains(req.ID, "/") && teamID == "") {
		resp.Diagnostics.AddError("Error importing Alert Rule", fmt.Sprintf("Invalid id %q. Expected `team_id/alert_rule_id` or `alert_rule_id`.", req.ID))
		return
	}

	out, err := r.client.GetAlertRule(ctx, id, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule", fmt.Sprintf("Could not get Alert Rule %s, unexpected error: %s", id, err))
		return
	}
	if out.IsDefault {
		resp.Diagnostics.AddError("Unsupported default Alert Rule", "The team default alert rule cannot be imported because the API only permits notification updates for it.")
		return
	}
	if out.Type == client.AlertRuleTypeCustom && out.QuerySupported != nil && !*out.QuerySupported {
		resp.Diagnostics.AddError("Unsupported custom Alert Rule query", "This custom alert rule uses a legacy query that the Alerts v3 API cannot represent and cannot be imported into Terraform.")
		return
	}

	resolvedTeamID := toTeamID(r.client.TeamID(teamID))
	result, diags := alertRuleFromAPI(ctx, out, resolvedTeamID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "imported alert rule", map[string]any{"team_id": resolvedTeamID.ValueString(), "alert_rule_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
