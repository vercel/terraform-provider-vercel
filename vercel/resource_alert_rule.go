package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                   = &alertRuleResource{}
	_ resource.ResourceWithConfigure      = &alertRuleResource{}
	_ resource.ResourceWithImportState    = &alertRuleResource{}
	_ resource.ResourceWithModifyPlan     = &alertRuleResource{}
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
		MarkdownDescription: "Creates a built-in Vercel alert rule using the Alerts v3 API. Built-in rules select anomaly triggers across a team scope. Notification channel links are managed separately from this resource.",
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
				MarkdownDescription: "The alert rule type. Currently only `built-in` is supported.",
				Validators: []validator.String{
					stringvalidator.OneOf(client.AlertRuleTypeBuiltIn),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A human-readable name for the alert rule.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
					validateStringIsTrimmed(),
				},
			},
			"rule_scope": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The projects affected by the rule. Use `all`, `include`, or `exclude`.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("all", "include", "exclude"),
						},
					},
					"project_ids": schema.SetAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "The project IDs included in or excluded from a built-in rule.",
						Validators: []validator.Set{
							setvalidator.SizeBetween(1, 100),
							setvalidator.ValueStringsAre(
								stringvalidator.LengthBetween(1, 256),
								validateStringIsTrimmed(),
							),
						},
					},
				},
			},
			"trigger_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The API trigger selection mode. `all` and an empty `selected` set are response-only legacy states; omit `triggers` to preserve either state after import.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"triggers": schema.SetNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The built-in anomaly triggers enabled for a built-in rule. A nonempty set is required when creating a rule. Omit this attribute to preserve a response-only legacy trigger mode after import.",
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The built-in anomaly trigger type.",
							Validators:          []validator.String{stringvalidator.OneOf(client.AlertRuleBuiltInTriggerTypes...)},
						},
						"filter": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A KQL filter for `error_anomaly` or `usage_anomaly`. See the trigger filter documentation above for supported fields and values.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 2048),
								validateStringIsTrimmed(),
							},
						},
					},
				},
			},
			"match_minimum_severity_level": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The minimum severity matched by a built-in rule.",
				Validators: []validator.String{
					stringvalidator.OneOf("low", "medium", "high", "critical"),
				},
			},
			"notification_settings": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Notification delivery settings stored on the rule. Notification channel links are managed separately.",
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseNonNullStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"enable_team_owner_notifications": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						// The parent object can be known while this omitted computed
						// child is unknown, so preserve the API-resolved value here too.
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
					},
					"incident_io_routing_key": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 256),
							validateStringIsTrimmed(),
						},
					},
				},
			},
			"is_default": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the immutable team default rule. Default rules cannot be managed by this resource.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Creation time as a Unix epoch timestamp in milliseconds.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Last update time as a Unix epoch timestamp in milliseconds.",
			},
		},
	}
}

type AlertRule struct {
	ID                        types.String    `tfsdk:"id"`
	TeamID                    types.String    `tfsdk:"team_id"`
	Type                      types.String    `tfsdk:"type"`
	Name                      types.String    `tfsdk:"name"`
	RuleScope                 *AlertRuleScope `tfsdk:"rule_scope"`
	TriggerMode               types.String    `tfsdk:"trigger_mode"`
	Triggers                  types.Set       `tfsdk:"triggers"`
	MatchMinimumSeverityLevel types.String    `tfsdk:"match_minimum_severity_level"`
	// A framework object is necessary here because an omitted Optional+Computed
	// nested attribute is unknown in the create plan, which a struct pointer
	// cannot represent.
	NotificationSettings types.Object `tfsdk:"notification_settings"`
	IsDefault            types.Bool   `tfsdk:"is_default"`
	CreatedAt            types.Int64  `tfsdk:"created_at"`
	UpdatedAt            types.Int64  `tfsdk:"updated_at"`
}

type AlertRuleScope struct {
	Type       types.String `tfsdk:"type"`
	ProjectIDs types.Set    `tfsdk:"project_ids"`
}

type AlertRuleTrigger struct {
	Type   types.String `tfsdk:"type"`
	Filter types.String `tfsdk:"filter"`
}

