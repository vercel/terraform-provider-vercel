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
}

type AlertRuleNotificationTarget struct {
	Type      string `json:"type"`
	ConfigID  string `json:"configId,omitempty"`
	ChannelID string `json:"channelId,omitempty"`
	WebhookID string `json:"webhookId,omitempty"`
}

type AlertRuleNotificationRequest struct {
	TeamID      string `json:"-"`
	AlertRuleID string `json:"-"`
	AlertRuleNotificationTarget
}

type alertRuleNotificationsEnvelope struct {
	Notifications []AlertRuleNotification `json:"notifications"`
}

func (c *Client) GetAlertRuleNotifications(ctx context.Context, alertRuleID, teamID string) ([]AlertRuleNotification, error) {
	u := c.alertRuleURL(teamID, "/"+url.PathEscape(alertRuleID)+"/notifications", nil)
	tflog.Info(ctx, "getting alert rule notifications", map[string]any{"url": u})

	var response alertRuleNotificationsEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &response)
	return response.Notifications, err
}

func (c *Client) LinkAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest) error {
	return c.mutateAlertRuleNotification(ctx, request, "POST")
}

func (c *Client) UnlinkAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest) error {
	return c.mutateAlertRuleNotification(ctx, request, "DELETE")
}

func (c *Client) mutateAlertRuleNotification(ctx context.Context, request AlertRuleNotificationRequest, method string) error {
	u := c.alertRuleURL(request.TeamID, "/"+url.PathEscape(request.AlertRuleID)+"/notifications/links", nil)
	tflog.Info(ctx, "mutating alert rule notification", map[string]any{
		"url":               u,
		"method":            method,
		"notification_type": request.Type,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: method,
		url:    u,
		body:   string(mustMarshal(request.AlertRuleNotificationTarget)),
	}, nil)
}
