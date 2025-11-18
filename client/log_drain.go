package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// LogDrain represents a simplified drain shape used by the provider
// regardless of the underlying API representation.
type LogDrain struct {
	ID             string            `json:"id"`
	TeamID         string            `json:"ownerId"`
	Name           string            `json:"name"`
	DeliveryFormat string            `json:"deliveryFormat"`
	Environments   []string          `json:"environments"`
	Headers        map[string]string `json:"headers"`
	ProjectIDs     []string          `json:"projectIds"`
	SamplingRate   *float64          `json:"samplingRate"`
	Secret         string            `json:"secret"`
	Sources        []string          `json:"sources"`
	Endpoint       string            `json:"url"`
}

// CreateLogDrainRequest is the provider-level request used to create drains.
// It maps to the new /v1/drains API payload internally.
type CreateLogDrainRequest struct {
	TeamID         string            `json:"-"`
	Name           string            `json:"name"`
	DeliveryFormat string            `json:"deliveryFormat"`
	Environments   []string          `json:"environments"`
	Headers        map[string]string `json:"headers,omitempty"`
	ProjectIDs     []string          `json:"projectIds,omitempty"`
	SamplingRate   float64           `json:"samplingRate,omitempty"`
	Secret         string            `json:"secret,omitempty"`
	Sources        []string          `json:"sources"`
	Endpoint       string            `json:"url"`
}

// Internal request/response shapes for /v1/drains
type drainsCreateRequest struct {
	Name       string                         `json:"name"`
	Projects   string                         `json:"projects"` // "some" | "all"
	ProjectIDs []string                       `json:"projectIds,omitempty"`
	Filter     drainsFilterV2Request          `json:"filter"`
	Schemas    map[string]drainsSchemaVersion `json:"schemas"`
	Delivery   drainsDeliveryHTTP             `json:"delivery"`
	Sampling   []drainsSampling               `json:"sampling,omitempty"`
	Source     drainsSource                   `json:"source"`
}

type drainsSchemaVersion struct {
	Version string `json:"version"`
}

type drainsSource struct {
	Kind string `json:"kind"` // "self-served" or "integration"
}

type drainsFilterV2Request struct {
	Version string               `json:"version"` // e.g. "v2"
	Filter  drainsBasicFilterReq `json:"filter"`
}

type drainsBasicFilterReq struct {
	Type       string                  `json:"type"` // "basic"
	Log        *drainsLogFilterReq     `json:"log,omitempty"`
	Deployment *drainsDeploymentFilter `json:"deployment,omitempty"`
}

type drainsLogFilterReq struct {
	Sources []string `json:"sources,omitempty"`
}

type drainsDeploymentFilter struct {
	Environments []string `json:"environments,omitempty"`
}

type drainsDeliveryHTTP struct {
	Type     string            `json:"type"` // "http"
	Endpoint string            `json:"endpoint"`
	Encoding string            `json:"encoding"` // json | ndjson
	Headers  map[string]string `json:"headers"`
	Secret   string            `json:"secret,omitempty"`
}

type drainsSampling struct {
	Type        string  `json:"type"` // "head_sampling"
	Rate        float64 `json:"rate"`
	Env         string  `json:"env,omitempty"`
	RequestPath string  `json:"requestPath,omitempty"`
}

type drainsResponse struct {
	ID         string   `json:"id"`
	OwnerID    string   `json:"ownerId"`
	Name       string   `json:"name"`
	ProjectIDs []string `json:"projectIds"`

	Delivery struct {
		Type     string            `json:"type"`
		Endpoint string            `json:"endpoint"`
		Encoding string            `json:"encoding"`
		Headers  map[string]string `json:"headers"`
		Secret   string            `json:"secret"`
	} `json:"delivery"`

	FilterV2 struct {
		Version string `json:"version"`
		Filter  struct {
			Type string `json:"type"`
			Log  *struct {
				Sources []string `json:"sources"`
			} `json:"log,omitempty"`
			Deployment *struct {
				Environments []string `json:"environments"`
			} `json:"deployment,omitempty"`
		} `json:"filter"`
	} `json:"filterV2"`

	Sampling []drainsSampling `json:"sampling"`
}

func drainsRespToLogDrain(r drainsResponse) LogDrain {
	var sampling *float64
	if len(r.Sampling) > 0 {
		// Use the first sampling rule's rate if present
		rate := r.Sampling[0].Rate
		sampling = &rate
	}

	var environments []string
	if r.FilterV2.Filter.Deployment != nil {
		environments = r.FilterV2.Filter.Deployment.Environments
	}
	var sources []string
	if r.FilterV2.Filter.Log != nil {
		sources = r.FilterV2.Filter.Log.Sources
	}

	return LogDrain{
		ID:             r.ID,
		TeamID:         r.OwnerID,
		Name:           r.Name,
		DeliveryFormat: r.Delivery.Encoding,
		Environments:   environments,
		Headers:        r.Delivery.Headers,
		ProjectIDs:     r.ProjectIDs,
		SamplingRate:   sampling,
		Secret:         r.Delivery.Secret,
		Sources:        sources,
		Endpoint:       r.Delivery.Endpoint,
	}
}

func (c *Client) CreateLogDrain(ctx context.Context, request CreateLogDrainRequest) (l LogDrain, err error) {
	url := fmt.Sprintf("%s/v1/drains", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	projects := "all"
	if len(request.ProjectIDs) > 0 {
		projects = "some"
	}

	// ensure headers is a non-nil object per API contract
	hdrs := request.Headers
	if hdrs == nil {
		hdrs = map[string]string{}
	}

	payload := drainsCreateRequest{
		Name:       request.Name,
		Projects:   projects,
		ProjectIDs: request.ProjectIDs,
		Filter: drainsFilterV2Request{
			Version: "v2",
			Filter: drainsBasicFilterReq{
				Type: "basic",
				Log: &drainsLogFilterReq{
					Sources: request.Sources,
				},
				Deployment: &drainsDeploymentFilter{
					Environments: request.Environments,
				},
			},
		},
		Schemas: map[string]drainsSchemaVersion{
			"log": {Version: "v1"},
		},
		Delivery: drainsDeliveryHTTP{
			Type:     "http",
			Endpoint: request.Endpoint,
			Encoding: request.DeliveryFormat,
			Headers:  hdrs,
			Secret:   request.Secret,
		},
		Source: drainsSource{Kind: "self-served"},
	}
	if request.SamplingRate > 0 {
		payload.Sampling = []drainsSampling{{Type: "head_sampling", Rate: request.SamplingRate}}
	}

	body := string(mustMarshal(payload))
	tflog.Info(ctx, "creating drain", map[string]any{
		"url":     url,
		"payload": body,
	})

	var resp drainsResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   body,
	}, &resp)
	if err != nil {
		return l, err
	}
	return drainsRespToLogDrain(resp), nil
}

func (c *Client) DeleteLogDrain(ctx context.Context, id, teamID string) error {
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

func (c *Client) GetLogDrain(ctx context.Context, id, teamID string) (l LogDrain, err error) {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "reading drain", map[string]any{
		"url": url,
	})
	var resp drainsResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &resp)
	if err != nil {
		return l, err
	}
	return drainsRespToLogDrain(resp), nil
}

func (c *Client) GetEndpointVerificationCode(ctx context.Context, teamID string) (code string, err error) {
	url := fmt.Sprintf("%s/v1/verify-endpoint", c.baseURL)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
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