type AlertRuleNotificationSettings struct {
	EnableTeamOwnerNotifications types.Bool   `tfsdk:"enable_team_owner_notifications"`
	IncidentIORoutingKey         types.String `tfsdk:"incident_io_routing_key"`
}

var alertRuleTriggerAttrType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType, "filter": types.StringType,
}}

var alertRuleNotificationSettingsAttrType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"enable_team_owner_notifications": types.BoolType,
	"incident_io_routing_key":         types.StringType,
}}

const alertRuleCanonicalFiltersPrivateKey = "alert_rule_canonical_filters"

type alertRulePrivateState interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// alertRuleCanonicalFilters stores the API representation separately from the
// user-authored representation in Terraform state. A nil value means the
// trigger has no filter; a missing key means the trigger was not returned.
type alertRuleCanonicalFilters map[string]*string

type alertRuleFilterReconciliationMode int

const (
	// An apply response describes the configuration just accepted by the API, so
	// Terraform state must retain that configuration's representation.
	alertRuleFilterReconcileApply alertRuleFilterReconciliationMode = iota
	// A refresh retains the state representation only while the API's canonical
	// representation is unchanged. Otherwise the API value exposes remote drift.
	alertRuleFilterReconcileRefresh
)

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

func alertRuleNotificationSettingsToClient(ctx context.Context, value types.Object) (*client.AlertRuleNotificationSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		// The API owns the default when the Optional+Computed block is omitted.
		return nil, diags
	}

	var settings AlertRuleNotificationSettings
	diags.Append(value.As(ctx, &settings, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	enableTeamOwnerNotifications := true
	if !settings.EnableTeamOwnerNotifications.IsNull() && !settings.EnableTeamOwnerNotifications.IsUnknown() {
		enableTeamOwnerNotifications = settings.EnableTeamOwnerNotifications.ValueBool()
	}
	return &client.AlertRuleNotificationSettings{
		EnableTeamOwnerNotifications: enableTeamOwnerNotifications,
		IncidentIORoutingKey:         optionalString(settings.IncidentIORoutingKey),
	}, diags
}

func (model AlertRule) toCreateRequest(ctx context.Context) (client.AlertRuleCreate, diag.Diagnostics) {
	scope, diags := alertRuleScopeToClient(ctx, model.RuleScope)
	triggers, triggerDiags := alertRuleTriggersToClient(ctx, model.Triggers)
	diags.Append(triggerDiags...)
	notificationSettings, notificationSettingsDiags := alertRuleNotificationSettingsToClient(ctx, model.NotificationSettings)
	diags.Append(notificationSettingsDiags...)
	if diags.HasError() {
		return client.AlertRuleCreate{}, diags
	}

	return client.AlertRuleCreate{
		Type:                      model.Type.ValueString(),
		Name:                      model.Name.ValueString(),
		RuleScope:                 scope,
		Triggers:                  triggers,
		MatchMinimumSeverityLevel: optionalString(model.MatchMinimumSeverityLevel),
		NotificationSettings:      notificationSettings,
	}, diags
}

func alertRuleScopesEqual(a, b *AlertRuleScope) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Type.Equal(b.Type) && a.ProjectIDs.Equal(b.ProjectIDs)
}

