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
	_ resource.Resource                = &alertRuleSlackNotificationResource{}
	_ resource.ResourceWithConfigure   = &alertRuleSlackNotificationResource{}
	_ resource.ResourceWithImportState = &alertRuleSlackNotificationResource{}
)

func newAlertRuleSlackNotificationResource() resource.Resource {
	return &alertRuleSlackNotificationResource{}
}

type alertRuleSlackNotificationResource struct {
	client *client.Client
}

type AlertRuleSlackNotification struct {
	ID                  types.String `tfsdk:"id"`
	TeamID              types.String `tfsdk:"team_id"`
	AlertRuleID         types.String `tfsdk:"alert_rule_id"`
	SlackChannelID      types.String `tfsdk:"slack_channel_id"`
	SlackInstallationID types.String `tfsdk:"slack_installation_id"`
}

func (r *alertRuleSlackNotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule_slack_notification"
}

func (r *alertRuleSlackNotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configuredClient, ok := configureAlertRuleNotificationResource(req, resp)
	if ok {
		r.client = configuredClient
	}
}

func (r *alertRuleSlackNotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	identityValidators := []validator.String{
		stringvalidator.LengthBetween(1, 256),
		validateStringIsTrimmed(),
	}
	attributes := alertRuleNotificationCommonAttributes("The ID of this Slack notification link. Format: `alert_rule_id/slack_installation_id/slack_channel_id`.")
	attributes["slack_channel_id"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "The ID of the Slack channel.",
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Validators:          identityValidators,
	}
	attributes["slack_installation_id"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "The ID of the completed Slack integration installation that owns the channel. It can be omitted when the team has exactly one completed Slack installation; the resolved ID is then stored in state. It must be configured when the team has multiple Slack installations.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplaceIfConfigured(),
			stringplanmodifier.UseNonNullStateForUnknown(),
		},
		Validators: identityValidators,
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a link between a Vercel alert rule and one Slack channel.",
		Attributes:          attributes,
	}
}

func (notification AlertRuleSlackNotification) target() client.AlertRuleNotificationTarget {
	return client.AlertRuleNotificationTarget{
		Type:      client.AlertRuleNotificationTypeSlack,
		ConfigID:  optionalStringValue(notification.SlackInstallationID),
		ChannelID: notification.SlackChannelID.ValueString(),
	}
}

func (notification AlertRuleSlackNotification) resolvedTarget() (client.AlertRuleNotificationTarget, error) {
	target := notification.target()
	if target.ConfigID == "" {
		return client.AlertRuleNotificationTarget{}, fmt.Errorf("slack_installation_id is not resolved")
	}
	return target, nil
}

func (notification AlertRuleSlackNotification) resourceID() (string, error) {
	target, err := notification.resolvedTarget()
	if err != nil {
		return "", err
	}
	return strings.Join([]string{notification.AlertRuleID.ValueString(), target.ConfigID, target.ChannelID}, "/"), nil
}

