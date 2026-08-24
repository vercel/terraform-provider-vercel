package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AIGatewayAPIKey struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	PartialKey   string  `json:"partialKey"`
	TeamID       string  `json:"teamId"`
	APIKeyString *string `json:"-"`
}

type CreateAIGatewayAPIKeyRequest struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	TeamID  string `json:"-"`
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

func (c *Client) apiKeyURL(keyID, teamID string) string {
	requestURL := fmt.Sprintf("%s/v1/api-keys", c.baseURL)
	if keyID != "" {
		requestURL = fmt.Sprintf("%s/%s", requestURL, url.PathEscape(keyID))
	}
	query := url.Values{}
	if teamID := c.TeamID(teamID); teamID != "" {
		query.Set("teamId", teamID)
	}
	return urlWithQuery(requestURL, query)
}
