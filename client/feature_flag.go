package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type FeatureFlag struct {
	ID           string                            `json:"id"`
	Slug         string                            `json:"slug"`
	Kind         string                            `json:"kind"`
	Description  string                            `json:"description,omitempty"`
	State        string                            `json:"state"`
	ProjectID    string                            `json:"projectId"`
	OwnerID      string                            `json:"ownerId"`
	TypeName     string                            `json:"typeName"`
	CreatedBy    string                            `json:"createdBy"`
	Seed         int                               `json:"seed"`
	Revision     int                               `json:"revision"`
	Variants     []FeatureFlagVariant              `json:"variants"`
	Environments map[string]FeatureFlagEnvironment `json:"environments"`
}

type FeatureFlagVariant struct {
	ID          string `json:"id"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Value       any    `json:"value"`
}

type FeatureFlagEnvironment struct {
	Active        bool                                                       `json:"active"`
	Revision      *int                                                       `json:"revision,omitempty"`
	Rules         []json.RawMessage                                          `json:"rules"`
	Fallthrough   FeatureFlagOutcome                                         `json:"fallthrough"`
	PausedOutcome FeatureFlagOutcome                                         `json:"pausedOutcome"`
	Reuse         *FeatureFlagReuse                                          `json:"reuse,omitempty"`
	Targets       map[string]map[string]map[string][]FeatureFlagSegmentValue `json:"targets,omitempty"`
}

type FeatureFlagOutcome struct {
	Type             string                          `json:"type"`
	VariantID        string                          `json:"variantId,omitempty"`
	Base             *FeatureFlagSegmentConditionLHS `json:"base,omitempty"`
	Weights          map[string]float64              `json:"weights,omitempty"`
	DefaultVariantID string                          `json:"defaultVariantId,omitempty"`
}

type FeatureFlagReuse struct {
	Active      bool   `json:"active"`
	Environment string `json:"environment"`
}

type CreateFeatureFlagRequest struct {
	ProjectID    string                            `json:"-"`
	TeamID       string                            `json:"-"`
	Key          string                            `json:"-"`
	Slug         string                            `json:"slug"`
	Kind         string                            `json:"kind"`
	Variants     []FeatureFlagVariant              `json:"variants,omitempty"`
	Environments map[string]FeatureFlagEnvironment `json:"environments"`
	Description  string                            `json:"description,omitempty"`
	State        string                            `json:"state,omitempty"`
	Seed         *int64                            `json:"seed,omitempty"`
}

type UpdateFeatureFlagRequest struct {
	ProjectID    string                            `json:"-"`
	FlagID       string                            `json:"-"`
	FlagIDOrSlug string                            `json:"-"`
	TeamID       string                            `json:"-"`
	Key          string                            `json:"-"`
	CreatedBy    string                            `json:"createdBy,omitempty"`
	Message      string                            `json:"message,omitempty"`
	Slug         string                            `json:"slug,omitempty"`
	Kind         string                            `json:"kind,omitempty"`
	Variants     []FeatureFlagVariant              `json:"variants,omitempty"`
	Environments map[string]FeatureFlagEnvironment `json:"environments,omitempty"`
	Description  string                            `json:"description,omitempty"`
	State        string                            `json:"state,omitempty"`
	Seed         *int64                            `json:"seed,omitempty"`
}

type GetFeatureFlagRequest struct {
	ProjectID    string
	FlagID       string
	FlagIDOrSlug string
	TeamID       string
}

type DeleteFeatureFlagRequest struct {
	ProjectID    string
	FlagID       string
	FlagIDOrSlug string
	TeamID       string
}

func (c *Client) CreateFeatureFlag(ctx context.Context, request CreateFeatureFlagRequest) (r FeatureFlag, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, "flags", request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating feature flag", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   payload,
	}, &r)
	return r, err
}

func (c *Client) GetFeatureFlag(ctx context.Context, request GetFeatureFlagRequest) (r FeatureFlag, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("flags/%s", featureFlagIdentifier(request.FlagIDOrSlug, request.FlagID)), request.TeamID)
	tflog.Info(ctx, "getting feature flag", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &r)
	return r, err
}

func (c *Client) UpdateFeatureFlag(ctx context.Context, request UpdateFeatureFlagRequest) (r FeatureFlag, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("flags/%s", featureFlagIdentifier(request.FlagIDOrSlug, request.FlagID)), request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating feature flag", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	return r, err
}

func (c *Client) DeleteFeatureFlag(ctx context.Context, request DeleteFeatureFlagRequest) error {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("flags/%s", featureFlagIdentifier(request.FlagIDOrSlug, request.FlagID)), request.TeamID)
	tflog.Info(ctx, "deleting feature flag", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

type FeatureFlagSegment struct {
	ID             string                 `json:"id"`
	Slug           string                 `json:"slug"`
	Label          string                 `json:"label"`
	Description    string                 `json:"description,omitempty"`
	CreatedBy      string                 `json:"createdBy,omitempty"`
	UsedByFlags    []string               `json:"usedByFlags,omitempty"`
	UsedBySegments []string               `json:"usedBySegments,omitempty"`
	CreatedAt      int64                  `json:"createdAt"`
	UpdatedAt      int64                  `json:"updatedAt"`
	ProjectID      string                 `json:"projectId"`
	TypeName       string                 `json:"typeName"`
	Data           FeatureFlagSegmentData `json:"data"`
	Hint           string                 `json:"hint"`
}

type FeatureFlagSegmentData struct {
	Rules   []FeatureFlagSegmentRule                        `json:"rules,omitempty"`
	Include map[string]map[string][]FeatureFlagSegmentValue `json:"include,omitempty"`
	Exclude map[string]map[string][]FeatureFlagSegmentValue `json:"exclude,omitempty"`
}

type FeatureFlagSegmentRule struct {
	ID         string                        `json:"id"`
	Conditions []FeatureFlagSegmentCondition `json:"conditions"`
	Outcome    FeatureFlagSegmentOutcome     `json:"outcome"`
}

type FeatureFlagSegmentCondition struct {
	LHS FeatureFlagSegmentConditionLHS `json:"lhs"`
	CMP string                         `json:"cmp"`
	RHS any                            `json:"rhs,omitempty"`
}

type FeatureFlagSegmentConditionLHS struct {
	Type      string `json:"type"`
	Kind      string `json:"kind,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}

type FeatureFlagSegmentOutcome struct {
	Type         string                          `json:"type"`
	Base         *FeatureFlagSegmentConditionLHS `json:"base,omitempty"`
	PassPromille *float64                        `json:"passPromille,omitempty"`
}

type FeatureFlagSegmentValue struct {
	Value string `json:"value"`
	Note  string `json:"note,omitempty"`
}

type CreateFeatureFlagSegmentRequest struct {
	ProjectID   string                 `json:"-"`
	TeamID      string                 `json:"-"`
	Slug        string                 `json:"slug"`
	Label       string                 `json:"label"`
	Description string                 `json:"description,omitempty"`
	Data        FeatureFlagSegmentData `json:"data"`
	Hint        string                 `json:"hint"`
}

type UpdateFeatureFlagSegmentRequest struct {
	ProjectID       string                 `json:"-"`
	SegmentID       string                 `json:"-"`
	SegmentIDOrSlug string                 `json:"-"`
	TeamID          string                 `json:"-"`
	Slug            string                 `json:"slug,omitempty"`
	Label           string                 `json:"label,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Data            FeatureFlagSegmentData `json:"data,omitempty"`
	Hint            string                 `json:"hint,omitempty"`
}

type GetFeatureFlagSegmentRequest struct {
	ProjectID       string
	SegmentID       string
	SegmentIDOrSlug string
	TeamID          string
}

type DeleteFeatureFlagSegmentRequest struct {
	ProjectID       string
	SegmentID       string
	SegmentIDOrSlug string
	TeamID          string
}

func (c *Client) CreateFeatureFlagSegment(ctx context.Context, request CreateFeatureFlagSegmentRequest) (r FeatureFlagSegment, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, "segments", request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating feature flag segment", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   payload,
	}, &r)
	return r, err
}

func (c *Client) GetFeatureFlagSegment(ctx context.Context, request GetFeatureFlagSegmentRequest) (r FeatureFlagSegment, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("segments/%s", featureFlagIdentifier(request.SegmentIDOrSlug, request.SegmentID)), request.TeamID)
	tflog.Info(ctx, "getting feature flag segment", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &r)
	return r, err
}

func (c *Client) UpdateFeatureFlagSegment(ctx context.Context, request UpdateFeatureFlagSegmentRequest) (r FeatureFlagSegment, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("segments/%s", featureFlagIdentifier(request.SegmentIDOrSlug, request.SegmentID)), request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating feature flag segment", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	return r, err
}

func (c *Client) DeleteFeatureFlagSegment(ctx context.Context, request DeleteFeatureFlagSegmentRequest) error {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("segments/%s", featureFlagIdentifier(request.SegmentIDOrSlug, request.SegmentID)), request.TeamID)
	tflog.Info(ctx, "deleting feature flag segment", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

type FeatureFlagSDKKey struct {
	HashKey          string `json:"hashKey"`
	ProjectID        string `json:"projectId"`
	Type             string `json:"type"`
	Environment      string `json:"environment"`
	CreatedBy        string `json:"createdBy"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
	Label            string `json:"label,omitempty"`
	DeletedAt        *int64 `json:"deletedAt,omitempty"`
	KeyValue         string `json:"keyValue,omitempty"`
	TokenValue       string `json:"tokenValue,omitempty"`
	ConnectionString string `json:"connectionString,omitempty"`
}

type CreateFeatureFlagSDKKeyRequest struct {
	ProjectID   string `json:"-"`
	TeamID      string `json:"-"`
	Type        string `json:"sdkKeyType"`
	Environment string `json:"environment"`
	Label       string `json:"label,omitempty"`
}

type ListFeatureFlagSDKKeysRequest struct {
	ProjectID string
	TeamID    string
}

type DeleteFeatureFlagSDKKeyRequest struct {
	ProjectID string
	HashKey   string
	TeamID    string
}

func (c *Client) CreateFeatureFlagSDKKey(ctx context.Context, request CreateFeatureFlagSDKKeyRequest) (r FeatureFlagSDKKey, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, "sdk-keys", request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating feature flag sdk key", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   payload,
	}, &r)
	return r, err
}

