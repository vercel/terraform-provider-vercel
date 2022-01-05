package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type MissingFilesError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Missing []string `json:"missing"`
}

func (e MissingFilesError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

func (c *Client) CreateDeployment(ctx context.Context, request CreateDeploymentRequest) (r DeploymentResponse, err error) {
	request.Name = request.ProjectID // Name is ignored if project is specified
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
	var apiErr APIError
	if errors.As(err, &apiErr) && apiErr.Code == "missing_files" {
		var missingFilesError MissingFilesError
		err = json.Unmarshal(apiErr.RawMessage, &struct {
			Error *MissingFilesError `json:"error"`
		}{
			Error: &missingFilesError,
		})
		if err != nil {
			return r, fmt.Errorf("error unmarshaling missing files error: %w", err)
		}
		return r, missingFilesError
	}
	return r, err
}
