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
	_ resource.Resource                = &monitorSlackNotificationResource{}
	_ resource.ResourceWithConfigure   = &monitorSlackNotificationResource{}
	_ resource.ResourceWithImportState = &monitorSlackNotificationResource{}
)

func newMonitorSlackNotificationResource() resource.Resource {
	return &monitorSlackNotificationResource{}
}

type monitorSlackNotificationResource struct {
	client *client.Client
}

type MonitorSlackNotification struct {
	ID                   types.String `tfsdk:"id"`
	TeamID               types.String `tfsdk:"team_id"`
	MonitorID            types.String `tfsdk:"monitor_id"`
	SlackChannelID       types.String `tfsdk:"slack_channel_id"`
	SlackInstallationID  types.String `tfsdk:"slack_installation_id"`
	MinimumSeverityLevel types.String `tfsdk:"minimum_severity_level"`
}

func (r *monitorSlackNotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor_slack_notification"
}

func (r *monitorSlackNotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configuredClient, ok := configureAlertRuleNotificationResource(req, resp)
	if ok {
		r.client = configuredClient
	}
}

func (r *monitorSlackNotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	identityValidators := []validator.String{
		stringvalidator.LengthBetween(1, 256),
		validateStringIsTrimmed(),
	}
	attributes := monitorNotificationCommonAttributes("The ID of this Slack notification link. Format: `monitor_id/slack_installation_id/slack_channel_id`.")
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
	attributes["minimum_severity_level"] = alertRuleNotificationMinimumSeverityLevelAttribute()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a link between a Vercel monitor and one Slack channel.",
		Attributes:          attributes,
	}
}

func (notification MonitorSlackNotification) target() client.AlertRuleNotificationTarget {
	return client.AlertRuleNotificationTarget{
		Type:      client.AlertRuleNotificationTypeSlack,
		ConfigID:  alertRuleNotificationOptionalStringValue(notification.SlackInstallationID),
		ChannelID: notification.SlackChannelID.ValueString(),
	}
}

func (notification MonitorSlackNotification) resolvedTarget() (client.AlertRuleNotificationTarget, error) {
	target := notification.target()
	if target.ConfigID == "" {
		return client.AlertRuleNotificationTarget{}, fmt.Errorf("slack_installation_id is not resolved")
	}
	return target, nil
}

func (notification MonitorSlackNotification) resourceID() (string, error) {
	target, err := notification.resolvedTarget()
	if err != nil {
		return "", err
	}
	return strings.Join([]string{notification.MonitorID.ValueString(), target.ConfigID, target.ChannelID}, "/"), nil
}

