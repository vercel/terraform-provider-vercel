package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &monitorWebhookNotificationResource{}
	_ resource.ResourceWithConfigure   = &monitorWebhookNotificationResource{}
	_ resource.ResourceWithImportState = &monitorWebhookNotificationResource{}
)

func newMonitorWebhookNotificationResource() resource.Resource {
	return &monitorWebhookNotificationResource{}
}

type monitorWebhookNotificationResource struct {
	client *client.Client
}

type MonitorWebhookNotification struct {
	ID                   types.String `tfsdk:"id"`
	TeamID               types.String `tfsdk:"team_id"`
	MonitorID            types.String `tfsdk:"monitor_id"`
	WebhookID            types.String `tfsdk:"webhook_id"`
	MinimumSeverityLevel types.String `tfsdk:"minimum_severity_level"`
}

func (r *monitorWebhookNotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor_webhook_notification"
}

func (r *monitorWebhookNotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configuredClient, ok := configureAlertRuleNotificationResource(req, resp)
	if ok {
		r.client = configuredClient
	}
}

func (r *monitorWebhookNotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := monitorNotificationCommonAttributes("The ID of this webhook notification link. Format: `monitor_id/webhook_id`.")
	attributes["webhook_id"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "The ID of an existing account webhook subscribed to the `alerts.triggered` event.",
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Validators: []validator.String{
			stringvalidator.LengthBetween(1, 256),
			validateStringIsTrimmed(),
		},
	}
	attributes["minimum_severity_level"] = alertRuleNotificationMinimumSeverityLevelAttribute()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a link between a Vercel monitor and one existing account webhook.",
		Attributes:          attributes,
	}
}

func (notification MonitorWebhookNotification) target() client.AlertRuleNotificationTarget {
	return client.AlertRuleNotificationTarget{
		Type:      client.AlertRuleNotificationTypeWebhook,
		WebhookID: notification.WebhookID.ValueString(),
	}
}

func (notification MonitorWebhookNotification) resourceID() string {
	return strings.Join([]string{notification.MonitorID.ValueString(), notification.WebhookID.ValueString()}, "/")
}

func (r *monitorWebhookNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MonitorWebhookNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for Monitor Webhook Notification", "Configure a default team in the provider or set `team_id` on the resource.")
		return
	}
	target := plan.target()
	target.MinimumSeverityLevel = optionalString(plan.MinimumSeverityLevel)
	resolved, err := r.client.LinkMonitorNotification(ctx, client.MonitorNotificationRequest{
		TeamID:                      teamID,
		MonitorID:                   plan.MonitorID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error linking Monitor Webhook Notification", fmt.Sprintf("Could not link webhook to Monitor %s, unexpected error: %s", plan.MonitorID.ValueString(), err))
		return
	}
	if resolved.Type != target.Type || resolved.WebhookID == "" || resolved.WebhookID != target.WebhookID {
		resp.Diagnostics.AddError(
			"Invalid Monitor Webhook Notification response",
			fmt.Sprintf("The webhook was linked to Monitor %s, but the API returned webhook identity type=%q webhook_id=%q.", plan.MonitorID.ValueString(), resolved.Type, resolved.WebhookID),
		)
		return
	}
	if err := reconcileMonitorNotificationMinimumSeverityLevel(ctx, r.client, client.MonitorNotificationRequest{
		TeamID:                      teamID,
		MonitorID:                   plan.MonitorID.ValueString(),
		AlertRuleNotificationTarget: target,
	}, resolved); err != nil {
		resp.Diagnostics.AddError("Error updating Monitor Webhook Notification", fmt.Sprintf("The webhook was already linked to Monitor %s with a different minimum severity, and it could not be updated: %s", plan.MonitorID.ValueString(), err))
		return
	}

	plan.ID = types.StringValue(plan.resourceID())
	plan.TeamID = types.StringValue(teamID)
	tflog.Info(ctx, "linked monitor webhook notification", map[string]any{
		"team_id":    teamID,
		"monitor_id": plan.MonitorID.ValueString(),
		"webhook_id": plan.WebhookID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *monitorWebhookNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MonitorWebhookNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notifications, err := r.client.GetMonitorNotifications(ctx, state.MonitorID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Monitor Webhook Notification", fmt.Sprintf("Could not read notification destinations for Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}
	notification, ok := findAlertRuleNotification(state.target(), notifications)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	state.MinimumSeverityLevel = stringValue(notification.MinimumSeverityLevel)

	tflog.Info(ctx, "read monitor webhook notification", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"monitor_id": state.MonitorID.ValueString(),
		"webhook_id": state.WebhookID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *monitorWebhookNotificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MonitorWebhookNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target := state.target()
	target.MinimumSeverityLevel = optionalString(plan.MinimumSeverityLevel)
	if _, err := r.client.UpdateMonitorNotification(ctx, client.MonitorNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		MonitorID:                   state.MonitorID.ValueString(),
		AlertRuleNotificationTarget: target,
	}); err != nil {
		resp.Diagnostics.AddError("Error updating Monitor Webhook Notification", fmt.Sprintf("Could not update the webhook notification for Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}

	state.MinimumSeverityLevel = plan.MinimumSeverityLevel
	tflog.Info(ctx, "updated monitor webhook notification", map[string]any{
		"team_id":                state.TeamID.ValueString(),
		"monitor_id":             state.MonitorID.ValueString(),
		"webhook_id":             state.WebhookID.ValueString(),
		"minimum_severity_level": state.MinimumSeverityLevel.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *monitorWebhookNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MonitorWebhookNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UnlinkMonitorNotification(ctx, client.MonitorNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		MonitorID:                   state.MonitorID.ValueString(),
		AlertRuleNotificationTarget: state.target(),
	})
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error unlinking Monitor Webhook Notification", fmt.Sprintf("Could not unlink webhook from Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "unlinked monitor webhook notification", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"monitor_id": state.MonitorID.ValueString(),
		"webhook_id": state.WebhookID.ValueString(),
	})
}

func (r *monitorWebhookNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := validAlertRuleNotificationImportParts(req.ID, 2, 3)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Monitor Webhook Notification",
			fmt.Sprintf("Invalid id %q. Expected `[team_id/]monitor_id/webhook_id`.", req.ID),
		)
		return
	}

	state := MonitorWebhookNotification{
		ID:                   types.StringNull(),
		TeamID:               types.StringNull(),
		MinimumSeverityLevel: types.StringNull(),
	}
	if len(parts) == 2 {
		state.MonitorID = types.StringValue(parts[0])
		state.WebhookID = types.StringValue(parts[1])
	} else {
		state.TeamID = types.StringValue(parts[0])
		state.MonitorID = types.StringValue(parts[1])
		state.WebhookID = types.StringValue(parts[2])
	}

	teamID := r.client.TeamID(state.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing Monitor Webhook Notification", "Configure a default team in the provider or include `team_id` in the import ID.")
		return
	}
	state.TeamID = types.StringValue(teamID)
	notifications, err := r.client.GetMonitorNotifications(ctx, state.MonitorID.ValueString(), teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error importing Monitor Webhook Notification", fmt.Sprintf("Could not read notification destinations for Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}
	notification, ok := findAlertRuleNotification(state.target(), notifications)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	state.MinimumSeverityLevel = stringValue(notification.MinimumSeverityLevel)

	state.ID = types.StringValue(state.resourceID())
	tflog.Info(ctx, "imported monitor webhook notification", map[string]any{
		"team_id":    teamID,
		"monitor_id": state.MonitorID.ValueString(),
		"webhook_id": state.WebhookID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
