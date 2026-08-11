package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Alert rule types supported by the Vercel alerts API.
const (
	AlertRuleTypeUsageAnomaly = "usage_anomaly"
	AlertRuleTypeErrorAnomaly = "error_anomaly"
	AlertRuleTypeCustomAlert  = "custom_alert"
)

// AlertRule represents a Vercel alert rule. Alert rules decide which anomalies
// and custom Observability thresholds raise an alert, and who is subscribed to
// the alerts that are raised.
type AlertRule struct {
	ID     string
	TeamID string
	Name   string
	// ProjectID holds an OData project filter such as `projectId in ('prj_123')`
	// for built-in anomaly rules, and a raw project ID for custom alert rules.
	// It is nil for team-wide rules.
	ProjectID                  *string
	AlertTypes                 []AlertRuleType
	SensitivityLevel           *int64
	AutosubscribeOwners        *bool
	AutosubscribeProjectAdmins *bool
	CustomAlert                *CustomAlert
	IsDefault                  bool
}

// AlertRuleType enables one class of alert on a rule, optionally narrowed by an
// OData filter.
type AlertRuleType struct {
	Type   string  `json:"type"`
	Filter *string `json:"filter,omitempty"`
}

// CustomAlert describes a custom Observability metric alert. The Vercel API
// transports the Observability query as an escaped JSON string, but callers
// work with the decoded CustomAlertQuery instead.
type CustomAlert struct {
	Query            CustomAlertQuery
	TriggerType      string
	TriggerOperator  string
	TriggerThreshold float64
	MinThreshold     *float64
	Formula          *CustomAlertFormula
}

// CustomAlertQuery is the Observability query a custom alert evaluates.
type CustomAlertQuery struct {
	Scope       *CustomAlertQueryScope       `json:"scope,omitempty"`
	Event       string                       `json:"event"`
	Rollups     map[string]CustomAlertRollup `json:"rollups"`
	GroupBy     []string                     `json:"groupBy,omitempty"`
	Filter      *string                      `json:"filter,omitempty"`
	Granularity *CustomAlertGranularity      `json:"granularity,omitempty"`
}

// CustomAlertQueryScope limits a custom alert query to a team and project. It is
// derived from the alert rule's team and project, so callers do not set it.
type CustomAlertQueryScope struct {
	Type       string   `json:"type"`
	OwnerID    string   `json:"ownerId"`
	ProjectIDs []string `json:"projectIds"`
}

// CustomAlertRollup aggregates a single measure within a custom alert query.
type CustomAlertRollup struct {
	Measure     string  `json:"measure"`
	Aggregation string  `json:"aggregation"`
	Filter      *string `json:"filter,omitempty"`
}

// CustomAlertGranularity is the window a custom alert query is bucketed into.
type CustomAlertGranularity struct {
	Minutes *int64 `json:"minutes,omitempty"`
	Hours   *int64 `json:"hours,omitempty"`
	Days    *int64 `json:"days,omitempty"`
}

// CustomAlertFormula combines two named rollups into a single value, such as an
// error rate.
type CustomAlertFormula struct {
	Operator string `json:"operator"`
	Left     string `json:"left"`
	Right    string `json:"right"`
}

// IsCustomAlert reports whether the rule is a custom Observability metric alert
// rather than a built-in anomaly rule.
func (a AlertRule) IsCustomAlert() bool {
	if a.CustomAlert != nil {
		return true
	}
	for _, alertType := range a.AlertTypes {
		if alertType.Type == AlertRuleTypeCustomAlert {
			return true
		}
	}
	return false
}

type customAlertPayload struct {
	QueryJSONString  string              `json:"queryJsonString"`
	TriggerType      string              `json:"triggerType"`
	TriggerOperator  string              `json:"triggerOperator"`
	TriggerThreshold float64             `json:"triggerThreshold"`
	MinThreshold     *float64            `json:"minThreshold,omitempty"`
	Formula          *CustomAlertFormula `json:"formula,omitempty"`
}

type alertRuleResponse struct {
	ID                                string              `json:"id"`
	TeamID                            string              `json:"teamId"`
	Name                              string              `json:"name"`
	ProjectID                         *string             `json:"projectId"`
	AlertTypes                        []AlertRuleType     `json:"alertTypes"`
	SensitivityLevel                  *int64              `json:"sensitivityLevel"`
	AutosubscribeOwnersInKnock        *bool               `json:"autosubscribeOwnersInKnock"`
	AutosubscribeProjectAdminsInKnock *bool               `json:"autosubscribeProjectAdminsInKnock"`
	CustomAlert                       *customAlertPayload `json:"customAlert"`
	IsDefault                         bool                `json:"isDefault"`
}