func (r *monitorSlackNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MonitorSlackNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := r.client.TeamID(plan.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Missing team for Monitor Slack Notification", "Configure a default team in the provider or set `team_id` on the resource.")
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
		resp.Diagnostics.AddError("Error linking Monitor Slack Notification", fmt.Sprintf("Could not link Slack channel to Monitor %s, unexpected error: %s", plan.MonitorID.ValueString(), err))
		return
	}
	if resolved.Type != target.Type || resolved.ConfigID == "" || resolved.ChannelID == "" || resolved.ChannelID != target.ChannelID || (target.ConfigID != "" && resolved.ConfigID != target.ConfigID) {
		resp.Diagnostics.AddError(
			"Invalid Monitor Slack Notification response",
			fmt.Sprintf("The Slack channel was linked to Monitor %s, but the API returned Slack identity type=%q slack_installation_id=%q slack_channel_id=%q.", plan.MonitorID.ValueString(), resolved.Type, resolved.ConfigID, resolved.ChannelID),
		)
		return
	}

	reconcileTarget := target
	reconcileTarget.ConfigID = resolved.ConfigID
	if err := reconcileMonitorNotificationMinimumSeverityLevel(ctx, r.client, client.MonitorNotificationRequest{
		TeamID:                      teamID,
		MonitorID:                   plan.MonitorID.ValueString(),
		AlertRuleNotificationTarget: reconcileTarget,
	}, resolved); err != nil {
		resp.Diagnostics.AddError("Error updating Monitor Slack Notification", fmt.Sprintf("The Slack channel was already linked to Monitor %s with a different minimum severity, and it could not be updated: %s", plan.MonitorID.ValueString(), err))
		return
	}

	plan.SlackInstallationID = types.StringValue(resolved.ConfigID)
	id, err := plan.resourceID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Monitor Slack Notification", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	plan.TeamID = types.StringValue(teamID)
	tflog.Info(ctx, "linked monitor Slack notification", map[string]any{
		"team_id":               teamID,
		"monitor_id":            plan.MonitorID.ValueString(),
		"slack_installation_id": plan.SlackInstallationID.ValueString(),
		"slack_channel_id":      plan.SlackChannelID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *monitorSlackNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MonitorSlackNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Monitor Slack Notification state", err.Error())
		return
	}
	notifications, err := r.client.GetMonitorNotifications(ctx, state.MonitorID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Monitor Slack Notification", fmt.Sprintf("Could not read notification destinations for Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}
	notification, ok := findAlertRuleNotification(target, notifications)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	state.MinimumSeverityLevel = stringValue(notification.MinimumSeverityLevel)

	tflog.Info(ctx, "read monitor Slack notification", map[string]any{
		"team_id":               state.TeamID.ValueString(),
		"monitor_id":            state.MonitorID.ValueString(),
		"slack_installation_id": state.SlackInstallationID.ValueString(),
		"slack_channel_id":      state.SlackChannelID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *monitorSlackNotificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MonitorSlackNotification
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Monitor Slack Notification state", err.Error())
		return
	}
	target.MinimumSeverityLevel = optionalString(plan.MinimumSeverityLevel)
	if _, err := r.client.UpdateMonitorNotification(ctx, client.MonitorNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		MonitorID:                   state.MonitorID.ValueString(),
		AlertRuleNotificationTarget: target,
	}); err != nil {
		resp.Diagnostics.AddError("Error updating Monitor Slack Notification", fmt.Sprintf("Could not update the Slack notification for Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}

	state.MinimumSeverityLevel = plan.MinimumSeverityLevel
	tflog.Info(ctx, "updated monitor Slack notification", map[string]any{
		"team_id":                state.TeamID.ValueString(),
		"monitor_id":             state.MonitorID.ValueString(),
		"slack_installation_id":  state.SlackInstallationID.ValueString(),
		"slack_channel_id":       state.SlackChannelID.ValueString(),
		"minimum_severity_level": state.MinimumSeverityLevel.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *monitorSlackNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MonitorSlackNotification
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Monitor Slack Notification state", err.Error())
		return
	}
	_, err = r.client.UnlinkMonitorNotification(ctx, client.MonitorNotificationRequest{
		TeamID:                      state.TeamID.ValueString(),
		MonitorID:                   state.MonitorID.ValueString(),
		AlertRuleNotificationTarget: target,
	})
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error unlinking Monitor Slack Notification", fmt.Sprintf("Could not unlink Slack channel from Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "unlinked monitor Slack notification", map[string]any{
		"team_id":               state.TeamID.ValueString(),
		"monitor_id":            state.MonitorID.ValueString(),
		"slack_installation_id": state.SlackInstallationID.ValueString(),
		"slack_channel_id":      state.SlackChannelID.ValueString(),
	})
}

func (r *monitorSlackNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := validAlertRuleNotificationImportParts(req.ID, 3, 4)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Monitor Slack Notification",
			fmt.Sprintf("Invalid id %q. Expected `[team_id/]monitor_id/slack_installation_id/slack_channel_id`.", req.ID),
		)
		return
	}

	state := MonitorSlackNotification{
		ID:                   types.StringNull(),
		TeamID:               types.StringNull(),
		MinimumSeverityLevel: types.StringNull(),
	}
	if len(parts) == 3 {
		state.MonitorID = types.StringValue(parts[0])
		state.SlackInstallationID = types.StringValue(parts[1])
		state.SlackChannelID = types.StringValue(parts[2])
	} else {
		state.TeamID = types.StringValue(parts[0])
		state.MonitorID = types.StringValue(parts[1])
		state.SlackInstallationID = types.StringValue(parts[2])
		state.SlackChannelID = types.StringValue(parts[3])
	}

	teamID := r.client.TeamID(state.TeamID.ValueString())
	if teamID == "" {
		resp.Diagnostics.AddError("Error importing Monitor Slack Notification", "Configure a default team in the provider or include `team_id` in the import ID.")
		return
	}
	state.TeamID = types.StringValue(teamID)
	target, err := state.resolvedTarget()
	if err != nil {
		resp.Diagnostics.AddError("Error importing Monitor Slack Notification", err.Error())
		return
	}
	notifications, err := r.client.GetMonitorNotifications(ctx, state.MonitorID.ValueString(), teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error importing Monitor Slack Notification", fmt.Sprintf("Could not read notification destinations for Monitor %s, unexpected error: %s", state.MonitorID.ValueString(), err))
		return
	}
	notification, ok := findAlertRuleNotification(target, notifications)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	state.MinimumSeverityLevel = stringValue(notification.MinimumSeverityLevel)

	id, err := state.resourceID()
	if err != nil {
		resp.Diagnostics.AddError("Error importing Monitor Slack Notification", err.Error())
		return
	}
	state.ID = types.StringValue(id)
	tflog.Info(ctx, "imported monitor Slack notification", map[string]any{
		"team_id":               teamID,
		"monitor_id":            state.MonitorID.ValueString(),
		"slack_installation_id": state.SlackInstallationID.ValueString(),
		"slack_channel_id":      state.SlackChannelID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