func (c *Client) ListFeatureFlagSDKKeys(ctx context.Context, request ListFeatureFlagSDKKeysRequest) (r []FeatureFlagSDKKey, err error) {
	url := featureFlagScopedURL(c, request.ProjectID, "sdk-keys", request.TeamID)
	tflog.Info(ctx, "listing feature flag sdk keys", map[string]any{
		"url": url,
	})
	var response struct {
		Data []FeatureFlagSDKKey `json:"data"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	return response.Data, err
}

func (c *Client) DeleteFeatureFlagSDKKey(ctx context.Context, request DeleteFeatureFlagSDKKeyRequest) error {
	url := featureFlagScopedURL(c, request.ProjectID, fmt.Sprintf("sdk-keys/%s", request.HashKey), request.TeamID)
	tflog.Info(ctx, "deleting feature flag sdk key", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

func featureFlagScopedURL(c *Client, projectIDOrName, path, teamID string) string {
	url := fmt.Sprintf("%s/v1/projects/%s/feature-flags/%s", c.baseURL, projectIDOrName, path)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	return url
}

func featureFlagIdentifier(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func (r CreateFeatureFlagRequest) MarshalJSON() ([]byte, error) {
	type requestBody struct {
		Slug         string                            `json:"slug"`
		Kind         string                            `json:"kind"`
		Variants     []FeatureFlagVariant              `json:"variants,omitempty"`
		Environments map[string]FeatureFlagEnvironment `json:"environments,omitempty"`
		Description  string                            `json:"description"`
		State        string                            `json:"state,omitempty"`
		Seed         *int64                            `json:"seed,omitempty"`
	}

	return json.Marshal(requestBody{
		Slug:         featureFlagIdentifier(r.Slug, r.Key),
		Kind:         r.Kind,
		Variants:     r.Variants,
		Environments: r.Environments,
		Description:  r.Description,
		State:        r.State,
		Seed:         r.Seed,
	})
}

func (r UpdateFeatureFlagRequest) MarshalJSON() ([]byte, error) {
	type requestBody struct {
		CreatedBy    string                            `json:"createdBy,omitempty"`
		Message      string                            `json:"message,omitempty"`
		Slug         string                            `json:"slug,omitempty"`
		Kind         string                            `json:"kind,omitempty"`
		Variants     []FeatureFlagVariant              `json:"variants,omitempty"`
		Environments map[string]FeatureFlagEnvironment `json:"environments,omitempty"`
		Description  string                            `json:"description,omitempty"`
		State        string                            `json:"state,omitempty"`
		Seed         *int64                            `json:"seed,omitempty"`
	}

	return json.Marshal(requestBody{
		CreatedBy:    r.CreatedBy,
		Message:      r.Message,
		Slug:         featureFlagIdentifier(r.Slug, r.Key),
		Kind:         r.Kind,
		Variants:     r.Variants,
		Environments: r.Environments,
		Description:  r.Description,
		State:        r.State,
		Seed:         r.Seed,
	})
}

func (r CreateFeatureFlagSegmentRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"slug":  r.Slug,
		"label": r.Label,
		"hint":  r.Hint,
		"data":  featureFlagSegmentRequestData(r.Data),
	}
	if r.Description != "" {
		body["description"] = r.Description
	}

	return json.Marshal(body)
}

func (r UpdateFeatureFlagSegmentRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{}
	if r.Slug != "" {
		body["slug"] = r.Slug
	}
	if r.Label != "" {
		body["label"] = r.Label
	}
	if r.Description != "" {
		body["description"] = r.Description
	}
	if r.Hint != "" {
		body["hint"] = r.Hint
	}
	if r.Data.Include != nil || r.Data.Exclude != nil || r.Data.Rules != nil {
		body["data"] = featureFlagSegmentRequestData(r.Data)
	}

	return json.Marshal(body)
}

func featureFlagSegmentRequestData(data FeatureFlagSegmentData) map[string]any {
	body := map[string]any{}
	if data.Include != nil {
		body["include"] = data.Include
	}
	if data.Exclude != nil {
		body["exclude"] = data.Exclude
	}
	if data.Rules != nil {
		body["rules"] = data.Rules
	}
	return body
}
