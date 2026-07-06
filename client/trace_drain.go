package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type TraceDrain struct {
	ID             string                   `json:"id"`
	TeamID         string                   `json:"ownerId"`
	Name           string                   `json:"name"`
	DeliveryFormat string                   `json:"deliveryFormat"`
	Headers        map[string]string        `json:"headers"`
	ProjectIDs     []string                 `json:"projectIds"`
	SamplingRules  []TraceDrainSamplingRule `json:"samplingRules"`
	Secret         string                   `json:"secret"`
	Endpoint       string                   `json:"url"`
}

type TraceDrainSamplingRule struct {
	Rate        float64 `json:"rate"`
	Environment string  `json:"env,omitempty"`
	RequestPath string  `json:"requestPath,omitempty"`
}

type CreateTraceDrainRequest struct {
	TeamID         string                   `json:"-"`
	Name           string                   `json:"name"`
	DeliveryFormat string                   `json:"deliveryFormat"`
	Headers        map[string]string        `json:"headers,omitempty"`
	ProjectIDs     []string                 `json:"projectIds,omitempty"`
	SamplingRules  []TraceDrainSamplingRule `json:"samplingRules,omitempty"`
	Secret         string                   `json:"secret,omitempty"`
	Endpoint       string                   `json:"url"`
}

type drainsTraceCreateRequest struct {
	Name       string                         `json:"name"`
	Projects   string                         `json:"projects"`
	ProjectIDs []string                       `json:"projectIds,omitempty"`
	Schemas    map[string]drainsSchemaVersion `json:"schemas"`
	Delivery   drainsDeliveryOTLPHTTP         `json:"delivery"`
	Sampling   []drainsSampling               `json:"sampling,omitempty"`
	Source     drainsSource                   `json:"source"`
}

type drainsDeliveryOTLPHTTP struct {
	Type     string             `json:"type"`
	Endpoint drainsOTLPEndpoint `json:"endpoint"`
	Encoding string             `json:"encoding"`
	Headers  map[string]string  `json:"headers"`
	Secret   string             `json:"secret,omitempty"`
}

type drainsOTLPEndpoint struct {
	Traces string `json:"traces"`
}

type traceDrainsResponse struct {
	ID         string   `json:"id"`
	OwnerID    string   `json:"ownerId"`
	Name       string   `json:"name"`
	ProjectIDs []string `json:"projectIds"`

	Delivery struct {
		Type     string             `json:"type"`
		Endpoint drainsOTLPEndpoint `json:"endpoint"`
		Encoding string             `json:"encoding"`
		Headers  map[string]string  `json:"headers"`
		Secret   string             `json:"secret"`
	} `json:"delivery"`

	Sampling []drainsSampling `json:"sampling"`
}

func drainsRespToTraceDrain(r traceDrainsResponse) TraceDrain {
	samplingRules := make([]TraceDrainSamplingRule, 0, len(r.Sampling))
	for _, rule := range r.Sampling {
		samplingRules = append(samplingRules, TraceDrainSamplingRule{
			Rate:        rule.Rate,
			Environment: rule.Env,
			RequestPath: rule.RequestPath,
		})
	}

	return TraceDrain{
		ID:             r.ID,
		TeamID:         r.OwnerID,
		Name:           r.Name,
		DeliveryFormat: r.Delivery.Encoding,
		Headers:        r.Delivery.Headers,
		ProjectIDs:     r.ProjectIDs,
		SamplingRules:  samplingRules,
		Secret:         r.Delivery.Secret,
		Endpoint:       r.Delivery.Endpoint.Traces,
	}
}

func (c *Client) CreateTraceDrain(ctx context.Context, request CreateTraceDrainRequest) (l TraceDrain, err error) {
	url := fmt.Sprintf("%s/v1/drains", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	projects := "all"
	if len(request.ProjectIDs) > 0 {
		projects = "some"
	}

	hdrs := request.Headers
	if hdrs == nil {
		hdrs = map[string]string{}
	}

	sampling := make([]drainsSampling, 0, len(request.SamplingRules))
	for _, rule := range request.SamplingRules {
		sampling = append(sampling, drainsSampling{
			Type:        "head_sampling",
			Rate:        rule.Rate,
			Env:         rule.Environment,
			RequestPath: rule.RequestPath,
		})
	}

	payload := drainsTraceCreateRequest{
		Name:       request.Name,
		Projects:   projects,
		ProjectIDs: request.ProjectIDs,
		Schemas: map[string]drainsSchemaVersion{
			"trace": {Version: "v1"},
		},
		Delivery: drainsDeliveryOTLPHTTP{
			Type:     "otlphttp",
			Endpoint: drainsOTLPEndpoint{Traces: request.Endpoint},
			Encoding: request.DeliveryFormat,
			Headers:  hdrs,
			Secret:   request.Secret,
		},
		Source: drainsSource{Kind: "self-served"},
	}
	if len(sampling) > 0 {
		payload.Sampling = sampling
	}

	tflog.Info(ctx, "creating trace drain", map[string]any{
		"url": url,
	})

	var resp traceDrainsResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(payload)),
	}, &resp)
	if err != nil {
		return l, err
	}
	return drainsRespToTraceDrain(resp), nil
}

func (c *Client) DeleteTraceDrain(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting trace drain", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

func (c *Client) GetTraceDrain(ctx context.Context, id, teamID string) (l TraceDrain, err error) {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "reading trace drain", map[string]any{
		"url": url,
	})
	var resp traceDrainsResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &resp)
	if err != nil {
		return l, err
	}
	return drainsRespToTraceDrain(resp), nil
}
