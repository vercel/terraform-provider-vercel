package client

import (
	"context"
	"fmt"
)

type FirewallBypassRule struct {
	Domain       string `json:"domain,omitempty"`
	SourceIp     string `json:"sourceIp"`
	ProjectScope bool   `json:"projectScope,omitempty"`
}

type FirewallBypass struct {
	OwnerId       string `json:"OwnerId"`
	Id            string `json:"Id"`
	Domain        string `json:"Domain"`
	Ip            string `json:"Ip"`
	IsProjectRule bool   `json:"IsProjectRule"`
}

func (c *Client) GetFirewallBypass(ctx context.Context, teamID, projectID string, request FirewallBypassRule) (a FirewallBypass, err error) {
	url := fmt.Sprintf("%s/v1/security/firewall/bypass?projectId=%s", c.baseURL, projectID)
	if tid := c.teamID(teamID); tid != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, tid)
	}
	url = fmt.Sprintf("%s&sourceIp=%s", url, request.SourceIp)
	if request.Domain == "*" {
		url = fmt.Sprintf("%s&projectScope=true", url)
	} else {
		url = fmt.Sprintf("%s&domain=%s", url, request.Domain)
	}

	var res struct {
		Result []FirewallBypass `json:"result"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &res)
	if err != nil || len(res.Result) == 0 {
		return FirewallBypass{}, err
	}
	return res.Result[0], err
}

func (c *Client) CreateFirewallBypass(ctx context.Context, teamID, projectID string, request FirewallBypassRule) (a FirewallBypass, err error) {
	url := fmt.Sprintf("%s/v1/security/firewall/bypass?projectId=%s", c.baseURL, projectID)
	if tid := c.teamID(teamID); tid != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, tid)
	}
	if request.Domain == "*" {
		request.Domain = ""
		request.ProjectScope = true
	}

	payload := string(mustMarshal(request))
	var res struct {
		Result []FirewallBypass `json:"result"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &res)
	if err != nil {
		return FirewallBypass{}, err
	}
	if len(res.Result) == 0 {
		return FirewallBypass{}, fmt.Errorf("no result returned")
	}
	return res.Result[0], err
}

func (c *Client) RemoveFirewallBypass(ctx context.Context, teamID, projectID string, request FirewallBypassRule) (a FirewallBypass, err error) {
	url := fmt.Sprintf("%s/v1/security/firewall/bypass?projectId=%s", c.baseURL, projectID)
	if tid := c.teamID(teamID); tid != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, tid)
	}

	payload := string(mustMarshal(request))
	var res FirewallBypass
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   payload,
	}, &res)
	if err != nil {
		return a, err
	}
	return FirewallBypass{}, err
}
