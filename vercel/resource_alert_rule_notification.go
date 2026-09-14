package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                     = &alertRuleNotificationResource{}
	_ resource.ResourceWithConfigure        = &alertRuleNotificationResource{}
	_ resource.ResourceWithConfigValidators = &alertRuleNotificationResource{}
	_ resource.ResourceWithImportState      = &alertRuleNotificationResource{}
)

func newAlertRuleNotificationResource() resource.Resource {
	return &alertRuleNotificationResource{}
}

type alertRuleNotificationResource struct {
	client *client.Client
}

func (r *alertRuleNotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule_notification"
}

func (r *alertRuleNotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *alertRuleNotificationResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("webhook_id"),
			path.MatchRoot("slack_channel_id"),
		),
	}
}

func (r *alertRuleNotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	identityPlanModifiers := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	identityValidators := []validator.String{
		stringvalidator.LengthBetween(1, 256),
		validateStringIsTrimmed(),
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a link between a Vercel alert rule and one existing notification destination. Exactly one account webhook or Slack channel can be linked by each resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this notification link. Format: `alert_rule_id/webhook/webhook_id` or `alert_rule_id/slack/slack_installation_id/slack_channel_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the team that owns the alert rule and notification destination. Required if a default team is not configured in the provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseNonNullStateForUnknown(),
				},
			},
			"alert_rule_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the alert rule. Use `ar_default` to manage a notification destination for the team's default rule.",
				PlanModifiers:       identityPlanModifiers,
				Validators:          identityValidators,
			},
			"webhook_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of an existing account webhook subscribed to the `alerts.triggered` event. Exactly one of `webhook_id` or `slack_channel_id` must be configured.",
				PlanModifiers:       identityPlanModifiers,
				Validators:          identityValidators,
			},
			"slack_channel_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the Slack channel. Exactly one of `webhook_id` or `slack_channel_id` must be configured.",
				PlanModifiers:       identityPlanModifiers,
				Validators:          identityValidators,
			},
			"slack_installation_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the completed Slack integration installation that owns the channel. It can be omitted when the team has exactly one completed Slack installation; the resolved ID is then stored in state. It must be configured when the team has multiple Slack installations.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseNonNullStateForUnknown(),
				},
				Validators: append(identityValidators,
					stringvalidator.AlsoRequires(path.MatchRoot("slack_channel_id")),
				),
			},
		},
	}
}

type AlertRuleNotification struct {
	ID                  types.String `tfsdk:"id"`
	TeamID              types.String `tfsdk:"team_id"`
	AlertRuleID         types.String `tfsdk:"alert_rule_id"`
	WebhookID           types.String `tfsdk:"webhook_id"`
	SlackChannelID      types.String `tfsdk:"slack_channel_id"`
	SlackInstallationID types.String `tfsdk:"slack_installation_id"`
}

func (notification AlertRuleNotification) target() (client.AlertRuleNotificationTarget, error) {
	webhookConfigured := !notification.WebhookID.IsNull() && !notification.WebhookID.IsUnknown()
	slackChannelConfigured := !notification.SlackChannelID.IsNull() && !notification.SlackChannelID.IsUnknown()
	slackInstallationConfigured := !notification.SlackInstallationID.IsNull() && !notification.SlackInstallationID.IsUnknown()

	switch {
	case webhookConfigured && !slackChannelConfigured && !slackInstallationConfigured:
		return client.AlertRuleNotificationTarget{
			Type:      client.AlertRuleNotificationTypeWebhook,
			WebhookID: notification.WebhookID.ValueString(),
		}, nil
	case !webhookConfigured && slackChannelConfigured:
		return client.AlertRuleNotificationTarget{
			Type:      client.AlertRuleNotificationTypeSlack,
			ConfigID:  optionalStringValue(notification.SlackInstallationID),
			ChannelID: notification.SlackChannelID.ValueString(),
		}, nil
	default:
		return client.AlertRuleNotificationTarget{}, fmt.Errorf("configure exactly one webhook or Slack channel; slack_installation_id may only be set with slack_channel_id")
	}
}

func optionalStringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func (notification AlertRuleNotification) resolvedTarget() (client.AlertRuleNotificationTarget, error) {
	target, err := notification.target()
	if err != nil {
		return client.AlertRuleNotificationTarget{}, err
	}
	if target.Type == client.AlertRuleNotificationTypeSlack && target.ConfigID == "" {
		return client.AlertRuleNotificationTarget{}, fmt.Errorf("slack_installation_id is not resolved")
	}
	return target, nil
}

func (notification *AlertRuleNotification) applyResolvedTarget(requested, resolved client.AlertRuleNotificationTarget) error {
	if resolved.Type != requested.Type {
		return fmt.Errorf("API returned notification type %q for requested type %q", resolved.Type, requested.Type)
	}

	switch resolved.Type {
	case client.AlertRuleNotificationTypeWebhook:
		if resolved.WebhookID == "" || resolved.WebhookID != requested.WebhookID {
			return fmt.Errorf("API returned webhook ID %q for requested webhook %q", resolved.WebhookID, requested.WebhookID)
		}
		notification.WebhookID = types.StringValue(resolved.WebhookID)
		notification.SlackChannelID = types.StringNull()
		notification.SlackInstallationID = types.StringNull()
	case client.AlertRuleNotificationTypeSlack:
		if resolved.ConfigID == "" {
			return fmt.Errorf("API did not return the resolved Slack installation ID")
		}
		if resolved.ChannelID == "" || resolved.ChannelID != requested.ChannelID {
			return fmt.Errorf("API returned Slack channel ID %q for requested channel %q", resolved.ChannelID, requested.ChannelID)
		}
		if requested.ConfigID != "" && resolved.ConfigID != requested.ConfigID {
			return fmt.Errorf("API returned Slack installation ID %q for requested installation %q", resolved.ConfigID, requested.ConfigID)
		}
		notification.WebhookID = types.StringNull()
		notification.SlackInstallationID = types.StringValue(resolved.ConfigID)
	default:
		return fmt.Errorf("API returned unsupported notification type %q", resolved.Type)
	}
	return nil
}

func (notification AlertRuleNotification) resourceID() (string, error) {
	target, err := notification.resolvedTarget()
	if err != nil {
		return "", err
	}
	if target.Type == client.AlertRuleNotificationTypeWebhook {
		return strings.Join([]string{notification.AlertRuleID.ValueString(), target.Type, target.WebhookID}, "/"), nil
	}
	return strings.Join([]string{notification.AlertRuleID.ValueString(), target.Type, target.ConfigID, target.ChannelID}, "/"), nil
}

func alertRuleNotificationExists(target client.AlertRuleNotificationTarget, notifications []client.AlertRuleNotification) bool {
	for _, notification := range notifications {
		switch target.Type {
		case client.AlertRuleNotificationTypeWebhook:
			if notification.Type == target.Type && notification.Webhook.ID == target.WebhookID {
				return true
			}
		case client.AlertRuleNotificationTypeSlack:
			if notification.Type == target.Type && notification.ConfigID == target.ConfigID && notification.ChannelID == target.ChannelID {
				return true
			}
		}
	}
	return false
}

func (r *alertRuleNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertRuleNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := plan.target()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Notification", err.Error())
		return
	}
	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for Alert Rule Notification", "Configure a default team in the provider or set `team_id` on the resource.")
		return
	}

	resolvedTarget, err := r.client.LinkAlertRuleNotification(ctx, client.AlertRuleNotificationRequest{
		TeamID:                      teamID,
		AlertRuleID:                 plan.AlertRuleID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error linking Alert Rule Notification", fmt.Sprintf("Could not link notification destination to Alert Rule %s, unexpected error: %s", plan.AlertRuleID.ValueString(), err))
		return
	}
	if err = plan.applyResolvedTarget(target, resolvedTarget); err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Notification response", fmt.Sprintf("The notification was linked to Alert Rule %s, but the API returned an invalid destination identity: %s", plan.AlertRuleID.ValueString(), err))
		return
	}

	id, err := plan.resourceID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Notification", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	plan.TeamID = types.StringValue(teamID)
	tflog.Info(ctx, "linked alert rule notification", map[string]any{
		"team_id":           teamID,
		"alert_rule_id":     plan.AlertRuleID.ValueString(),
		"notification_type": target.Type,
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *alertRuleNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertRuleNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Notification state", err.Error())
		return
	}
	notifications, err := r.client.GetAlertRuleNotifications(ctx, state.AlertRuleID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Alert Rule Notification", fmt.Sprintf("Could not read notification destinations for Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}
	if !alertRuleNotificationExists(target, notifications) {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "read alert rule notification", map[string]any{
		"team_id":           state.TeamID.ValueString(),
		"alert_rule_id":     state.AlertRuleID.ValueString(),
		"notification_type": target.Type,
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *alertRuleNotificationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Alert Rule Notification cannot be updated", "Changing an Alert Rule Notification must replace the resource.")
}

func (r *alertRuleNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRuleNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Notification state", err.Error())
		return
	}
	_, err = r.client.UnlinkAlertRuleNotification(ctx, client.AlertRuleNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		AlertRuleID:                 state.AlertRuleID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error unlinking Alert Rule Notification", fmt.Sprintf("Could not unlink notification destination from Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "unlinked alert rule notification", map[string]any{
		"team_id":           state.TeamID.ValueString(),
		"alert_rule_id":     state.AlertRuleID.ValueString(),
		"notification_type": target.Type,
	})
}

func (r *alertRuleNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	state, ok := parseAlertRuleNotificationImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Alert Rule Notification",
			fmt.Sprintf("Invalid id %q. Expected `[team_id/]alert_rule_id/webhook/webhook_id` or `[team_id/]alert_rule_id/slack/slack_installation_id/slack_channel_id`.", req.ID),
		)
		return
	}
	teamID := r.client.TeamID(state.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing Alert Rule Notification", "Configure a default team in the provider or include `team_id` in the import ID.")
		return
	}
	state.TeamID = types.StringValue(teamID)

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Notification", err.Error())
		return
	}
	notifications, err := r.client.GetAlertRuleNotifications(ctx, state.AlertRuleID.ValueString(), teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Notification", fmt.Sprintf("Could not read notification destinations for Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}
	if !alertRuleNotificationExists(target, notifications) {
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := state.resourceID()
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Notification", err.Error())
		return
	}
	state.ID = types.StringValue(id)
	tflog.Info(ctx, "imported alert rule notification", map[string]any{
		"team_id":           teamID,
		"alert_rule_id":     state.AlertRuleID.ValueString(),
		"notification_type": target.Type,
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func parseAlertRuleNotificationImportID(importID string) (AlertRuleNotification, bool) {
	parts := strings.Split(importID, "/")
	for _, part := range parts {
		if part == "" {
			return AlertRuleNotification{}, false
		}
	}

	state := AlertRuleNotification{
		ID:                  types.StringNull(),
		TeamID:              types.StringNull(),
		WebhookID:           types.StringNull(),
		SlackChannelID:      types.StringNull(),
		SlackInstallationID: types.StringNull(),
	}
	switch {
	case len(parts) == 3 && parts[1] == client.AlertRuleNotificationTypeWebhook:
		state.AlertRuleID = types.StringValue(parts[0])
		state.WebhookID = types.StringValue(parts[2])
	case len(parts) == 4 && parts[2] == client.AlertRuleNotificationTypeWebhook:
		state.TeamID = types.StringValue(parts[0])
		state.AlertRuleID = types.StringValue(parts[1])
		state.WebhookID = types.StringValue(parts[3])
	case len(parts) == 4 && parts[1] == client.AlertRuleNotificationTypeSlack:
		state.AlertRuleID = types.StringValue(parts[0])
		state.SlackInstallationID = types.StringValue(parts[2])
		state.SlackChannelID = types.StringValue(parts[3])
	case len(parts) == 5 && parts[2] == client.AlertRuleNotificationTypeSlack:
		state.TeamID = types.StringValue(parts[0])
		state.AlertRuleID = types.StringValue(parts[1])
		state.SlackInstallationID = types.StringValue(parts[3])
		state.SlackChannelID = types.StringValue(parts[4])
	default:
		return AlertRuleNotification{}, false
	}
	return state, true
}
