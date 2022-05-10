package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateAliasRequest defines the request the Vercel API expects in order to create an alias.
type CreateAliasRequest struct {
	Alias string `json:"alias"`
}

// CreateAlias creates an alias within Vercel.
func (c *Client) CreateAlias(ctx context.Context, request CreateAliasRequest, deploymentID string, teamID string) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/now/deployments/%s/aliases", c.baseURL, deploymentID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "creating alias", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, err
	}

	return r, nil
}
