package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                   = &customAlertRuleResource{}
	_ resource.ResourceWithConfigure      = &customAlertRuleResource{}
	_ resource.ResourceWithValidateConfig = &customAlertRuleResource{}
	_ resource.ResourceWithImportState    = &customAlertRuleResource{}
)

func newCustomAlertRuleResource() resource.Resource { return &customAlertRuleResource{} }

type customAlertRuleResource struct{ client *client.Client }

type CustomAlertRule struct {
	ID                   types.String `tfsdk:"id"`
	TeamID               types.String `tfsdk:"team_id"`
	Name                 types.String `tfsdk:"name"`
	ProjectID            types.String `tfsdk:"project_id"`
	Evaluation           types.Object `tfsdk:"evaluation"`
	Trigger              types.Object `tfsdk:"trigger"`
	Severity             types.String `tfsdk:"severity"`
	InvestigationPrompt  types.String `tfsdk:"investigation_prompt"`
	AgentTriageEnabled   types.Bool   `tfsdk:"agent_triage_enabled"`
	NotificationSettings types.Object `tfsdk:"notification_settings"`
	CreatedAt            types.Int64  `tfsdk:"created_at"`
	UpdatedAt            types.Int64  `tfsdk:"updated_at"`
}

type customAlertEvaluation struct {
	Window types.String `tfsdk:"window"`
	Query  types.String `tfsdk:"query"`
}

type customAlertTrigger struct {
	Type               types.String  `tfsdk:"type"`
	Output             types.String  `tfsdk:"output"`
	Operator           types.String  `tfsdk:"operator"`
	Threshold          types.Float64 `tfsdk:"threshold"`
	StandardDeviations types.Float64 `tfsdk:"standard_deviations"`
	Minimum            types.Object  `tfsdk:"minimum"`
}

type customAlertMinimum struct {
	Output    types.String  `tfsdk:"output"`
	Threshold types.Float64 `tfsdk:"threshold"`
}

var customAlertEvaluationAttrTypes = map[string]attr.Type{
	"window": types.StringType, "query": types.StringType,
}
var customAlertMinimumAttrTypes = map[string]attr.Type{
	"output": types.StringType, "threshold": types.Float64Type,
}
var customAlertTriggerAttrTypes = map[string]attr.Type{
	"type": types.StringType, "output": types.StringType, "operator": types.StringType,
	"threshold": types.Float64Type, "standard_deviations": types.Float64Type,
	"minimum": types.ObjectType{AttrTypes: customAlertMinimumAttrTypes},
}

func (r *customAlertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_alert_rule"
}

func (r *customAlertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if configured, ok := configureAlertRuleNotificationResource(req, resp); ok {
		r.client = configured
	}
}

