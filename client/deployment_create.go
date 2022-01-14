package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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
	AliasError *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"aliasError"`
	AliasWarning *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Link    string `json:"link"`
		Action  string `json:"action"`
	} `json:"aliasWarning"`
	Creator struct {
		Username string `json:"username"`
	} `json:"creator"`
	Team *struct {
		Slug string `json:"slug"`
	} `json:"team"`
	ID               string  `json:"id"`
	ProjectID        string  `json:"projectId"`
	Target           *string `json:"target"`
	URL              string  `json:"url"`
	AliasAssigned    bool    `json:"aliasAssigned"`
	ChecksConclusion string  `json:"checksConclusion"`
	ReadyState       string  `json:"readyState"`
	ErrorCode        string  `json:"errorCode"`
	ErrorMessage     string  `json:"errorMessage"`
}

func (dr *DeploymentResponse) IsComplete() bool {
	return dr.AliasAssigned && dr.AliasError == nil
}

func (dr *DeploymentResponse) DeploymentLogsURL(projectID string) string {
	teamSlug := dr.Creator.Username
	if dr.Team != nil {
		teamSlug = dr.Creator.Username
	}
	return fmt.Sprintf(
		"https://vercel.com/%s/%s/%s",
		teamSlug,
		projectID,
		strings.TrimPrefix(dr.ID, "dpl_"),
	)
}

func (dr *DeploymentResponse) CheckForError(projectID string) error {
	if dr.ReadyState == "CANCELED" {
		return fmt.Errorf("deployment canceled")
	}

	if dr.ReadyState == "ERROR" {
		return fmt.Errorf(
			"%s - %s. Visit %s for more information",
			dr.ErrorCode,
			dr.ErrorMessage,
			dr.DeploymentLogsURL(projectID),
		)
	}

	if dr.ChecksConclusion == "failed" {
		return fmt.Errorf(
			"deployment checks have failed. Visit %s for more information",
			dr.DeploymentLogsURL(projectID),
		)
	}

	if dr.AliasError != nil {
		return fmt.Errorf(
			"%s - %s. Visit %s for more information",
			dr.AliasError.Code,
			dr.AliasError.Message,
			dr.DeploymentLogsURL(projectID),
		)
	}

	return nil
}

type MissingFilesError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Missing []string `json:"missing"`
}

func (e MissingFilesError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

func (c *Client) CreateDeployment(ctx context.Context, request CreateDeploymentRequest, teamID string) (r DeploymentResponse, err error) {
	request.Name = request.ProjectID // Name is ignored if project is specified
	url := fmt.Sprintf("%s/v12/now/deployments?skipAutoDetectionConfirmation=1", c.baseURL)
	if teamID != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
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
	if err != nil {
		return r, err
	}

	// Now we've successfully created a deployment, but the deployment process is async.
	// So poll the deployment until it either fails, or is completed.
	for !r.IsComplete() {
		err = r.CheckForError(request.ProjectID)
		if err != nil {
			return r, err
		}
		time.Sleep(5 * time.Second)
		r, err = c.GetDeployment(ctx, r.ID, teamID)
		if err != nil {
			return r, fmt.Errorf("error getting deployment: %w", err)
		}
	}

	if r.AliasWarning != nil {
		// Log out that there is a warning for an alias.
		log.Printf("[WARN] %s - %s: %s - %s", r.AliasWarning.Code, r.AliasWarning.Message, r.AliasWarning.Action, r.AliasWarning.Link)
	}

	return r, nil
}
