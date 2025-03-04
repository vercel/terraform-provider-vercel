package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type LogDrain struct {
	ID             string            `json:"id"`
	TeamID         string            `json:"ownerId"`
	DeliveryFormat string            `json:"deliveryFormat"`
	Environments   []string          `json:"environments"`
	Headers        map[string]string `json:"headers"`
	ProjectIDs     []string          `json:"projectIds"`
	SamplingRate   *float64          `json:"samplingRate"`
	Secret         string            `json:"secret"`
	Sources        []string          `json:"sources"`
	Endpoint       string            `json:"url"`
}

type CreateLogDrainRequest struct {
	TeamID         string            `json:"-"`
	DeliveryFormat string            `json:"deliveryFormat"`
	Environments   []string          `json:"environments"`
	Headers        map[string]string `json:"headers,omitempty"`
	ProjectIDs     []string          `json:"projectIds,omitempty"`
	SamplingRate   float64           `json:"samplingRate,omitempty"`
	Secret         string            `json:"secret,omitempty"`
	Sources        []string          `json:"sources"`
	Endpoint       string            `json:"url"`
}

func (c *Client) CreateLogDrain(ctx context.Context, request CreateLogDrainRequest) (l LogDrain, err error) {
	url := fmt.Sprintf("%s/v1/log-drains", c.baseURL)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating log drain", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &l)
	return l, err
}

func (c *Client) DeleteLogDrain(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/log-drains/%s", c.baseURL, id)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "deleting log drain", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

func (c *Client) GetLogDrain(ctx context.Context, id, teamID string) (l LogDrain, err error) {
	url := fmt.Sprintf("%s/v1/log-drains/%s", c.baseURL, id)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "reading log drain", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &l)
	return l, err
}

func (c *Client) GetEndpointVerificationCode(ctx context.Context, teamID string) (code string, err error) {
	url := fmt.Sprintf("%s/v1/verify-endpoint", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	var l struct {
		Code string `json:"verificationCode"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &l)
	return l.Code, err
}