func (r *customAlertRuleResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	var builtIn resource.SchemaResponse
	newAlertRuleResource().Schema(ctx, resource.SchemaRequest{}, &builtIn)
	attributes := map[string]schema.Attribute{}
	for _, name := range []string{"id", "team_id", "name", "notification_settings", "created_at", "updated_at"} {
		attributes[name] = builtIn.Schema.Attributes[name]
	}
	attributes["project_id"] = schema.StringAttribute{
		Required: true, MarkdownDescription: "The project evaluated by this custom rule. Creating a rule or changing its query requires Observability Plus on this project.",
		Validators: []validator.String{stringvalidator.LengthBetween(1, 256), validateStringIsTrimmed()},
	}
	attributes["severity"] = schema.StringAttribute{
		Required: true, MarkdownDescription: "Severity assigned when the rule triggers: `low`, `medium`, or `high`. Critical severity is assigned only by a completed agent investigation.",
		Validators: []validator.String{stringvalidator.OneOf("low", "medium", "high")},
	}
	attributes["investigation_prompt"] = schema.StringAttribute{
		Optional: true, MarkdownDescription: "Optional guidance stored for agent investigations. The API currently stores this prompt without passing it to the investigation workflow. Removing it clears the stored prompt.",
		Validators: []validator.String{stringvalidator.LengthAtMost(2000), validateStringIsTrimmed()},
	}
	attributes["agent_triage_enabled"] = schema.BoolAttribute{
		Optional: true, Computed: true, Default: booldefault.StaticBool(false),
		MarkdownDescription: "When true, publish notifications only after the alert is classified as Critical. Defaults to false.",
	}
	attributes["evaluation"] = schema.SingleNestedAttribute{
		Required: true, MarkdownDescription: "The metric query and evaluation window.",
		Attributes: map[string]schema.Attribute{
			"window": schema.StringAttribute{Required: true, MarkdownDescription: "Aggregation granularity and detection cadence: `5m`, `15m`, `1h`, or `1d`.", Validators: []validator.String{stringvalidator.OneOf("5m", "15m", "1h", "1d")}},
			"query":  schema.StringAttribute{Required: true, MarkdownDescription: "The Alerts v3 query as a JSON object. Prefer `jsonencode(...)`. Use one metric without formulas, or two metrics with a division formula named `formula`. Discover supported metrics with `vc metrics schema <metric-or-prefix>`.", Validators: []validator.String{validateJSONObject()}},
		},
	}
	attributes["trigger"] = schema.SingleNestedAttribute{
		Required: true, MarkdownDescription: "A fixed threshold or anomaly condition on the query output.",
		Attributes: map[string]schema.Attribute{
			"type":                schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("threshold", "anomaly")}, MarkdownDescription: "The condition type: `threshold` or `anomaly`."},
			"output":              schema.StringAttribute{Required: true, MarkdownDescription: "The query's single output alias."},
			"operator":            schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.OneOf("gt", "gte", "lt", "lte")}, MarkdownDescription: "Required for threshold triggers; omitted for anomaly triggers."},
			"threshold":           schema.Float64Attribute{Optional: true, MarkdownDescription: "Required numeric threshold for threshold triggers. Zero is supported."},
			"standard_deviations": schema.Float64Attribute{Optional: true, Validators: []validator.Float64{float64validator.AtLeast(0.1)}, MarkdownDescription: "Required anomaly threshold in standard deviations, at least 0.1."},
			"minimum": schema.SingleNestedAttribute{Optional: true, MarkdownDescription: "Optional evaluation floor. Allowed for anomalies or ratio thresholds using `gt` or `gte`.", Attributes: map[string]schema.Attribute{
				"output":    schema.StringAttribute{Required: true, MarkdownDescription: "The primitive metric alias, or ratio numerator alias."},
				"threshold": schema.Float64Attribute{Required: true, Validators: []validator.Float64{float64validator.AtLeast(0)}, MarkdownDescription: "Non-negative floor below which the rule is not evaluated."},
			}},
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a project-scoped custom Vercel alert rule using the Alerts v3 API. Supports threshold and anomaly conditions on a metric or same-event ratio. Custom alerts must be enabled for the team. Manage Slack and webhook links with `vercel_alert_rule_slack_notification` and `vercel_alert_rule_webhook_notification`.",
		Attributes:          attributes,
	}
}

func (r *customAlertRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config CustomAlertRule
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Trigger.IsNull() || config.Trigger.IsUnknown() {
		return
	}
	var trigger customAlertTrigger
	resp.Diagnostics.Append(config.Trigger.As(ctx, &trigger, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || trigger.Type.IsUnknown() {
		return
	}
	switch trigger.Type.ValueString() {
	case "threshold":
		if trigger.Operator.IsNull() || trigger.Threshold.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Incomplete threshold trigger", "Threshold triggers require `operator` and `threshold`.")
		}
		if !trigger.StandardDeviations.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Invalid threshold trigger", "Threshold triggers cannot set `standard_deviations`.")
		}
	case "anomaly":
		if trigger.StandardDeviations.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Incomplete anomaly trigger", "Anomaly triggers require `standard_deviations`.")
		}
		if !trigger.Operator.IsNull() || !trigger.Threshold.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Invalid anomaly trigger", "Anomaly triggers cannot set `operator` or `threshold`.")
		}
	}
}

