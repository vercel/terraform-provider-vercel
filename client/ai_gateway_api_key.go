package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AIGatewayAPIKey struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	PartialKey   string                `json:"partialKey"`
	TeamID       string                `json:"teamId"`
	ProjectID    *string               `json:"projectId"`
	ExpiresAt    *int64                `json:"expiresAt"`
	Quota        *AIGatewayAPIKeyQuota `json:"quota"`
	APIKeyString *string               `json:"-"`
}

// AIGatewayAPIKeyQuota is the quota associated with an API key. It is only
// populated by the list endpoint, not the get-by-ID endpoint.
type AIGatewayAPIKeyQuota struct {
	LimitAmount     float64 `json:"limitAmount"`
	RefreshPeriod   string  `json:"refreshPeriod"`
	AlertThresholds []int64 `json:"alertThresholds,omitempty"`
}

type CreateAIGatewayAPIKeyRequest struct {
	Name           string                `json:"name"`
	Purpose        string                `json:"purpose"`
	ExpiresAt      *int64                `json:"expiresAt,omitempty"`
	ProjectID      *string               `json:"projectId,omitempty"`
	AIGatewayQuota *AIGatewayAPIKeyQuota `json:"aiGatewayQuota,omitempty"`
	TeamID         string                `json:"-"`
}

func (c *Client) CreateAIGatewayAPIKey(ctx context.Context, request CreateAIGatewayAPIKeyRequest) (k AIGatewayAPIKey, err error) {
	requestURL := c.apiKeyURL("", request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating AI Gateway API key", map[string]any{
		"url":     requestURL,
		"payload": payload,
	})

	var response struct {
		APIKeyString string          `json:"apiKeyString"`
		APIKey       AIGatewayAPIKey `json:"apiKey"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    requestURL,
		body:   payload,
	}, &response)
	if response.APIKeyString != "" {
		response.APIKey.APIKeyString = &response.APIKeyString
	}
	return response.APIKey, err
}

func (c *Client) GetAIGatewayAPIKey(ctx context.Context, keyID, teamID string) (k AIGatewayAPIKey, err error) {
	requestURL := c.apiKeyURL(keyID, teamID)
	tflog.Info(ctx, "getting AI Gateway API key", map[string]any{
		"url": requestURL,
	})

	var response struct {
		APIKey AIGatewayAPIKey `json:"apiKey"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    requestURL,
	}, &response)
	return response.APIKey, err
}

// GetAIGatewayAPIKeyQuota returns the quota for an API key, or nil when the
// key has no active quota. The get-by-ID endpoint does not return quota
// information, so this pages through the list endpoint, which decorates
// AI Gateway keys with their quota.
func (c *Client) GetAIGatewayAPIKeyQuota(ctx context.Context, keyID, teamID string) (*AIGatewayAPIKeyQuota, error) {
	cursor := ""
	for {
		query := url.Values{}
		query.Set("purpose", "ai-gateway")
		query.Set("limit", "100")
		if teamID := c.TeamID(teamID); teamID != "" {
			query.Set("teamId", teamID)
		}
		if cursor != "" {
			query.Set("until", cursor)
		}
		requestURL := urlWithQuery(fmt.Sprintf("%s/v1/api-keys", c.baseURL), query)
		tflog.Info(ctx, "listing AI Gateway API keys for quota", map[string]any{
			"url": requestURL,
		})

		var response struct {
			APIKeys    []AIGatewayAPIKey `json:"apiKeys"`
			Pagination struct {
				Next *string `json:"next"`
			} `json:"pagination"`
		}
		if err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "GET",
			url:    requestURL,
		}, &response); err != nil {
			return nil, err
		}

		for _, key := range response.APIKeys {
			if key.ID == keyID {
				return key.Quota, nil
			}
		}

		if response.Pagination.Next == nil || *response.Pagination.Next == "" {
			return nil, nil
		}
		if *response.Pagination.Next == cursor {
			return nil, fmt.Errorf("pagination cursor did not advance")
		}
		cursor = *response.Pagination.Next
	}
}

type UpdateAIGatewayAPIKeyRequest struct {
	KeyID  string `json:"-"`
	TeamID string `json:"-"`
	Name   string `json:"name"`
}

func (c *Client) UpdateAIGatewayAPIKey(ctx context.Context, request UpdateAIGatewayAPIKeyRequest) (k AIGatewayAPIKey, err error) {
	requestURL := c.apiKeyURL(request.KeyID, request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating AI Gateway API key", map[string]any{
		"url":     requestURL,
		"payload": payload,
	})

	var response struct {
		APIKey AIGatewayAPIKey `json:"apiKey"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    requestURL,
		body:   payload,
	}, &response)
	return response.APIKey, err
}

type UpdateAIGatewayAPIKeyQuotaRequest struct {
	KeyID  string `json:"-"`
	TeamID string `json:"-"`
	// LimitAmount must always be set when upserting: the API falls back to
	// creating the quota when one does not exist yet, which requires it.
	LimitAmount     *float64 `json:"limitAmount,omitempty"`
	RefreshPeriod   string   `json:"refreshPeriod,omitempty"`
	AlertThresholds *[]int64 `json:"alertThresholds,omitempty"`
	// Archived set to true archives (soft-deletes) the quota. The API key
	// itself is unaffected. Any non-archiving update revives an archived
	// quota.
	Archived *bool `json:"archived,omitempty"`
}

func (c *Client) UpdateAIGatewayAPIKeyQuota(ctx context.Context, request UpdateAIGatewayAPIKeyQuotaRequest) (*AIGatewayAPIKeyQuota, error) {
	requestURL := fmt.Sprintf("%s/quota", c.apiKeyPath(request.KeyID))
	query := url.Values{}
	if teamID := c.TeamID(request.TeamID); teamID != "" {
		query.Set("teamId", teamID)
	}
	requestURL = urlWithQuery(requestURL, query)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating AI Gateway API key quota", map[string]any{
		"url":     requestURL,
		"payload": payload,
	})

	var response struct {
		Quota *AIGatewayAPIKeyQuota `json:"quota"`
	}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    requestURL,
		body:   payload,
	}, &response)
	return response.Quota, err
}

func (c *Client) DeleteAIGatewayAPIKey(ctx context.Context, keyID, teamID string) error {
	requestURL := c.apiKeyURL(keyID, teamID)
	tflog.Info(ctx, "deleting AI Gateway API key", map[string]any{
		"url": requestURL,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    requestURL,
	}, nil)
}

func (c *Client) apiKeyPath(keyID string) string {
	requestURL := fmt.Sprintf("%s/v1/api-keys", c.baseURL)
	if keyID != "" {
		requestURL = fmt.Sprintf("%s/%s", requestURL, url.PathEscape(keyID))
	}
	return requestURL
}

func (c *Client) apiKeyURL(keyID, teamID string) string {
	query := url.Values{}
	if teamID := c.TeamID(teamID); teamID != "" {
		query.Set("teamId", teamID)
	}
	return urlWithQuery(c.apiKeyPath(keyID), query)
}
