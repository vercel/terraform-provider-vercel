package client

import (
	"context"
	"log"
	"net/http"
	"strings"
)

type DeploymentFile struct {
	File string `json:"file,omitempty"`
	Sha  string `json:"sha,omitempty"`
	Size int    `json:"size,omitempty"`
}

type CreateDeploymentRequest struct {
	Aliases   []string               `json:"alias,omitempty"`
	Files     []DeploymentFile       `json:"files,omitempty"`
	Functions map[string]interface{} `json:"functions,omitempty"`
	ProjectID string                 `json:"project,omitempty"`
	Name      string                 `json:"name"`
	Regions   []string               `json:"regions,omitempty"`
	Routes    []interface{}          `json:"routes,omitempty"`
	Target    string                 `json:"target,omitempty"`
}

type DeploymentResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	CreatedIn string `json:"createdIn"`
}

func (c *Client) CreateDeployment(ctx context.Context, request CreateDeploymentRequest) (r DeploymentResponse, err error) {
	request.Name = request.ProjectID // Name is ignored if project is specified
	body := string(mustMarshal(request))
	log.Printf("[DEBUG] CreateDeploymentRequest: %s", body)
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/v12/now/deployments?skipAutoDetectionConfirmation=1",
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