func customAlertEvaluationToClient(ctx context.Context, value types.Object) (*client.CustomAlertEvaluation, diag.Diagnostics) {
	var model customAlertEvaluation
	diags := value.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	if model.Window.IsUnknown() || model.Query.IsUnknown() {
		diags.AddError("Unknown custom alert evaluation", "The evaluation must be known before applying the rule.")
		return nil, diags
	}
	var query map[string]json.RawMessage
	if err := json.Unmarshal([]byte(model.Query.ValueString()), &query); err != nil || query == nil {
		diags.AddError("Invalid custom alert query", "The query must be a JSON object.")
		return nil, diags
	}
	return &client.CustomAlertEvaluation{Window: model.Window.ValueString(), Query: json.RawMessage(model.Query.ValueString())}, diags
}

func customAlertTriggerToClient(ctx context.Context, value types.Object) (*client.CustomAlertTrigger, diag.Diagnostics) {
	var model customAlertTrigger
	diags := value.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	trigger := &client.CustomAlertTrigger{Type: model.Type.ValueString(), Output: model.Output.ValueString(), Operator: optionalString(model.Operator)}
	if !model.Threshold.IsNull() && !model.Threshold.IsUnknown() {
		value := model.Threshold.ValueFloat64()
		trigger.Threshold = &value
	}
	if !model.StandardDeviations.IsNull() && !model.StandardDeviations.IsUnknown() {
		value := model.StandardDeviations.ValueFloat64()
		trigger.StandardDeviations = &value
	}
	if !model.Minimum.IsNull() {
		var minimum customAlertMinimum
		diags.Append(model.Minimum.As(ctx, &minimum, basetypes.ObjectAsOptions{})...)
		trigger.Minimum = &client.CustomAlertMinimum{Output: minimum.Output.ValueString(), Threshold: minimum.Threshold.ValueFloat64()}
	}
	return trigger, diags
}

func (model CustomAlertRule) toCreateRequest(ctx context.Context) (client.CreateAlertRuleRequest, diag.Diagnostics) {
	evaluation, diags := customAlertEvaluationToClient(ctx, model.Evaluation)
	trigger, triggerDiags := customAlertTriggerToClient(ctx, model.Trigger)
	diags.Append(triggerDiags...)
	settings, settingsDiags := alertRuleNotificationSettingsToClient(ctx, model.NotificationSettings)
	diags.Append(settingsDiags...)
	severity := model.Severity.ValueString()
	triage := model.AgentTriageEnabled.ValueBool()
	return client.CreateAlertRuleRequest{TeamID: model.TeamID.ValueString(), AlertRuleCreate: client.AlertRuleCreate{
		Type: client.AlertRuleTypeCustom, Name: model.Name.ValueString(),
		RuleScope:  client.AlertRuleScope{Type: "project", ProjectID: model.ProjectID.ValueString()},
		Evaluation: evaluation, Trigger: trigger, Severity: &severity,
		InvestigationPrompt: optionalString(model.InvestigationPrompt), AgentTriageEnabled: &triage, NotificationSettings: settings,
	}}, diags
}

