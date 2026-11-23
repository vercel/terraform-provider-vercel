package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type EdgeConfigToken struct {
	TeamID       string
	Token        string `json:"token"`
	Label        string `json:"label"`
	ID           string `json:"id"`
	EdgeConfigID string `json:"edgeConfigId"`
}

func (e EdgeConfigToken) ConnectionString() string {
	return fmt.Sprintf(
		"https://edge-config.vercel.com/%s?token=%s",
		e.EdgeConfigID,
		e.Token,
	)
}

type CreateEdgeConfigTokenRequest struct {
	Label        string `json:"label"`
	TeamID       string `json:"-"`
	EdgeConfigID string `json:"-"`
}

func (c *Client) CreateEdgeConfigToken(ctx context.Context, request CreateEdgeConfigTokenRequest) (e EdgeConfigToken, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/token", c.baseURL, request.EdgeConfigID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating edge config token", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &e)
	e.Label = request.Label
	e.TeamID = request.TeamID
	e.EdgeConfigID = request.EdgeConfigID
	return e, err
}

type EdgeConfigTokenRequest struct {
	TeamID       string
	EdgeConfigID string
	Token        string
	// ID is preferred over Token wherever the API accepts it, so the
	// plaintext token stays out of URLs, request bodies, and logs.
	ID string
}

func (c *Client) DeleteEdgeConfigToken(ctx context.Context, request EdgeConfigTokenRequest) error {
	url := fmt.Sprintf("%s/v1/edge-config/%s/tokens", c.baseURL, request.EdgeConfigID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	body := struct {
		IDs    []string `json:"ids,omitempty"`
		Tokens []string `json:"tokens,omitempty"`
	}{}
	if request.ID != "" {
		body.IDs = []string{request.ID}
	} else {
		body.Tokens = []string{request.Token}
	}

	tflog.Info(ctx, "deleting edge config token", map[string]any{
		"edge_config_id": request.EdgeConfigID,
		"token_id":       request.ID,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   string(mustMarshal(body)),
	}, nil)
}

// GetEdgeConfigTokenByID looks a token up by its ID through the token list.
// The list never returns the plaintext token, so the returned Token is empty.
func (c *Client) GetEdgeConfigTokenByID(ctx context.Context, request EdgeConfigTokenRequest) (e EdgeConfigToken, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/tokens", c.baseURL, request.EdgeConfigID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	tflog.Info(ctx, "getting edge config token by id", map[string]any{
		"edge_config_id": request.EdgeConfigID,
		"token_id":       request.ID,
	})

	var tokens []EdgeConfigToken
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &tokens)
	if err != nil {
		return e, err
	}
	for _, t := range tokens {
		if t.ID == request.ID {
			t.Token = ""
			t.TeamID = request.TeamID
			return t, nil
		}
	}
	return e, APIError{
		StatusCode: 404,
		Code:       "not_found",
		Message:    fmt.Sprintf("Edge Config token %s not found", request.ID),
	}
}

func (c *Client) GetEdgeConfigToken(ctx context.Context, request EdgeConfigTokenRequest) (e EdgeConfigToken, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/token/%s", c.baseURL, request.EdgeConfigID, request.Token)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	tflog.Info(ctx, "getting edge config token", map[string]any{
		"edge_config_id": request.EdgeConfigID,
	})

	err = c.doRequest(clientRequest{
		ctx:               ctx,
		method:            "GET",
		url:               url,
		sensitiveResponse: true,
	}, &e)
	// The plaintext `token` on this endpoint's response is deprecated and will
	// be removed in a future API rollout. It's already part of the request
	// path, so we repopulate it from the request to keep the returned struct —
	// and the computed `connection_string` that depends on it — stable across
	// that response-shape change.
	e.Token = request.Token
	e.TeamID = request.TeamID
	return e, err
}
