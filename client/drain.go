package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Drain struct {
	ID         string            `json:"id"`
	OwnerID    string            `json:"ownerId"`
	Name       string            `json:"name"`
	Projects   string            `json:"projects"` // "some" or "all"
	ProjectIds []string          `json:"projectIds"`
	Schemas    map[string]any    `json:"schemas"`
	Delivery   DeliveryConfig    `json:"delivery"`
	Sampling   []SamplingConfig  `json:"sampling,omitempty"`
	TeamID     string            `json:"teamId"`
	Status     string            `json:"status"`
	Filter     *string           `json:"filter,omitempty"`
	Transforms []TransformConfig `json:"transforms,omitempty"`
}

type OTLPDeliveryEndpoint struct {
	Traces string `json:"traces"`
}

type DeliveryConfig struct {
	Type        string            `json:"type"`
	Endpoint    any               `json:"endpoint"` // Can be string or object for different delivery types
	Encoding    string            `json:"encoding"`
	Compression *string           `json:"compression,omitempty"`
	Headers     map[string]string `json:"headers"`
	Secret      *string           `json:"secret,omitempty"`
}

type SamplingConfig struct {
	Type        string  `json:"type"`
	Rate        float64 `json:"rate"` // Must be between 0 and 1
	Env         *string `json:"env,omitempty"`
	RequestPath *string `json:"requestPath,omitempty"`
}

type TransformConfig struct {
	ID string `json:"id"`
}

type SchemaConfig struct {
	Version string `json:"version"`
}

type CreateDrainRequest struct {
	TeamID     string                  `json:"-"`
	Name       string                  `json:"name"`
	Projects   string                  `json:"projects"` // "some" or "all"
	ProjectIds []string                `json:"projectIds,omitempty"`
	Filter     *string                 `json:"filter,omitempty"`
	Schemas    map[string]SchemaConfig `json:"schemas"`
	Delivery   DeliveryConfig          `json:"delivery"`
	Sampling   []SamplingConfig        `json:"sampling,omitempty"`
	Transforms []TransformConfig       `json:"transforms,omitempty"`
}

type UpdateDrainRequest struct {
	TeamID     string                  `json:"-"`
	Name       *string                 `json:"name,omitempty"`
	Projects   *string                 `json:"projects,omitempty"`
	ProjectIds []string                `json:"projectIds,omitempty"`
	Filter     *string                 `json:"filter,omitempty"`
	Schemas    map[string]SchemaConfig `json:"schemas,omitempty"`
	Delivery   *DeliveryConfig         `json:"delivery,omitempty"`
	Sampling   []SamplingConfig        `json:"sampling,omitempty"`
	Transforms []TransformConfig       `json:"transforms,omitempty"`
	Status     *string                 `json:"status,omitempty"` // "enabled" or "disabled"
}

type ListDrainsResponse struct {
	Drains []Drain `json:"drains"`
}

func (c *Client) CreateDrain(ctx context.Context, request CreateDrainRequest) (d Drain, err error) {
	url := fmt.Sprintf("%s/v1/drains", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating drain", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &d)
	return d, err
}

func (c *Client) GetDrain(ctx context.Context, id, teamID string) (d Drain, err error) {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "reading drain", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &d)
	return d, err
}

func (c *Client) UpdateDrain(ctx context.Context, id string, request UpdateDrainRequest) (d Drain, err error) {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating drain", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &d)
	return d, err
}

func (c *Client) DeleteDrain(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting drain", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

func (c *Client) ListDrains(ctx context.Context, teamID string) (response ListDrainsResponse, err error) {
	url := fmt.Sprintf("%s/v1/drains", c.baseURL)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "listing drains", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	return response, err
}