// alertRuleCreatePayload omits unset optional fields so that the API applies its
// own defaults.
type alertRuleCreatePayload struct {
	Name                              string              `json:"name"`
	ProjectID                         *string             `json:"projectId,omitempty"`
	AlertTypes                        []AlertRuleType     `json:"alertTypes"`
	SensitivityLevel                  *int64              `json:"sensitivityLevel,omitempty"`
	AutosubscribeOwnersInKnock        *bool               `json:"autosubscribeOwnersInKnock,omitempty"`
	AutosubscribeProjectAdminsInKnock *bool               `json:"autosubscribeProjectAdminsInKnock,omitempty"`
	CustomAlert                       *customAlertPayload `json:"customAlert,omitempty"`
}

// alertRuleUpdatePayload sends unset optional fields as explicit nulls, which is
// how the API clears them. customAlert is the exception: the API rejects a null
// customAlert, so it is omitted when unset. Toggling a rule between built-in and
// custom therefore has to replace the rule rather than update it.
type alertRuleUpdatePayload struct {
	Name                              string              `json:"name"`
	ProjectID                         *string             `json:"projectId"`
	AlertTypes                        []AlertRuleType     `json:"alertTypes"`
	SensitivityLevel                  *int64              `json:"sensitivityLevel"`
	AutosubscribeOwnersInKnock        *bool               `json:"autosubscribeOwnersInKnock"`
	AutosubscribeProjectAdminsInKnock *bool               `json:"autosubscribeProjectAdminsInKnock"`
	CustomAlert                       *customAlertPayload `json:"customAlert,omitempty"`
}

// CreateAlertRuleRequest creates a new alert rule.
type CreateAlertRuleRequest struct {
	TeamID                     string
	Name                       string
	ProjectID                  *string
	AlertTypes                 []AlertRuleType
	SensitivityLevel           *int64
	AutosubscribeOwners        *bool
	AutosubscribeProjectAdmins *bool
	CustomAlert                *CustomAlert
}

// UpdateAlertRuleRequest replaces the managed fields of an existing alert rule.
type UpdateAlertRuleRequest struct {
	TeamID                     string
	ID                         string
	Name                       string
	ProjectID                  *string
	AlertTypes                 []AlertRuleType
	SensitivityLevel           *int64
	AutosubscribeOwners        *bool
	AutosubscribeProjectAdmins *bool
	CustomAlert                *CustomAlert
}

// customAlertToPayload encodes a custom alert for the wire, deriving the query
// scope from the rule's team and project when the caller has not set one. The
// API rejects custom alert queries that have no scope.
func customAlertToPayload(customAlert *CustomAlert, teamID string, projectID *string) (*customAlertPayload, error) {
	if customAlert == nil {
		return nil, nil
	}

	query := customAlert.Query
	if query.Scope == nil {
		if teamID == "" || projectID == nil || *projectID == "" {
			return nil, fmt.Errorf("custom alert rules must target a team and a project")
		}
		query.Scope = &CustomAlertQueryScope{
			Type:       "project",
			OwnerID:    teamID,
			ProjectIDs: []string{*projectID},
		}
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("encoding custom alert query: %w", err)
	}

	return &customAlertPayload{
		QueryJSONString:  string(queryJSON),
		TriggerType:      customAlert.TriggerType,
		TriggerOperator:  customAlert.TriggerOperator,
		TriggerThreshold: customAlert.TriggerThreshold,
		MinThreshold:     customAlert.MinThreshold,
		Formula:          customAlert.Formula,
	}, nil
}

func customAlertFromPayload(payload *customAlertPayload) (*CustomAlert, error) {
	if payload == nil {
		return nil, nil
	}

	var query CustomAlertQuery
	if payload.QueryJSONString != "" {
		if err := json.Unmarshal([]byte(payload.QueryJSONString), &query); err != nil {
			return nil, fmt.Errorf("decoding custom alert query: %w", err)
		}
	}

	return &CustomAlert{
		Query:            query,
		TriggerType:      payload.TriggerType,
		TriggerOperator:  payload.TriggerOperator,
		TriggerThreshold: payload.TriggerThreshold,
		MinThreshold:     payload.MinThreshold,
		Formula:          payload.Formula,
	}, nil
}

func alertRuleFromResponse(response alertRuleResponse) (AlertRule, error) {
	customAlert, err := customAlertFromPayload(response.CustomAlert)
	if err != nil {
		return AlertRule{}, fmt.Errorf("reading alert rule %q: %w", response.ID, err)
	}

	return AlertRule{
		ID:                         response.ID,
		TeamID:                     response.TeamID,
		Name:                       response.Name,
		ProjectID:                  response.ProjectID,
		AlertTypes:                 response.AlertTypes,
		SensitivityLevel:           response.SensitivityLevel,
		AutosubscribeOwners:        response.AutosubscribeOwnersInKnock,
		AutosubscribeProjectAdmins: response.AutosubscribeProjectAdminsInKnock,
		CustomAlert:                customAlert,
		IsDefault:                  response.IsDefault,
	}, nil
}

