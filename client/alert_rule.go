package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	AlertRuleTypeBuiltIn = "built-in"
)

var AlertRuleBuiltInTriggerTypes = []string{
	"usage_anomaly",
	"error_anomaly",
	"botId_anomaly",
	"botTraffic_anomaly",
	"firewallSystemRule_anomaly",
	"firewallCustomRule_anomaly",
	"testAlert_anomaly",
	"buildTime_anomaly",
}

type AlertRule struct {
	ID                        string                        `json:"id"`
	Type                      string                        `json:"type"`
	Name                      string                        `json:"name"`
	RuleScope                 AlertRuleScope                `json:"ruleScope"`
	Triggers                  *AlertRuleTriggers            `json:"triggers,omitempty"`
	MatchMinimumSeverityLevel *string                       `json:"matchMinimumSeverityLevel,omitempty"`
	NotificationSettings      AlertRuleNotificationSettings `json:"notificationSettings"`
	IsDefault                 bool                          `json:"isDefault"`
	CreatedAt                 *int64                        `json:"createdAt,omitempty"`
	UpdatedAt                 *int64                        `json:"updatedAt,omitempty"`
}

type AlertRuleScope struct {
	Type       string   `json:"type"`
	ProjectIDs []string `json:"projectIds,omitempty"`
}

type AlertRuleTriggers struct {
	Mode  string             `json:"mode"`
	Items []AlertRuleTrigger `json:"items,omitempty"`
}

type AlertRuleTrigger struct {
	Type   string  `json:"type"`
	Filter *string `json:"filter,omitempty"`
}

type AlertRuleNotificationSettings struct {
	EnableTeamOwnerNotifications bool    `json:"enableTeamOwnerNotifications"`
	IncidentIORoutingKey         *string `json:"incidentIoRoutingKey,omitempty"`
}

type AlertRuleCreate struct {
	Type                      string                         `json:"type"`
	Name                      string                         `json:"name"`
	RuleScope                 AlertRuleScope                 `json:"ruleScope"`
	Triggers                  *AlertRuleTriggers             `json:"triggers,omitempty"`
	MatchMinimumSeverityLevel *string                        `json:"matchMinimumSeverityLevel,omitempty"`
	NotificationSettings      *AlertRuleNotificationSettings `json:"notificationSettings,omitempty"`
}

type CreateAlertRuleRequest struct {
	TeamID string `json:"-"`
	AlertRuleCreate
}

type UpdateAlertRuleRequest struct {
	TeamID                    string                         `json:"-"`
	ID                        string                         `json:"-"`
	Type                      *string                        `json:"type,omitempty"`
	Name                      *string                        `json:"name,omitempty"`
	RuleScope                 *AlertRuleScope                `json:"ruleScope,omitempty"`
	Triggers                  *AlertRuleTriggers             `json:"triggers,omitempty"`
	MatchMinimumSeverityLevel *string                        `json:"matchMinimumSeverityLevel,omitempty"`
	NotificationSettings      *AlertRuleNotificationSettings `json:"notificationSettings,omitempty"`
}

type alertRuleEnvelope struct {
	Rule AlertRule `json:"rule"`
}

type alertRulesPage struct {
	Rules      []AlertRule `json:"rules"`
	Pagination struct {
		Next *string `json:"next"`
	} `json:"pagination"`
}

func (c *Client) alertRuleURL(teamID string, path string, query url.Values) string {
	u := fmt.Sprintf("%s/alerts/v3/alert-rules%s", c.baseURL, path)
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

func (c *Client) CreateAlertRule(ctx context.Context, request CreateAlertRuleRequest) (AlertRule, error) {
	u := c.alertRuleURL(request.TeamID, "", nil)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating alert rule", map[string]any{"url": u})

	var response alertRuleEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "POST", url: u, body: payload}, &response)
	return response.Rule, err
}

func (c *Client) GetAlertRule(ctx context.Context, id, teamID string) (AlertRule, error) {
	u := c.alertRuleURL(teamID, "/"+url.PathEscape(id), nil)
	tflog.Info(ctx, "getting alert rule", map[string]any{"url": u})

	var response alertRuleEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &response)
	return response.Rule, err
}

func (c *Client) UpdateAlertRule(ctx context.Context, request UpdateAlertRuleRequest) (AlertRule, error) {
	u := c.alertRuleURL(request.TeamID, "/"+url.PathEscape(request.ID), nil)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating alert rule", map[string]any{"url": u})

	var response alertRuleEnvelope
	err := c.doRequest(clientRequest{ctx: ctx, method: "PATCH", url: u, body: payload}, &response)
	return response.Rule, err
}

func (c *Client) DeleteAlertRule(ctx context.Context, id, teamID string) error {
	u := c.alertRuleURL(teamID, "/"+url.PathEscape(id), nil)
	tflog.Info(ctx, "deleting alert rule", map[string]any{"url": u})
	return c.doRequest(clientRequest{ctx: ctx, method: "DELETE", url: u}, nil)
}

func (c *Client) ListAlertRules(ctx context.Context, teamID string) ([]AlertRule, error) {
	var rules []AlertRule
	var cursor *string
	for {
		query := url.Values{"limit": []string{"100"}}
		if cursor != nil {
			query.Set("cursor", *cursor)
		}
		u := c.alertRuleURL(teamID, "", query)
		tflog.Info(ctx, "listing alert rules", map[string]any{"url": u})

		var page alertRulesPage
		if err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &page); err != nil {
			return nil, err
		}
		rules = append(rules, page.Rules...)
		cursor = page.Pagination.Next
		if cursor == nil {
			return rules, nil
		}
	}
}