func (r *alertRuleSlackNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertRuleSlackNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for Alert Rule Slack Notification", "Configure a default team in the provider or set `team_id` on the resource.")
		return
	}
	target := plan.target()
	resolved, err := r.client.LinkAlertRuleNotification(ctx, client.AlertRuleNotificationRequest{
		TeamID:                      teamID,
		AlertRuleID:                 plan.AlertRuleID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error linking Alert Rule Slack Notification", fmt.Sprintf("Could not link Slack channel to Alert Rule %s, unexpected error: %s", plan.AlertRuleID.ValueString(), err))
		return
	}
	if resolved.Type != target.Type || resolved.ConfigID == "" || resolved.ChannelID == "" || resolved.ChannelID != target.ChannelID || (target.ConfigID != "" && resolved.ConfigID != target.ConfigID) {
		resp.Diagnostics.AddError(
			"Invalid Alert Rule Slack Notification response",
			fmt.Sprintf("The Slack channel was linked to Alert Rule %s, but the API returned Slack identity type=%q slack_installation_id=%q slack_channel_id=%q.", plan.AlertRuleID.ValueString(), resolved.Type, resolved.ConfigID, resolved.ChannelID),
		)
		return
	}

	plan.SlackInstallationID = types.StringValue(resolved.ConfigID)
	id, err := plan.resourceID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Slack Notification", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	plan.TeamID = types.StringValue(teamID)
	tflog.Info(ctx, "linked alert rule Slack notification", map[string]any{
		"team_id":               teamID,
		"alert_rule_id":         plan.AlertRuleID.ValueString(),
		"slack_installation_id": plan.SlackInstallationID.ValueString(),
		"slack_channel_id":      plan.SlackChannelID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *alertRuleSlackNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertRuleSlackNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Slack Notification state", err.Error())
		return
	}
	notifications, err := r.client.GetAlertRuleNotifications(ctx, state.AlertRuleID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Alert Rule Slack Notification", fmt.Sprintf("Could not read notification destinations for Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}
	if !alertRuleNotificationExists(target, notifications) {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "read alert rule Slack notification", map[string]any{
		"team_id":               state.TeamID.ValueString(),
		"alert_rule_id":         state.AlertRuleID.ValueString(),
		"slack_installation_id": state.SlackInstallationID.ValueString(),
		"slack_channel_id":      state.SlackChannelID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *alertRuleSlackNotificationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Alert Rule Slack Notification cannot be updated", "Changing an Alert Rule Slack Notification must replace the resource.")
}

func (r *alertRuleSlackNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRuleSlackNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Alert Rule Slack Notification state", err.Error())
		return
	}
	_, err = r.client.UnlinkAlertRuleNotification(ctx, client.AlertRuleNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		AlertRuleID:                 state.AlertRuleID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error unlinking Alert Rule Slack Notification", fmt.Sprintf("Could not unlink Slack channel from Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "unlinked alert rule Slack notification", map[string]any{
		"team_id":               state.TeamID.ValueString(),
		"alert_rule_id":         state.AlertRuleID.ValueString(),
		"slack_installation_id": state.SlackInstallationID.ValueString(),
		"slack_channel_id":      state.SlackChannelID.ValueString(),
	})
}

func (r *alertRuleSlackNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := validAlertRuleNotificationImportParts(req.ID, 3, 4)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Alert Rule Slack Notification",
			fmt.Sprintf("Invalid id %q. Expected `[team_id/]alert_rule_id/slack_installation_id/slack_channel_id`.", req.ID),
		)
		return
	}

	state := AlertRuleSlackNotification{
		ID:     types.StringNull(),
		TeamID: types.StringNull(),
	}
	if len(parts) == 3 {
		state.AlertRuleID = types.StringValue(parts[0])
		state.SlackInstallationID = types.StringValue(parts[1])
		state.SlackChannelID = types.StringValue(parts[2])
	} else {
		state.TeamID = types.StringValue(parts[0])
		state.AlertRuleID = types.StringValue(parts[1])
		state.SlackInstallationID = types.StringValue(parts[2])
		state.SlackChannelID = types.StringValue(parts[3])
	}

	teamID := r.client.TeamID(state.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing Alert Rule Slack Notification", "Configure a default team in the provider or include `team_id` in the import ID.")
		return
	}
	state.TeamID = types.StringValue(teamID)
	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Slack Notification", err.Error())
		return
	}
	notifications, err := r.client.GetAlertRuleNotifications(ctx, state.AlertRuleID.ValueString(), teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Slack Notification", fmt.Sprintf("Could not read notification destinations for Alert Rule %s, unexpected error: %s", state.AlertRuleID.ValueString(), err))
		return
	}
	if !alertRuleNotificationExists(target, notifications) {
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := state.resourceID()
	if err != nil {
		resp.Diagnostics.AddError("Error importing Alert Rule Slack Notification", err.Error())
		return
	}
	state.ID = types.StringValue(id)
	tflog.Info(ctx, "imported alert rule Slack notification", map[string]any{
		"team_id":               teamID,
		"alert_rule_id":         state.AlertRuleID.ValueString(),
		"slack_installation_id": state.SlackInstallationID.ValueString(),
		"slack_channel_id":      state.SlackChannelID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