// Only changed fields are sent: metadata updates must not recompile the query
// or require an Observability Plus subscription on a previously enabled project.
func (plan CustomAlertRule) toUpdateRequest(ctx context.Context, state CustomAlertRule) (client.UpdateAlertRuleRequest, diag.Diagnostics) {
	request := client.UpdateAlertRuleRequest{ID: state.ID.ValueString(), TeamID: state.TeamID.ValueString()}
	var diags diag.Diagnostics
	if !plan.Name.Equal(state.Name) {
		request.Name = optionalString(plan.Name)
	}
	if !plan.ProjectID.Equal(state.ProjectID) {
		request.RuleScope = &client.AlertRuleScope{Type: "project", ProjectID: plan.ProjectID.ValueString()}
	}
	if !plan.Severity.Equal(state.Severity) {
		request.Severity = optionalString(plan.Severity)
	}
	if !plan.AgentTriageEnabled.Equal(state.AgentTriageEnabled) {
		value := plan.AgentTriageEnabled.ValueBool()
		request.AgentTriageEnabled = &value
	}
	if !plan.InvestigationPrompt.Equal(state.InvestigationPrompt) {
		encoded, err := json.Marshal(optionalString(plan.InvestigationPrompt))
		if err != nil {
			diags.AddError("Invalid investigation prompt", err.Error())
		} else {
			request.InvestigationPrompt = encoded
		}
	}
	if !plan.Evaluation.Equal(state.Evaluation) {
		value, d := customAlertEvaluationToClient(ctx, plan.Evaluation)
		diags.Append(d...)
		request.Evaluation = value
	}
	if !plan.Trigger.Equal(state.Trigger) {
		value, d := customAlertTriggerToClient(ctx, plan.Trigger)
		diags.Append(d...)
		request.Trigger = value
	}
	if !plan.NotificationSettings.Equal(state.NotificationSettings) {
		value, d := alertRuleNotificationSettingsToClient(ctx, plan.NotificationSettings)
		diags.Append(d...)
		request.NotificationSettings = value
	}
	return request, diags
}

func customAlertRuleFromAPI(ctx context.Context, rule client.AlertRule, teamID types.String) (CustomAlertRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	if rule.Type != client.AlertRuleTypeCustom || !rule.QuerySupported || rule.Evaluation == nil || rule.Trigger == nil {
		diags.AddError("Unsupported custom Alert Rule", "This resource requires a custom rule with `querySupported: true`. Legacy rules with unsupported queries must be recreated before Terraform can manage their query.")
		return CustomAlertRule{}, diags
	}
	evaluation, d := types.ObjectValueFrom(ctx, customAlertEvaluationAttrTypes, customAlertEvaluation{Window: types.StringValue(rule.Evaluation.Window), Query: types.StringValue(string(rule.Evaluation.Query))})
	diags.Append(d...)
	minimum := types.ObjectNull(customAlertMinimumAttrTypes)
	if rule.Trigger.Minimum != nil {
		minimum, d = types.ObjectValueFrom(ctx, customAlertMinimumAttrTypes, customAlertMinimum{Output: types.StringValue(rule.Trigger.Minimum.Output), Threshold: types.Float64Value(rule.Trigger.Minimum.Threshold)})
		diags.Append(d...)
	}
	trigger, d := types.ObjectValueFrom(ctx, customAlertTriggerAttrTypes, customAlertTrigger{
		Type: types.StringValue(rule.Trigger.Type), Output: types.StringValue(rule.Trigger.Output), Operator: stringValue(rule.Trigger.Operator),
		Threshold: customAlertFloat64Value(rule.Trigger.Threshold), StandardDeviations: customAlertFloat64Value(rule.Trigger.StandardDeviations), Minimum: minimum,
	})
	diags.Append(d...)
	settings, d := types.ObjectValueFrom(ctx, alertRuleNotificationSettingsAttrType.AttrTypes, AlertRuleNotificationSettings{
		EnableTeamOwnerNotifications: types.BoolValue(rule.NotificationSettings.EnableTeamOwnerNotifications), IncidentIORoutingKey: stringValue(rule.NotificationSettings.IncidentIORoutingKey),
	})
	diags.Append(d...)
	return CustomAlertRule{
		ID: types.StringValue(rule.ID), TeamID: teamID, Name: types.StringValue(rule.Name), ProjectID: types.StringValue(rule.RuleScope.ProjectID),
		Evaluation: evaluation, Trigger: trigger, Severity: stringValue(rule.Severity), InvestigationPrompt: stringValue(rule.InvestigationPrompt),
		AgentTriageEnabled: types.BoolValue(rule.AgentTriageEnabled), NotificationSettings: settings,
		CreatedAt: int64Value(rule.CreatedAt), UpdatedAt: int64Value(rule.UpdatedAt),
	}, diags
}

