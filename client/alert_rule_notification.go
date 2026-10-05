package client

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	AlertRuleNotificationTypeSlack   = "slack"
	AlertRuleNotificationTypeWebhook = "webhook"
)

type AlertRuleNotificationWebhook struct {
	ID string `json:"id"`
}

type AlertRuleNotification struct {
	Type      string                       `json:"type"`
	ConfigID  string                       `json:"configId,omitempty"`
	ChannelID string                       `json:"channelId,omitempty"`
	Webhook   AlertRuleNotificationWebhook `json:"webhook,omitempty"`
	// MinimumSeverityLevel is set when the destination only receives alerts of
	// at least this severity.
	MinimumSeverityLevel *string `json:"minimumSeverityLevel,omitempty"`
}

type AlertRuleNotificationTarget struct {
	Type                 string  `json:"type"`
	ConfigID             string  `json:"configId,omitempty"`
	ChannelID            string  `json:"channelId,omitempty"`
	WebhookID            string  `json:"webhookId,omitempty"`
	MinimumSeverityLevel *string `json:"minimumSeverityLevel,omitempty"`
}

// alertRuleNotificationUpdateBody always serializes minimumSeverityLevel
// because the API preserves the current value when the field is omitted and
// clears it when the field is null.
type alertRuleNotificationUpdateBody struct {
	Type                 string  `json:"type"`
	ConfigID             string  `json:"configId,omitempty"`
	ChannelID            string  `json:"channelId,omitempty"`
	WebhookID            string  `json:"webhookId,omitempty"`
	MinimumSeverityLevel *string `json:"minimumSeverityLevel"`
}

type AlertRuleNotificationRequest struct {
	TeamID      string `json:"-"`
	AlertRuleID string `json:"-"`
	AlertRuleNotificationTarget
}

type alertRuleNotificationsEnvelope struct {
	Notifications []AlertRuleNotification `json:"notifications"`
}

type alertRuleNotificationMutationEnvelope struct {
	Notification AlertRuleNotificationTarget `json:"notification"`
}

func (c *Client) GetAlertRuleNotifications(ctx context.Context, alertRuleID, teamID string) ([]AlertRuleNotification, error) {
	u := c.alertRuleURL(teamID, "/"+url.PathEscape(alertRuleID)+"/notifications", nil)
	tflog.Info(ctx, "getting alert rule notifications", map[string]any{"url": u})

	var response alertRuleNotificationsEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &response)
	return response.Notifications, err
}

func (c *Client) LinkAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest) (AlertRuleNotificationTarget, error) {
	return c.mutateAlertRuleNotification(ctx, request, "POST", request.AlertRuleNotificationTarget)
}

// UpdateAlertRuleNotification sets the minimum severity of an existing link.
// A nil MinimumSeverityLevel clears it.
func (c *Client) UpdateAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest) (AlertRuleNotificationTarget, error) {
	return c.mutateAlertRuleNotification(ctx, request, "PATCH", alertRuleNotificationUpdateBody(request.AlertRuleNotificationTarget))
}

func (c *Client) UnlinkAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest) (AlertRuleNotificationTarget, error) {
	target := request.AlertRuleNotificationTarget
	target.MinimumSeverityLevel = nil
	return c.mutateAlertRuleNotification(ctx, request, "DELETE", target)
}

func (c *Client) mutateAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest, method string, body any) (AlertRuleNotificationTarget, error) {
	u := c.alertRuleURL(request.TeamID, "/"+url.PathEscape(request.AlertRuleID)+"/notifications/links", nil)
	tflog.Info(ctx, "mutating alert rule notification", map[string]any{
		"url":               u,
		"method":            method,
		"notification_type": request.Type,
	})
	var response alertRuleNotificationMutationEnvelope
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: method,
		url:    u,
		body:   string(mustMarshal(body)),
	}, &response)
	return response.Notification, err
}
