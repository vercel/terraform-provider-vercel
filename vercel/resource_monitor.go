package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                   = &monitorResource{}
	_ resource.ResourceWithConfigure      = &monitorResource{}
	_ resource.ResourceWithImportState    = &monitorResource{}
	_ resource.ResourceWithModifyPlan     = &monitorResource{}
	_ resource.ResourceWithValidateConfig = &monitorResource{}
)

func newMonitorResource() resource.Resource {
	return &monitorResource{}
}

// monitorResource manages built-in monitors, the renamed alert rules. It talks
// to the monitors API and shares its model and conversion helpers with
// alertRuleResource, including the provider-owned tag.
type monitorResource struct {
	client *client.Client
}

func (r *monitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *monitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *monitorResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	var base resource.SchemaResponse
	(&alertRuleResource{}).Schema(ctx, resource.SchemaRequest{}, &base)

	attributes := base.Schema.Attributes
	override := func(name string, description string) {
		switch attribute := attributes[name].(type) {
		case schema.StringAttribute:
			attribute.MarkdownDescription = description
			attributes[name] = attribute
		case schema.BoolAttribute:
			attribute.MarkdownDescription = description
			attributes[name] = attribute
		case schema.SetAttribute:
			attribute.MarkdownDescription = description
			attributes[name] = attribute
		case schema.SingleNestedAttribute:
			attribute.MarkdownDescription = description
			attributes[name] = attribute
		}
	}
	override("id", "The ID of the monitor.")
	override("team_id", "The ID of the team that owns the monitor. Required if a default team is not configured in the provider.")
	override("type", "The monitor type. Currently only `built-in` is supported.")
	override("name", "A human-readable name for the monitor.")
	override("rule_scope", "The projects affected by the monitor. Use `all`, `include`, or `exclude`.")
	override("tags", alertRuleTagsDescription("monitor"))
	override("notification_settings", "Notification delivery settings stored on the monitor. Notification destination links are managed with the `vercel_monitor_slack_notification` and `vercel_monitor_webhook_notification` resources.")
	override("is_default", "Whether this is the immutable team default monitor. Default monitors cannot be managed by this resource.")

	base.Schema.MarkdownDescription = "Creates a built-in Vercel monitor (formerly alert rule) using the monitors API. Built-in monitors select anomaly triggers across a team scope. Notification destination links are managed with the `vercel_monitor_slack_notification` and `vercel_monitor_webhook_notification` resources."
	base.Schema.DeprecationMessage = ""
	resp.Schema = base.Schema
}

func (r *monitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyBuiltInPlan(ctx, req, resp, "monitor")
}

func (r *monitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AlertRule
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateAlertRuleScope(ctx, config.RuleScope, "monitor", resp)
	validateBuiltInTriggers(ctx, config.Triggers, resp)
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	out, err := r.client.CreateMonitor(ctx, client.CreateAlertRuleRequest{TeamID: plan.TeamID.ValueString(), AlertRuleCreate: payload})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Monitor", fmt.Sprintf("Could not create Monitor, unexpected error: %s", err))
		return
	}
	teamID := toTeamID(r.client.TeamID(plan.TeamID.ValueString()))
	result, resultDiags := alertRuleFromAPI(ctx, out, teamID, plan.Tags)
	resp.Diagnostics.Append(resultDiags...)
	currentCanonical := alertRuleCanonicalFiltersFromClient(out.Triggers)
	result.Triggers, resultDiags = alertRuleTriggersPreservingFilters(ctx, result.Triggers, plan.Triggers, nil, currentCanonical, alertRuleFilterReconcileApply)
	resp.Diagnostics.Append(resultDiags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, currentCanonical)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created monitor", map[string]any{"team_id": teamID.ValueString(), "monitor_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	out, err := r.client.GetMonitor(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Monitor", fmt.Sprintf("Could not get Monitor %s, unexpected error: %s", state.ID.ValueString(), err))
		return
	}
	if out.IsDefault {
		resp.Diagnostics.AddError("Unsupported default Monitor", "The team default monitor cannot be managed by `vercel_monitor` because the API only permits notification updates for it.")
		return
	}
	if out.Type != client.AlertRuleTypeBuiltIn {
		resp.Diagnostics.AddError("Unsupported Monitor type", fmt.Sprintf("Monitor %s has type %q, but `vercel_monitor` currently supports only built-in monitors.", out.ID, out.Type))
		return
	}

	result, diags := alertRuleFromAPI(ctx, out, state.TeamID, state.Tags)
	resp.Diagnostics.Append(diags...)
	currentCanonical := alertRuleCanonicalFiltersFromClient(out.Triggers)
	result.Triggers, diags = alertRuleTriggersPreservingFilters(ctx, result.Triggers, state.Triggers, previousCanonical, currentCanonical, alertRuleFilterReconcileRefresh)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, currentCanonical)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read monitor", map[string]any{"team_id": state.TeamID.ValueString(), "monitor_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	out, err := r.client.UpdateMonitor(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Monitor", fmt.Sprintf("Could not update Monitor %s, unexpected error: %s", plan.ID.ValueString(), err))
		return
	}
	result, resultDiags := alertRuleFromAPI(ctx, out, plan.TeamID, plan.Tags)
	resp.Diagnostics.Append(resultDiags...)
	currentCanonical := alertRuleCanonicalFiltersFromClient(out.Triggers)
	result.Triggers, resultDiags = alertRuleTriggersPreservingFilters(ctx, result.Triggers, plan.Triggers, nil, currentCanonical, alertRuleFilterReconcileApply)
	resp.Diagnostics.Append(resultDiags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, currentCanonical)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "updated monitor", map[string]any{"team_id": plan.TeamID.ValueString(), "monitor_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMonitor(ctx, state.ID.ValueString(), state.TeamID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error deleting Monitor", fmt.Sprintf("Could not delete Monitor %s, unexpected error: %s", state.ID.ValueString(), err))
		return
	}
	tflog.Info(ctx, "deleted monitor", map[string]any{"team_id": state.TeamID.ValueString(), "monitor_id": state.ID.ValueString()})
}

func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok || id == "" || (strings.Contains(req.ID, "/") && teamID == "") {
		resp.Diagnostics.AddError("Error importing Monitor", fmt.Sprintf("Invalid id %q. Expected `team_id/monitor_id` or `monitor_id`.", req.ID))
		return
	}
	resolvedTeamID := r.client.TeamID(teamID)
	if resolvedTeamID == "" {
		resp.Diagnostics.AddError("Error importing Monitor", "Alerts v3 requires a team. Configure a default team in the provider or import using `team_id/monitor_id`.")
		return
	}

	out, err := r.client.GetMonitor(ctx, id, resolvedTeamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing Monitor", fmt.Sprintf("Could not get Monitor %s, unexpected error: %s", id, err))
		return
	}
	if out.IsDefault {
		resp.Diagnostics.AddError("Unsupported default Monitor", "The team default monitor cannot be imported because the API only permits notification updates for it.")
		return
	}
	if out.Type != client.AlertRuleTypeBuiltIn {
		resp.Diagnostics.AddError("Unsupported Monitor type", fmt.Sprintf("Monitor %s has type %q, but `vercel_monitor` currently supports only built-in monitors.", out.ID, out.Type))
		return
	}

	teamIDValue := toTeamID(resolvedTeamID)
	result, diags := alertRuleFromAPI(ctx, out, teamIDValue)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(setAlertRuleCanonicalFilters(ctx, resp.Private, alertRuleCanonicalFiltersFromClient(out.Triggers))...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "imported monitor", map[string]any{"team_id": teamIDValue.ValueString(), "monitor_id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
