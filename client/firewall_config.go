package client

import (
	"context"
	"fmt"
)

type FirewallConfig struct {
	ProjectID       string                 `json:"-"`
	TeamID          string                 `json:"-"`
	Enabled         bool                   `json:"firewallEnabled"`
	ManagedRulesets map[string]ManagedRule `json:"managedRules,omitempty"`

	Rules   []FirewallRule         `json:"rules,omitempty"`
	IPRules []IPRule               `json:"ips,omitempty"`
	CRS     map[string]CoreRuleSet `json:"crs,omitempty"`
}
type ManagedRule struct {
	Active bool `json:"active"`
}

type FirewallRule struct {
	ID             string           `json:"id,omitempty"`
	Name           string           `json:"name"`
	Description    string           `json:"description,omitempty"`
	Active         bool             `json:"active"`
	ConditionGroup []ConditionGroup `json:"conditionGroup"`
	Action         Action           `json:"action"`
}

type ConditionGroup struct {
	Conditions []Condition `json:"conditions"`
}

type Condition struct {
	Type  string `json:"type"`
	Op    string `json:"op"`
	Neg   bool   `json:"neg"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Action struct {
	Mitigate Mitigate `json:"mitigate"`
}
type Mitigate struct {
	Action         string     `json:"action"`
	RateLimit      *RateLimit `json:"rateLimit,omitempty"`
	Redirect       *Redirect  `json:"redirect,omitempty"`
	ActionDuration string     `json:"actionDuration,omitempty"`
}

type RateLimit struct {
	Algo   string   `json:"algo"`
	Window int64    `json:"window"`
	Limit  int64    `json:"limit"`
	Keys   []string `json:"keys"`
	Action string   `json:"action"`
}

type Redirect struct {
	Location  string `json:"location"`
	Permanent bool   `json:"permanent"`
}

type IPRule struct {
	ID       string `json:"id,omitempty"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	Notes    string `json:"notes,omitempty"`
	Action   string `json:"action"`
}

type CoreRuleSet struct {
	Active bool   `json:"active"`
	Action string `json:"action"`
}

func (c *Client) GetFirewallConfig(ctx context.Context, projectId string, teamId string) (FirewallConfig, error) {
	teamId = c.teamID(teamId)
	url := fmt.Sprintf(
		"%s/v1/security/firewall/config/active?projectId=%s&teamId=%s",
		c.baseURL,
		projectId,
		teamId,
	)
	var res = FirewallConfig{}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &res)
	res.TeamID = teamId
	return res, err
}

func (c *Client) PutFirewallConfig(ctx context.Context, cfg FirewallConfig) (FirewallConfig, error) {
	teamId := c.teamID(cfg.TeamID)
	url := fmt.Sprintf(
		"%s/v1/security/firewall/config?projectId=%s&teamId=%s",
		c.baseURL,
		cfg.ProjectID,
		teamId,
	)

	var res struct {
		Active FirewallConfig    `json:"active"`
		Error  map[string]string `json:"error,omitempty"`
	}
	payload := mustMarshal(cfg)

	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   string(payload),
	}, &res)
	res.Active.TeamID = teamId
	return res.Active, err
}
