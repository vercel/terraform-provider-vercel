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
	_ resource.Resource                = &alertRuleWebhookNotificationResource{}
	_ resource.ResourceWithConfigure   = &alertRuleWebhookNotificationResource{}
	_ resource.ResourceWithImportState = &alertRuleWebhookNotificationResource{}
)

func newAlertRuleWebhookNotificationResource() resource.Resource {
	return &alertRuleWebhookNotificationResource{}
}

type alertRuleWebhookNotificationResource struct {
	client *client.Client
}

type AlertRuleWebhookNotification struct {
	ID          types.String `tfsdk:"id"`
	TeamID      types.String `tfsdk:"team_id"`
	AlertRuleID types.String `tfsdk:"alert_rule_id"`
	WebhookID   types.String `tfsdk:"webhook_id"`
}

func (r *alertRuleWebhookNotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule_webhook_notification"
}

func (r *alertRuleWebhookNotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configuredClient, ok := configureAlertRuleNotificationResource(req, resp)
	if ok {
		r.client = configuredClient
	}
}

func (r *alertRuleWebhookNotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := alertRuleNotificationCommonAttributes("The ID of this webhook notification link. Format: `alert_rule_id/webhook_id`.")
	attributes["webhook_id"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "The ID of an existing account webhook subscribed to the `alerts.triggered` event.",
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Validators: []validator.String{
			stringvalidator.LengthBetween(1, 256),
			validateStringIsTrimmed(),
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a link between a Vercel alert rule and one existing account webhook.",
		Attributes:          attributes,
	}
}

func (notification AlertRuleWebhookNotification) target() client.AlertRuleNotificationTarget {
	return client.AlertRuleNotificationTarget{
		Type:      client.AlertRuleNotificationTypeWebhook,
		WebhookID: notification.WebhookID.ValueString(),
	}
}

func (notification AlertRuleWebhookNotification) resourceID() string {
	return strings.Join([]string{notification.AlertRuleID.ValueString(), notification.WebhookID.ValueString()}, "/")
}

func (r *alertRuleWebhookNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertRuleWebhookNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for Alert Rule Webhook Notification", "Configure a default team in the provider or set `team_id` on the resource.")
		return
	}
	target := plan.target()
	resolved, err := r.client.LinkAlertRuleNotification(ctx, client.AlertRuleNotificationRequest{
		TeamID:                      teamID,
		AlertRuleID:                 plan.AlertRuleID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error linking Alert Rule Webhook Notification", fmt.Sprintf("Could not link webhook to Alert Rule %s, unexpected error: %s", plan.AlertRuleID.ValueString(), err))
		return
	}
	if resolved.Type != target.Type || resolved.WebhookID == "" || resolved.WebhookID != target.WebhookID {
		resp.Diagnostics.AddError(
			"Invalid Alert Rule Webhook Notification response",
			fmt.Sprintf("The webhook was linked to Alert Rule %s, but the API returned webhook identity type=%q webhook_id=%q.", plan.AlertRuleID.ValueString(), resolved.Type, resolved.WebhookID),
		)
		return
	}

	plan.ID = types.StringValue(plan.resourceID())
	plan.TeamID = types.StringValue(teamID)
	tflog.Info(ctx, "linked alert rule webhook notification", map[string]any{
		"team_id":       teamID,
		"alert_rule_id": plan.AlertRuleID.ValueString(),
		"webhook_id":    plan.WebhookID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *alertRuleWebhookNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertRuleWebhookNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notifications, err := r.client.GetAlertRuleNotifications(ctx, state.AlertRuleID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Alert Rule Webhook Notification", fmt.Sprintf("Could not read notification destinations for Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}
	if !alertRuleNotificationExists(state.target(), notifications) {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "read alert rule webhook notification", map[string]any{
		"team_id":       state.TeamID.ValueString(),
		"alert_rule_id": state.AlertRuleID.ValueString(),
		"webhook_id":    state.WebhookID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *alertRuleWebhookNotificationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Alert Rule Webhook Notification cannot be updated", "Changing an Alert Rule Webhook Notification must replace the resource.")
}

func (r *alertRuleWebhookNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRuleWebhookNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UnlinkAlertRuleNotification(ctx, client.AlertRuleNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		AlertRuleID:                 state.AlertRuleID.ValueString(),
		AlertRuleNotificationTarget: state.target(),
	})
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error unlinking Alert Rule Webhook Notification", fmt.Sprintf("Could not unlink webhook from Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "unlinked alert rule webhook notification", map[string]any{
		"team_id":       state.TeamID.ValueString(),
		"alert_rule_id": state.AlertRuleID.ValueString(),
		"webhook_id":    state.WebhookID.ValueString(),
	})
}

func (r *alertRuleWebhookNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := validAlertRuleNotificationImportParts(req.ID, 2, 3)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Alert Rule Webhook Notification",
			fmt.Sprintf("Invalid id %q. Expected `[team_id/]alert_rule_id/webhook_id`.", req.ID),
		)
		return
	}

	state := AlertRuleWebhookNotification{
		ID:     types.StringNull(),
		TeamID: types.StringNull(),
	}
	if len(parts) == 2 {
		state.AlertRuleID = types.StringValue(parts[0])
		state.WebhookID = types.StringValue(parts[1])
	} else {
		state.TeamID = types.StringValue(parts[0])
		state.AlertRuleID = types.StringValue(parts[1])
		state.WebhookID = types.StringValue(parts[2])
	}

	teamID := r.client.TeamID(state.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing Alert Rule Webhook Notification", "Configure a default team in the provider or include `team_id` in the import ID.")
		return
	}
	state.TeamID = types.StringValue(teamID)
	notifications, err := r.client.GetAlertRuleNotifications(ctx, state.AlertRuleID.ValueString(), teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Webhook Notification", fmt.Sprintf("Could not read notification destinations for Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}
	if !alertRuleNotificationExists(state.target(), notifications) {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(state.resourceID())
	tflog.Info(ctx, "imported alert rule webhook notification", map[string]any{
		"team_id":       teamID,
		"alert_rule_id": state.AlertRuleID.ValueString(),
		"webhook_id":    state.WebhookID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