// toUpdateRequest keeps PATCH semantics aligned with Alerts v3 by sending only
// fields that changed. This is required for readable legacy trigger filters:
// the API can preserve them when triggers is omitted, but may reject them if an
// unrelated update writes the legacy filter back through the current grammar.
func (plan AlertRule) toUpdateRequest(ctx context.Context, state AlertRule) (client.UpdateAlertRuleRequest, diag.Diagnostics) {
	request := client.UpdateAlertRuleRequest{
		TeamID: plan.TeamID.ValueString(),
		ID:     plan.ID.ValueString(),
	}
	var diags diag.Diagnostics

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		request.Name = &name
	}
	if !alertRuleScopesEqual(plan.RuleScope, state.RuleScope) {
		scope, scopeDiags := alertRuleScopeToClient(ctx, plan.RuleScope)
		diags.Append(scopeDiags...)
		request.RuleScope = &scope
	}
	if !plan.Triggers.Equal(state.Triggers) {
		triggers, triggerDiags := alertRuleTriggersToClient(ctx, plan.Triggers)
		diags.Append(triggerDiags...)
		request.Triggers = triggers
	}
	if !plan.MatchMinimumSeverityLevel.Equal(state.MatchMinimumSeverityLevel) {
		request.MatchMinimumSeverityLevel = optionalString(plan.MatchMinimumSeverityLevel)
	}
	if !plan.NotificationSettings.Equal(state.NotificationSettings) {
		notificationSettings, notificationSettingsDiags := alertRuleNotificationSettingsToClient(ctx, plan.NotificationSettings)
		diags.Append(notificationSettingsDiags...)
		request.NotificationSettings = notificationSettings
	}

	return request, diags
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
		Type: types.StringValue(scope.Type), ProjectIDs: projectIDs,
	}, diags
}

func alertRuleTriggersFromClient(ctx context.Context, triggers *client.AlertRuleTriggers) (types.Set, diag.Diagnostics) {
	if triggers == nil {
		return types.SetNull(alertRuleTriggerAttrType), nil
	}
	// `all` is a response-only API mode. Keep it distinct from `selected` rather
	// than expanding it into trigger objects that could not pass write validation
	// (notably a filterless error_anomaly trigger).
	if triggers.Mode == "all" {
		return types.SetNull(alertRuleTriggerAttrType), nil
	}
	models := make([]AlertRuleTrigger, 0, len(triggers.Items))
	for _, item := range triggers.Items {
		models = append(models, AlertRuleTrigger{Type: types.StringValue(item.Type), Filter: stringValue(item.Filter)})
	}
	return types.SetValueFrom(ctx, alertRuleTriggerAttrType, models)
}

func alertRuleFromAPI(ctx context.Context, out client.AlertRule, teamID types.String) (AlertRule, diag.Diagnostics) {
	scope, diags := alertRuleScopeFromClient(ctx, out.RuleScope)
	triggers, triggerDiags := alertRuleTriggersFromClient(ctx, out.Triggers)
	diags.Append(triggerDiags...)
	triggerMode := types.StringNull()
	if out.Triggers != nil {
		triggerMode = types.StringValue(out.Triggers.Mode)
	}
	notificationSettings, notificationSettingsDiags := types.ObjectValueFrom(ctx, alertRuleNotificationSettingsAttrType.AttrTypes, AlertRuleNotificationSettings{
		EnableTeamOwnerNotifications: types.BoolValue(out.NotificationSettings.EnableTeamOwnerNotifications),
		IncidentIORoutingKey:         stringValue(out.NotificationSettings.IncidentIORoutingKey),
	})
	diags.Append(notificationSettingsDiags...)

	return AlertRule{
		ID: types.StringValue(out.ID), TeamID: teamID, Type: types.StringValue(out.Type), Name: types.StringValue(out.Name),
		RuleScope: scope, TriggerMode: triggerMode, Triggers: triggers, MatchMinimumSeverityLevel: stringValue(out.MatchMinimumSeverityLevel),
		NotificationSettings: notificationSettings,
		IsDefault:            types.BoolValue(out.IsDefault),
		CreatedAt:            int64Value(out.CreatedAt), UpdatedAt: int64Value(out.UpdatedAt),
	}, diags
}

func alertRuleCanonicalFiltersFromClient(triggers *client.AlertRuleTriggers) alertRuleCanonicalFilters {
	filters := alertRuleCanonicalFilters{}
	if triggers == nil || triggers.Mode == "all" {
		return filters
	}
	for _, trigger := range triggers.Items {
		filters[trigger.Type] = trigger.Filter
	}
	return filters
}

func alertRuleFiltersEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// alertRuleTriggersPreservingFilters chooses which filter representation is
// served into Terraform state. The Alerts API parses KQL into its OData storage
// form and serializes it back to canonical KQL, so a semantically unchanged
// filter can have different text (for example, NOT statusGroup:4xx becomes
// statusGroup:5xx). Preserving the configured text after apply avoids changing
// a set element's hash. During refresh, the private canonical baseline lets us
// preserve that text only when the remote value is unchanged, so actual drift
// is still visible without duplicating the API's KQL parser in this provider.
func alertRuleTriggersPreservingFilters(
	ctx context.Context,
	apiTriggers types.Set,
	priorTriggers types.Set,
	previousCanonical alertRuleCanonicalFilters,
	currentCanonical alertRuleCanonicalFilters,
	mode alertRuleFilterReconciliationMode,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if apiTriggers.IsNull() || apiTriggers.IsUnknown() || priorTriggers.IsNull() || priorTriggers.IsUnknown() {
		return apiTriggers, diags
	}

	var apiModels []AlertRuleTrigger
	var priorModels []AlertRuleTrigger
	diags.Append(apiTriggers.ElementsAs(ctx, &apiModels, false)...)
	diags.Append(priorTriggers.ElementsAs(ctx, &priorModels, false)...)
	if diags.HasError() {
		return apiTriggers, diags
	}

	priorFilters := make(map[string]types.String, len(priorModels))
	for _, trigger := range priorModels {
		if !trigger.Type.IsNull() && !trigger.Type.IsUnknown() {
			priorFilters[trigger.Type.ValueString()] = trigger.Filter
		}
	}
	for i, trigger := range apiModels {
		if trigger.Type.IsNull() || trigger.Type.IsUnknown() {
			continue
		}
		triggerType := trigger.Type.ValueString()
		priorFilter, hasPriorFilter := priorFilters[triggerType]
		if !hasPriorFilter {
			continue
		}
		previousFilter, hadPreviousCanonical := previousCanonical[triggerType]
		currentFilter, hasCurrentCanonical := currentCanonical[triggerType]
		if mode == alertRuleFilterReconcileApply || (hadPreviousCanonical && hasCurrentCanonical && alertRuleFiltersEqual(previousFilter, currentFilter)) {
			apiModels[i].Filter = priorFilter
		}
	}

	preserved, convertedDiags := types.SetValueFrom(ctx, alertRuleTriggerAttrType, apiModels)
	diags.Append(convertedDiags...)
	return preserved, diags
}

func getAlertRuleCanonicalFilters(ctx context.Context, private alertRulePrivateState) (alertRuleCanonicalFilters, diag.Diagnostics) {
	encoded, diags := private.GetKey(ctx, alertRuleCanonicalFiltersPrivateKey)
	if diags.HasError() || len(encoded) == 0 {
		return nil, diags
	}
	var filters alertRuleCanonicalFilters
	if err := json.Unmarshal(encoded, &filters); err != nil {
		diags.AddError("Error decoding Alert Rule private state", fmt.Sprintf("Could not decode canonical alert rule filters: %s", err))
	}
	return filters, diags
}

func setAlertRuleCanonicalFilters(ctx context.Context, private alertRulePrivateState, filters alertRuleCanonicalFilters) diag.Diagnostics {
	var diags diag.Diagnostics
	encoded, err := json.Marshal(filters)
	if err != nil {
		diags.AddError("Error encoding Alert Rule private state", fmt.Sprintf("Could not encode canonical alert rule filters: %s", err))
		return diags
	}
	diags.Append(private.SetKey(ctx, alertRuleCanonicalFiltersPrivateKey, encoded)...)
	return diags
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func int64Value(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

func (r *alertRuleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config AlertRule
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The API can return legacy `all` and empty `selected` states, so triggers
	// must be Optional+Computed for existing/imported resources. Current creates,
	// however, support only an explicitly selected, nonempty trigger set.
	if req.State.Raw.IsNull() && config.Triggers.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("triggers"),
			"Missing built-in alert rule triggers",
			"A nonempty `triggers` set is required when creating a built-in alert rule.",
		)
		return
	}

	if !config.Triggers.IsNull() {
		// Every writable trigger configuration uses selected mode. Derive this
		// instead of blindly retaining state: an imported legacy `all` rule must
		// plan a mode change when the user starts managing its trigger set.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("trigger_mode"), types.StringValue("selected"))...)
	}
}

