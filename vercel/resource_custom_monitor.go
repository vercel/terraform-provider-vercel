package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                   = &customMonitorResource{}
	_ resource.ResourceWithConfigure      = &customMonitorResource{}
	_ resource.ResourceWithValidateConfig = &customMonitorResource{}
	_ resource.ResourceWithImportState    = &customMonitorResource{}
)

func newCustomMonitorResource() resource.Resource { return &customMonitorResource{} }

// customMonitorResource manages custom monitors, the renamed custom alert
// rules. It talks to the monitors API and shares its model, query handling and
// conversion helpers with customAlertRuleResource.
type customMonitorResource struct{ client *client.Client }

func (r *customMonitorResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	var base, monitor resource.SchemaResponse
	(&customAlertRuleResource{}).Schema(ctx, resource.SchemaRequest{}, &base)
	(&monitorResource{}).Schema(ctx, resource.SchemaRequest{}, &monitor)

	attributes := base.Schema.Attributes
	for _, name := range []string{"id", "team_id", "name", "tags", "notification_settings", "created_at", "updated_at"} {
		attributes[name] = monitor.Schema.Attributes[name]
	}
	if attribute, ok := attributes["notification_settings"].(schema.SingleNestedAttribute); ok {
		attribute.MarkdownDescription = "Notification delivery settings stored on the monitor. Notification destination links are managed with the `vercel_monitor_slack_notification` and `vercel_monitor_webhook_notification` resources."
		attributes["notification_settings"] = attribute
	}
	if attribute, ok := attributes["project_id"].(schema.StringAttribute); ok {
		attribute.MarkdownDescription = "The project evaluated by this custom monitor. Creating a monitor or changing its query requires Observability Plus on this project."
		attributes["project_id"] = attribute
	}
	if attribute, ok := attributes["severity"].(schema.StringAttribute); ok {
		attribute.MarkdownDescription = "Severity assigned when the monitor triggers: `low`, `medium`, or `high`. Critical severity is assigned only by a completed agent investigation."
		attributes["severity"] = attribute
	}

	base.Schema.MarkdownDescription = "Creates a project-scoped custom Vercel monitor (formerly custom alert rule) using the monitors API. Supports threshold and anomaly conditions on a metric or same-event ratio. Custom monitors must be enabled for the team. Manage Slack and webhook links with `vercel_monitor_slack_notification` and `vercel_monitor_webhook_notification`."
	base.Schema.DeprecationMessage = ""
	resp.Schema = base.Schema
}

func (r *customMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_monitor"
}

func (r *customMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if configured, ok := configureAlertRuleNotificationResource(req, resp); ok {
		r.client = configured
	}
}

func (r *customMonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
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

func (r *customMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomAlertRule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for custom Monitor", "Configure a default team in the provider or set `team_id`.")
		return
	}
	payload, d := plan.toCreateRequest(ctx)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload.TeamID = teamID
	out, err := r.client.CreateMonitor(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom Monitor", err.Error())
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, types.StringValue(teamID), plan.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	result.Evaluation, d = customAlertEvaluationPreservingQuery(ctx, result.Evaluation, plan.Evaluation, nil, out.Evaluation.Query, true)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, customAlertCanonicalQueryKey, out.Evaluation.Query)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *customMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomAlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetMonitor(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom Monitor", err.Error())
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, state.TeamID, state.Tags)
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

func (r *customMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	out, err := r.client.UpdateMonitor(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom Monitor", err.Error())
		return
	}
	result, d := customAlertRuleFromAPI(ctx, out, state.TeamID, plan.Tags)
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
		resp.Diagnostics.AddError("Custom Monitor query changed during update", "The API returned an evaluation that differs from the saved query baseline, but this update did not send an evaluation. Run terraform plan again to reconcile the remote change.")
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, customAlertCanonicalQueryKey, out.Evaluation.Query)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *customMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomAlertRule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMonitor(ctx, state.ID.ValueString(), state.TeamID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error deleting custom Monitor", err.Error())
	}
}

func (r *customMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok || id == "" || (strings.Contains(req.ID, "/") && teamID == "") {
		resp.Diagnostics.AddError("Error importing custom Monitor", "Expected `team_id/monitor_id` or `monitor_id`.")
		return
	}
	teamID = r.client.TeamID(teamID)
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing custom Monitor", "Configure a default team or include `team_id` in the import ID.")
		return
	}
	out, err := r.client.GetMonitor(ctx, id, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing custom Monitor", fmt.Sprintf("Could not get Monitor %s: %s", id, err))
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
