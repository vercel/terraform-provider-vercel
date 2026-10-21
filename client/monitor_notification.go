package client

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// The monitors notification endpoints behave exactly like the alert rule ones
// and keep the same payloads, so the AlertRuleNotification types are reused.

type MonitorNotificationRequest struct {
	TeamID    string `json:"-"`
	MonitorID string `json:"-"`
	AlertRuleNotificationTarget
}

func (c *Client) GetMonitorNotifications(ctx context.Context, monitorID, teamID string) ([]AlertRuleNotification, error) {
	u := c.monitorURL(teamID, "/"+url.PathEscape(monitorID)+"/notifications", nil)
	tflog.Info(ctx, "getting monitor notifications", map[string]any{"url": u})

	var response alertRuleNotificationsEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &response)
	return response.Notifications, err
}

func (c *Client) LinkMonitorNotification(ctx context.Context, request MonitorNotificationRequest) (AlertRuleNotificationTarget, error) {
	return c.mutateMonitorNotification(ctx, request, "POST", request.AlertRuleNotificationTarget)
}

// UpdateMonitorNotification sets the minimum severity of an existing link.
// A nil MinimumSeverityLevel clears it.
func (c *Client) UpdateMonitorNotification(ctx context.Context, request MonitorNotificationRequest) (AlertRuleNotificationTarget, error) {
	return c.mutateMonitorNotification(ctx, request, "PATCH", alertRuleNotificationUpdateBody(request.AlertRuleNotificationTarget))
}

func (c *Client) UnlinkMonitorNotification(ctx context.Context, request MonitorNotificationRequest) (AlertRuleNotificationTarget, error) {
	target := request.AlertRuleNotificationTarget
	target.MinimumSeverityLevel = nil
	return c.mutateMonitorNotification(ctx, request, "DELETE", target)
}

func (c *Client) mutateMonitorNotification(ctx context.Context, request MonitorNotificationRequest, method string, body any) (AlertRuleNotificationTarget, error) {
	u := c.monitorURL(request.TeamID, "/"+url.PathEscape(request.MonitorID)+"/notifications/links", nil)
	tflog.Info(ctx, "mutating monitor notification", map[string]any{
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
