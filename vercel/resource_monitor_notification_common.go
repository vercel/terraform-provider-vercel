package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func monitorNotificationCommonAttributes(idDescription string) map[string]schema.Attribute {
	return notificationCommonAttributes(idDescription, "monitor", "monitor_id", "default monitor")
}

func reconcileMonitorNotificationMinimumSeverityLevel(ctx context.Context, c *client.Client, request client.MonitorNotificationRequest, resolved client.AlertRuleNotificationTarget) error {
	if optionalStringsEqual(request.MinimumSeverityLevel, resolved.MinimumSeverityLevel) {
		return nil
	}
	_, err := c.UpdateMonitorNotification(ctx, request)
	return err
}
