package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	AlertRuleTypeBuiltIn = "built-in"
	AlertRuleTypeCustom  = "custom"
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
	Severity                  *string                       `json:"severity,omitempty"`
	Evaluation                *AlertRuleEvaluation          `json:"evaluation"`
	Trigger                   *AlertRuleCustomTrigger       `json:"trigger,omitempty"`
	NotificationSettings      AlertRuleNotificationSettings `json:"notificationSettings"`
	IsDefault                 bool                          `json:"isDefault"`
	QuerySupported            *bool                         `json:"querySupported,omitempty"`
	CreatedAt                 *int64                        `json:"createdAt,omitempty"`
	UpdatedAt                 *int64                        `json:"updatedAt,omitempty"`
}

type AlertRuleScope struct {
	Type       string   `json:"type"`
	ProjectID  *string  `json:"projectId,omitempty"`
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

type AlertRuleEvaluation struct {
	Window string               `json:"window"`
	Query  AlertRuleCustomQuery `json:"query"`
}

type AlertRuleCustomQuery struct {
	GroupBy  []string                            `json:"groupBy,omitempty"`
	Filter   *string                             `json:"filter,omitempty"`
	Metrics  map[string]AlertRuleMetricSelection `json:"metrics"`
	Formulas map[string]string                   `json:"formulas,omitempty"`
	Outputs  []string                            `json:"outputs"`
}

type AlertRuleMetricSelection struct {
	Metric      string   `json:"metric"`
	Aggregation string   `json:"aggregation"`
	Per         *string  `json:"per,omitempty"`
	Normalize   *string  `json:"normalize,omitempty"`
	Dimensions  []string `json:"dimensions,omitempty"`
	Filter      *string  `json:"filter,omitempty"`
}

type AlertRuleCustomTrigger struct {
	Type               string                   `json:"type"`
	Output             string                   `json:"output"`
	Operator           *string                  `json:"operator,omitempty"`
	Threshold          *float64                 `json:"threshold,omitempty"`
	StandardDeviations *float64                 `json:"standardDeviations,omitempty"`
	Minimum            *AlertRuleTriggerMinimum `json:"minimum,omitempty"`
}

type AlertRuleTriggerMinimum struct {
	Output    string  `json:"output"`
	Threshold float64 `json:"threshold"`
}

type AlertRuleCreate struct {
	Type                      string                         `json:"type"`
	Name                      string                         `json:"name"`
	RuleScope                 AlertRuleScope                 `json:"ruleScope"`
	Triggers                  *AlertRuleTriggers             `json:"triggers,omitempty"`
	MatchMinimumSeverityLevel *string                        `json:"matchMinimumSeverityLevel,omitempty"`
	Severity                  *string                        `json:"severity,omitempty"`
	Evaluation                *AlertRuleEvaluation           `json:"evaluation,omitempty"`
	Trigger                   *AlertRuleCustomTrigger        `json:"trigger,omitempty"`
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
	Severity                  *string                        `json:"severity,omitempty"`
	Evaluation                *AlertRuleEvaluation           `json:"evaluation,omitempty"`
	Trigger                   *AlertRuleCustomTrigger        `json:"trigger,omitempty"`
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