func customAlertFloat64Value(value *float64) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*value)
}

const customAlertCanonicalQueryKey = "custom_alert_canonical_query"

// As with built-in filters, the server canonicalizes KQL. Retain authored query
// text only while the saved canonical baseline matches the current API query.
func customAlertEvaluationPreservingQuery(ctx context.Context, api, prior types.Object, previous, current []byte, apply bool) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prior.IsNull() || prior.IsUnknown() {
		return api, diags
	}
	var priorModel, apiModel customAlertEvaluation
	diags.Append(prior.As(ctx, &priorModel, basetypes.ObjectAsOptions{})...)
	diags.Append(api.As(ctx, &apiModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return api, diags
	}
	before, beforeErr := normalizeJSON(string(previous))
	after, afterErr := normalizeJSON(string(current))
	if !priorModel.Query.IsNull() && !priorModel.Query.IsUnknown() && (apply || (beforeErr == nil && afterErr == nil && before == after)) {
		apiModel.Query = priorModel.Query
	}
	result, d := types.ObjectValueFrom(ctx, customAlertEvaluationAttrTypes, apiModel)
	diags.Append(d...)
	return result, diags
}

func (r *customAlertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomAlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for custom Alert Rule", "Configure a default team in the provider or set `team_id`.")
		return
	}
	payload, d := plan.toCreateRequest(ctx)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload.TeamID = teamID
	out, err := r.client.CreateAlertRule(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom Alert Rule", err.Error())
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, types.StringValue(teamID))
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	result.Evaluation, d = customAlertEvaluationPreservingQuery(ctx, result.Evaluation, plan.Evaluation, nil, out.Evaluation.Query, true)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, customAlertCanonicalQueryKey, out.Evaluation.Query)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *customAlertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomAlertRule
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
		resp.Diagnostics.AddError("Error reading custom Alert Rule", err.Error())
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, state.TeamID)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	previous, d := req.Private.GetKey(ctx, customAlertCanonicalQueryKey)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	result.Evaluation, d = customAlertEvaluationPreservingQuery(ctx, result.Evaluation, state.Evaluation, previous, out.Evaluation.Query, false)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, customAlertCanonicalQueryKey, out.Evaluation.Query)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *customAlertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CustomAlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, d := plan.toUpdateRequest(ctx, state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	previous, d := req.Private.GetKey(ctx, customAlertCanonicalQueryKey)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateAlertRule(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom Alert Rule", err.Error())
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, state.TeamID)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	result.Evaluation, d = customAlertEvaluationPreservingQuery(ctx, result.Evaluation, plan.Evaluation, previous, out.Evaluation.Query, payload.Evaluation != nil)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if payload.Evaluation == nil && !result.Evaluation.Equal(plan.Evaluation) {
		resp.Diagnostics.AddError("Custom Alert Rule query changed during update", "The API returned an evaluation that differs from the saved query baseline, but this update did not send an evaluation. Run terraform plan again to reconcile the remote change.")
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, customAlertCanonicalQueryKey, out.Evaluation.Query)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *customAlertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomAlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlertRule(ctx, state.ID.ValueString(), state.TeamID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error deleting custom Alert Rule", err.Error())
	}
}

func (r *customAlertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok || id == "" || (strings.Contains(req.ID, "/") && teamID == "") {
		resp.Diagnostics.AddError("Error importing custom Alert Rule", "Expected `team_id/alert_rule_id` or `alert_rule_id`.")
		return
	}
	teamID = r.client.TeamID(teamID)
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing custom Alert Rule", "Configure a default team or include `team_id` in the import ID.")
		return
	}
	out, err := r.client.GetAlertRule(ctx, id, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing custom Alert Rule", fmt.Sprintf("Could not get Alert Rule %s: %s", id, err))
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, types.StringValue(teamID))
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, customAlertCanonicalQueryKey, out.Evaluation.Query)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