func (r *alertRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AlertRule
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.RuleScope != nil && !config.RuleScope.Type.IsNull() && !config.RuleScope.Type.IsUnknown() {
		scopeType := config.RuleScope.Type.ValueString()
		if scopeType == "all" && !config.RuleScope.ProjectIDs.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("rule_scope"), "Invalid built-in alert rule scope", "An `all` scope cannot set `project_ids`.")
		}
		if (scopeType == "include" || scopeType == "exclude") && config.RuleScope.ProjectIDs.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("rule_scope").AtName("project_ids"), "Invalid built-in alert rule scope", "An `include` or `exclude` scope must set `project_ids`.")
		}
	}

	validateBuiltInTriggers(ctx, config.Triggers, resp)
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
	currentCanonical := alertRuleCanonicalFiltersFromClient(out.Triggers)
	result.Triggers, resultDiags = alertRuleTriggersPreservingFilters(ctx, result.Triggers, plan.Triggers, nil, currentCanonical, alertRuleFilterReconcileApply)
	resp.Diagnostics.Append(resultDiags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, currentCanonical)...)
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
	previousCanonical, privateDiags := getAlertRuleCanonicalFilters(ctx, req.Private)
	resp.Diagnostics.Append(privateDiags...)
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
	if out.Type != client.AlertRuleTypeBuiltIn {
		resp.Diagnostics.AddError("Unsupported Alert Rule type", fmt.Sprintf("Alert Rule %s has type %q, but `vercel_alert_rule` currently supports only built-in alert rules.", out.ID, out.Type))
		return
	}

	result, diags := alertRuleFromAPI(ctx, out, state.TeamID)
	resp.Diagnostics.Append(diags...)
	currentCanonical := alertRuleCanonicalFiltersFromClient(out.Triggers)
	result.Triggers, diags = alertRuleTriggersPreservingFilters(ctx, result.Triggers, state.Triggers, previousCanonical, currentCanonical, alertRuleFilterReconcileRefresh)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, currentCanonical)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read alert rule", map[string]any{"team_id": state.TeamID.ValueString(), "alert_rule_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AlertRule
	var state AlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := plan.toUpdateRequest(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateAlertRule(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Alert Rule", fmt.Sprintf("Could not update Alert Rule %s, unexpected error: %s", plan.ID.ValueString(), err))
		return
	}
	result, resultDiags := alertRuleFromAPI(ctx, out, plan.TeamID)
	resp.Diagnostics.Append(resultDiags...)
	currentCanonical := alertRuleCanonicalFiltersFromClient(out.Triggers)
	result.Triggers, resultDiags = alertRuleTriggersPreservingFilters(ctx, result.Triggers, plan.Triggers, nil, currentCanonical, alertRuleFilterReconcileApply)
	resp.Diagnostics.Append(resultDiags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, currentCanonical)...)
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
	resolvedTeamID := r.client.TeamID(teamID)
	if resolvedTeamID == "" {
		resp.Diagnostics.AddError("Error importing Alert Rule", "Alerts v3 requires a team. Configure a default team in the provider or import using `team_id/alert_rule_id`.")
		return
	}

	out, err := r.client.GetAlertRule(ctx, id, resolvedTeamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule", fmt.Sprintf("Could not get Alert Rule %s, unexpected error: %s", id, err))
		return
	}
	if out.IsDefault {
		resp.Diagnostics.AddError("Unsupported default Alert Rule", "The team default alert rule cannot be imported because the API only permits notification updates for it.")
		return
	}
	if out.Type != client.AlertRuleTypeBuiltIn {
		resp.Diagnostics.AddError("Unsupported Alert Rule type", fmt.Sprintf("Alert Rule %s has type %q, but `vercel_alert_rule` currently supports only built-in alert rules.", out.ID, out.Type))
		return
	}

	teamIDValue := toTeamID(resolvedTeamID)
	result, diags := alertRuleFromAPI(ctx, out, teamIDValue)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, alertRuleCanonicalFiltersFromClient(out.Triggers))...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "imported alert rule", map[string]any{"team_id": teamIDValue.ValueString(), "alert_rule_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