func (c *Client) alertRulesURL(teamID string, extraQuery url.Values) string {
	query := url.Values{}
	if c.TeamID(teamID) != "" {
		query.Set("teamId", c.TeamID(teamID))
	}
	for key, values := range extraQuery {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	return urlWithQuery(fmt.Sprintf("%s/alerts/v2/alert-rules", c.baseURL), query)
}

func (c *Client) alertRuleURL(id, teamID string) string {
	query := url.Values{}
	if c.TeamID(teamID) != "" {
		query.Set("teamId", c.TeamID(teamID))
	}
	return urlWithQuery(fmt.Sprintf("%s/alerts/v2/alert-rules/%s", c.baseURL, url.PathEscape(id)), query)
}

// CreateAlertRule creates an alert rule for a team.
func (c *Client) CreateAlertRule(ctx context.Context, request CreateAlertRuleRequest) (a AlertRule, err error) {
	customAlert, err := customAlertToPayload(request.CustomAlert, c.TeamID(request.TeamID), request.ProjectID)
	if err != nil {
		return a, err
	}

	alertTypes := request.AlertTypes
	if alertTypes == nil {
		alertTypes = []AlertRuleType{}
	}

	url := c.alertRulesURL(request.TeamID, nil)
	payload := alertRuleCreatePayload{
		Name:                              request.Name,
		ProjectID:                         request.ProjectID,
		AlertTypes:                        alertTypes,
		SensitivityLevel:                  request.SensitivityLevel,
		AutosubscribeOwnersInKnock:        request.AutosubscribeOwners,
		AutosubscribeProjectAdminsInKnock: request.AutosubscribeProjectAdmins,
		CustomAlert:                       customAlert,
	}

	tflog.Info(ctx, "creating alert rule", map[string]any{
		"url": url,
	})
	var response alertRuleResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(payload)),
	}, &response)
	if err != nil {
		return a, err
	}
	return alertRuleFromResponse(response)
}

// GetAlertRule reads a single alert rule by ID.
func (c *Client) GetAlertRule(ctx context.Context, id, teamID string) (a AlertRule, err error) {
	url := c.alertRuleURL(id, teamID)
	tflog.Info(ctx, "reading alert rule", map[string]any{
		"url": url,
	})
	var response alertRuleResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return a, err
	}
	return alertRuleFromResponse(response)
}

// UpdateAlertRule replaces the managed fields of an existing alert rule.
func (c *Client) UpdateAlertRule(ctx context.Context, request UpdateAlertRuleRequest) (a AlertRule, err error) {
	customAlert, err := customAlertToPayload(request.CustomAlert, c.TeamID(request.TeamID), request.ProjectID)
	if err != nil {
		return a, err
	}

	alertTypes := request.AlertTypes
	if alertTypes == nil {
		alertTypes = []AlertRuleType{}
	}

	url := c.alertRuleURL(request.ID, request.TeamID)
	payload := alertRuleUpdatePayload{
		Name:                              request.Name,
		ProjectID:                         request.ProjectID,
		AlertTypes:                        alertTypes,
		SensitivityLevel:                  request.SensitivityLevel,
		AutosubscribeOwnersInKnock:        request.AutosubscribeOwners,
		AutosubscribeProjectAdminsInKnock: request.AutosubscribeProjectAdmins,
		CustomAlert:                       customAlert,
	}

	tflog.Info(ctx, "updating alert rule", map[string]any{
		"url": url,
	})
	var response alertRuleResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   string(mustMarshal(payload)),
	}, &response)
	if err != nil {
		return a, err
	}
	return alertRuleFromResponse(response)
}

// DeleteAlertRule removes an alert rule.
func (c *Client) DeleteAlertRule(ctx context.Context, id, teamID string) error {
	url := c.alertRuleURL(id, teamID)
	tflog.Info(ctx, "deleting alert rule", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

// ListAlertRules lists the alert rules for a team, optionally narrowed to a
// single project.
func (c *Client) ListAlertRules(ctx context.Context, teamID, projectID string) ([]AlertRule, error) {
	extraQuery := url.Values{}
	if projectID != "" {
		extraQuery.Set("projectId", projectID)
	}

	url := c.alertRulesURL(teamID, extraQuery)
	tflog.Info(ctx, "listing alert rules", map[string]any{
		"url": url,
	})
	var response []alertRuleResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return nil, err
	}

	alertRules := make([]AlertRule, 0, len(response))
	for _, item := range response {
		alertRule, err := alertRuleFromResponse(item)
		if err != nil {
			return nil, err
		}
		alertRules = append(alertRules, alertRule)
	}
	return alertRules, nil
}
