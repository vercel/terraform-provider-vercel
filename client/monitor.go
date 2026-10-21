package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Monitors are the renamed alert rules. The monitors API accepts the same
// request bodies and returns the same objects as the alert rules API, so the
// AlertRule types are reused. Only the paths and response envelopes differ.

type monitorEnvelope struct {
	Monitor AlertRule `json:"monitor"`
}

type monitorsPage struct {
	Monitors   []AlertRule `json:"monitors"`
	Pagination struct {
		Next *string `json:"next"`
	} `json:"pagination"`
}

func (c *Client) monitorURL(teamID string, path string, query url.Values) string {
	u := fmt.Sprintf("%s/alerts/v3/monitors%s", c.baseURL, path)
	if query == nil {
		query = url.Values{}
	}
	if resolvedTeamID := c.TeamID(teamID); resolvedTeamID != "" {
		query.Set("teamId", resolvedTeamID)
	}
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	return u
}

func (c *Client) CreateMonitor(ctx context.Context, request CreateAlertRuleRequest) (AlertRule, error) {
	u := c.monitorURL(request.TeamID, "", nil)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating monitor", map[string]any{"url": u})

	var response monitorEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "POST", url: u, body: payload}, &response)
	return response.Monitor, err
}

func (c *Client) GetMonitor(ctx context.Context, id, teamID string) (AlertRule, error) {
	u := c.monitorURL(teamID, "/"+url.PathEscape(id), nil)
	tflog.Info(ctx, "getting monitor", map[string]any{"url": u})

	var response monitorEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &response)
	return response.Monitor, err
}

func (c *Client) UpdateMonitor(ctx context.Context, request UpdateAlertRuleRequest) (AlertRule, error) {
	u := c.monitorURL(request.TeamID, "/"+url.PathEscape(request.ID), nil)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating monitor", map[string]any{"url": u})

	var response monitorEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "PATCH", url: u, body: payload}, &response)
	return response.Monitor, err
}

func (c *Client) DeleteMonitor(ctx context.Context, id, teamID string) error {
	u := c.monitorURL(teamID, "/"+url.PathEscape(id), nil)
	tflog.Info(ctx, "deleting monitor", map[string]any{"url": u})
	return c.doRequest(clientRequest{ctx: ctx, method: "DELETE", url: u}, nil)
}

func (c *Client) ListMonitors(ctx context.Context, teamID string) ([]AlertRule, error) {
	var monitors []AlertRule
	var cursor *string
	for {
		query := url.Values{"limit": []string{"100"}}
		if cursor != nil {
			query.Set("cursor", *cursor)
		}
		u := c.monitorURL(teamID, "", query)
		tflog.Info(ctx, "listing monitors", map[string]any{"url": u})

		var page monitorsPage
		if err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &page); err != nil {
			return nil, err
		}
		monitors = append(monitors, page.Monitors...)
		cursor = page.Pagination.Next
		if cursor == nil {
			return monitors, nil
		}
	}
}
