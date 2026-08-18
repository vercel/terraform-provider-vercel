package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AuditLogDrain represents a team-wide Audit Log Drain.
type AuditLogDrain struct {
	ID     string `json:"id"`
	TeamID string `json:"ownerId"`
	Name   string `json:"name"`
	HTTP   *AuditLogDrainHTTPDelivery
	S3     *AuditLogDrainS3Delivery
}

// AuditLogDrainHTTPDelivery configures delivery to a custom HTTP endpoint.
type AuditLogDrainHTTPDelivery struct {
	Endpoint    string            `json:"endpoint"`
	Encoding    string            `json:"encoding"`
	Compression *string           `json:"compression,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Secret      *string           `json:"secret,omitempty"`
}

// AuditLogDrainS3Delivery configures delivery to an Amazon S3 bucket.
type AuditLogDrainS3Delivery struct {
	Endpoint             string  `json:"endpoint"`
	Encoding             string  `json:"encoding"`
	RoleARN              string  `json:"roleArn"`
	Region               string  `json:"region"`
	ServerSideEncryption *string `json:"serverSideEncryption,omitempty"`
	ObjectACL            *string `json:"objectAcl,omitempty"`
}

// CreateAuditLogDrainRequest is the provider-level request used to create an Audit Log Drain.
type CreateAuditLogDrainRequest struct {
	TeamID string
	Name   string
	HTTP   *AuditLogDrainHTTPDelivery
	S3     *AuditLogDrainS3Delivery
}

type auditLogDrainsCreateRequest struct {
	Name     string                         `json:"name"`
	Projects string                         `json:"projects"`
	Schemas  map[string]drainsSchemaVersion `json:"schemas"`
	Delivery any                            `json:"delivery"`
	Source   drainsSource                   `json:"source"`
}

type auditLogDrainsDeliveryHTTP struct {
	Type        string            `json:"type"`
	Endpoint    string            `json:"endpoint"`
	Encoding    string            `json:"encoding"`
	Compression *string           `json:"compression,omitempty"`
	Headers     map[string]string `json:"headers"`
	Secret      *string           `json:"secret,omitempty"`
}

type auditLogDrainsDeliveryS3 struct {
	Type                 string  `json:"type"`
	Endpoint             string  `json:"endpoint"`
	Encoding             string  `json:"encoding"`
	Compression          string  `json:"compression"`
	FileStructure        string  `json:"fileStructure"`
	RoleARN              string  `json:"roleArn"`
	Region               string  `json:"region"`
	ServerSideEncryption *string `json:"serverSideEncryption,omitempty"`
	ObjectACL            *string `json:"objectAcl,omitempty"`
}

type auditLogDrainsResponse struct {
	ID       string                                 `json:"id"`
	OwnerID  string                                 `json:"ownerId"`
	Name     string                                 `json:"name"`
	Schemas  map[string]auditLogDrainResponseSchema `json:"schemas"`
	Delivery json.RawMessage                        `json:"delivery"`
}

type auditLogDrainResponseSchema struct {
	Version string `json:"version"`
}

func auditLogDrainFromResponse(response auditLogDrainsResponse) (AuditLogDrain, error) {
	schema, ok := response.Schemas["audit_log"]
	if !ok {
		return AuditLogDrain{}, fmt.Errorf("drain %q is not an Audit Log Drain: audit_log schema is missing", response.ID)
	}
	if schema.Version != "" && schema.Version != "v1" {
		return AuditLogDrain{}, fmt.Errorf("drain %q uses unsupported audit_log schema version %q", response.ID, schema.Version)
	}

	var deliveryType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(response.Delivery, &deliveryType); err != nil {
		return AuditLogDrain{}, fmt.Errorf("decoding Audit Log Drain %q delivery type: %w", response.ID, err)
	}

	drain := AuditLogDrain{
		ID:     response.ID,
		TeamID: response.OwnerID,
		Name:   response.Name,
	}
	switch deliveryType.Type {
	case "http":
		var delivery auditLogDrainsDeliveryHTTP
		if err := json.Unmarshal(response.Delivery, &delivery); err != nil {
			return AuditLogDrain{}, fmt.Errorf("decoding Audit Log Drain %q HTTP delivery: %w", response.ID, err)
		}
		drain.HTTP = &AuditLogDrainHTTPDelivery{
			Endpoint:    delivery.Endpoint,
			Encoding:    delivery.Encoding,
			Compression: delivery.Compression,
			Headers:     delivery.Headers,
			Secret:      delivery.Secret,
		}
	case "s3":
		var delivery auditLogDrainsDeliveryS3
		if err := json.Unmarshal(response.Delivery, &delivery); err != nil {
			return AuditLogDrain{}, fmt.Errorf("decoding Audit Log Drain %q S3 delivery: %w", response.ID, err)
		}
		drain.S3 = &AuditLogDrainS3Delivery{
			Endpoint:             delivery.Endpoint,
			Encoding:             delivery.Encoding,
			RoleARN:              delivery.RoleARN,
			Region:               delivery.Region,
			ServerSideEncryption: delivery.ServerSideEncryption,
			ObjectACL:            delivery.ObjectACL,
		}
	default:
		return AuditLogDrain{}, fmt.Errorf("drain %q uses unsupported Audit Log Drain delivery type %q", response.ID, deliveryType.Type)
	}

	return drain, nil
}

func (c *Client) CreateAuditLogDrain(ctx context.Context, request CreateAuditLogDrainRequest) (AuditLogDrain, error) {
	if (request.HTTP == nil) == (request.S3 == nil) {
		return AuditLogDrain{}, fmt.Errorf("exactly one Audit Log Drain delivery must be configured")
	}

	var delivery any
	if request.HTTP != nil {
		headers := request.HTTP.Headers
		if headers == nil {
			headers = map[string]string{}
		}
		delivery = auditLogDrainsDeliveryHTTP{
			Type:        "http",
			Endpoint:    request.HTTP.Endpoint,
			Encoding:    request.HTTP.Encoding,
			Compression: request.HTTP.Compression,
			Headers:     headers,
			Secret:      request.HTTP.Secret,
		}
	} else {
		delivery = auditLogDrainsDeliveryS3{
			Type:                 "s3",
			Endpoint:             request.S3.Endpoint,
			Encoding:             request.S3.Encoding,
			Compression:          "none",
			FileStructure:        "hive",
			RoleARN:              request.S3.RoleARN,
			Region:               request.S3.Region,
			ServerSideEncryption: request.S3.ServerSideEncryption,
			ObjectACL:            request.S3.ObjectACL,
		}
	}

	url := fmt.Sprintf("%s/v1/drains", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := auditLogDrainsCreateRequest{
		Name:     request.Name,
		Projects: "all",
		Schemas: map[string]drainsSchemaVersion{
			"audit_log": {Version: "v1"},
		},
		Delivery: delivery,
		Source:   drainsSource{Kind: "self-served"},
	}

	tflog.Info(ctx, "creating Audit Log Drain", map[string]any{"url": url})
	var response auditLogDrainsResponse
	if err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(payload)),
	}, &response); err != nil {
		return AuditLogDrain{}, err
	}
	return auditLogDrainFromResponse(response)
}

func (c *Client) GetAuditLogDrain(ctx context.Context, id, teamID string) (AuditLogDrain, error) {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "reading Audit Log Drain", map[string]any{"url": url})

	var response auditLogDrainsResponse
	if err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: url}, &response); err != nil {
		return AuditLogDrain{}, err
	}
	return auditLogDrainFromResponse(response)
}

func (c *Client) DeleteAuditLogDrain(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/drains/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting Audit Log Drain", map[string]any{"url": url})
	return c.doRequest(clientRequest{ctx: ctx, method: "DELETE", url: url}, nil)
}
