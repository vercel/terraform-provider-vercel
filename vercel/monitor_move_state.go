package vercel

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var (
	_ resource.ResourceWithMoveState = &monitorResource{}
	_ resource.ResourceWithMoveState = &customMonitorResource{}
	_ resource.ResourceWithMoveState = &monitorSlackNotificationResource{}
	_ resource.ResourceWithMoveState = &monitorWebhookNotificationResource{}
)

func schemaOf(ctx context.Context, r resource.Resource) *schema.Schema {
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	return &resp.Schema
}

// matchesMoveSource reports whether a move request comes from the given
// resource type of this provider. The hostname is ignored so that mirrors and
// the registry resolve to the same provider.
func matchesMoveSource(req resource.MoveStateRequest, typeName string) bool {
	return req.SourceTypeName == typeName && strings.HasSuffix(req.SourceProviderAddress, "/vercel")
}

// Alert rules were renamed to monitors and the state shapes are identical, so a
// `moved` block can convert a deprecated resource into its monitor equivalent.

func (r *monitorResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		SourceSchema: schemaOf(ctx, &alertRuleResource{}),
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if !matchesMoveSource(req, "vercel_alert_rule") {
				return
			}
			var state AlertRule
			resp.Diagnostics.Append(req.SourceState.Get(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, state)...)
		},
	}}
}

func (r *customMonitorResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		SourceSchema: schemaOf(ctx, &customAlertRuleResource{}),
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if !matchesMoveSource(req, "vercel_custom_alert_rule") {
				return
			}
			var state CustomAlertRule
			resp.Diagnostics.Append(req.SourceState.Get(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, state)...)
		},
	}}
}

func (r *monitorSlackNotificationResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		SourceSchema: schemaOf(ctx, &alertRuleSlackNotificationResource{}),
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if !matchesMoveSource(req, "vercel_alert_rule_slack_notification") {
				return
			}
			var state AlertRuleSlackNotification
			resp.Diagnostics.Append(req.SourceState.Get(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, MonitorSlackNotification{
				ID:                   state.ID,
				TeamID:               state.TeamID,
				MonitorID:            state.AlertRuleID,
				SlackChannelID:       state.SlackChannelID,
				SlackInstallationID:  state.SlackInstallationID,
				MinimumSeverityLevel: state.MinimumSeverityLevel,
			})...)
		},
	}}
}

func (r *monitorWebhookNotificationResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		SourceSchema: schemaOf(ctx, &alertRuleWebhookNotificationResource{}),
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if !matchesMoveSource(req, "vercel_alert_rule_webhook_notification") {
				return
			}
			var state AlertRuleWebhookNotification
			resp.Diagnostics.Append(req.SourceState.Get(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, MonitorWebhookNotification{
				ID:                   state.ID,
				TeamID:               state.TeamID,
				MonitorID:            state.AlertRuleID,
				WebhookID:            state.WebhookID,
				MinimumSeverityLevel: state.MinimumSeverityLevel,
			})...)
		},
	}}
}
