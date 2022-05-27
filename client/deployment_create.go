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

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeploymentFile is a struct defining the required information about a singular file
// that should be used within a deployment.
type DeploymentFile struct {
	File string `json:"file"`
	Sha  string `json:"sha"`
	Size int    `json:"size"`
}

type gitSource struct {
	Type      string `json:"type"`
	Org       string `json:"org,omitempty"`
	Repo      string `json:"repo,omitempty"`
	ProjectID int64  `json:"projectId,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Slug      string `json:"slug,omitempty"`
	Ref       string `json:"ref"`
	SHA       string `json:"sha"`
}

// CreateDeploymentRequest defines the request the Vercel API expects in order to create a deployment.
type CreateDeploymentRequest struct {
	Files       []DeploymentFile       `json:"files,omitempty"`
	Functions   map[string]interface{} `json:"functions,omitempty"`
	Environment map[string]string      `json:"env,omitempty"`
	Build       struct {
		Environment map[string]string `json:"env,omitempty"`
	} `json:"build,omitempty"`
	ProjectID       string                 `json:"project,omitempty"`
	ProjectSettings map[string]interface{} `json:"projectSettings"`
	Name            string                 `json:"name"`
	Regions         []string               `json:"regions,omitempty"`
	Routes          []interface{}          `json:"routes,omitempty"`
	Target          string                 `json:"target,omitempty"`
	GitSource       *gitSource             `json:"gitSource,omitempty"`
	Ref             string                 `json:"-"`
}

// DeploymentResponse defines the response the Vercel API returns when a deployment is created or updated.
type DeploymentResponse struct {
	Aliases    []string `json:"alias"`
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
	Build struct {
		Environment []string `json:"env"`
	} `json:"build"`
	AliasAssigned    bool      `json:"aliasAssigned"`
	ChecksConclusion string    `json:"checksConclusion"`
	ErrorCode        string    `json:"errorCode"`
	ErrorMessage     string    `json:"errorMessage"`
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ReadyState       string    `json:"readyState"`
	Target           *string   `json:"target"`
	URL              string    `json:"url"`
	GitSource        gitSource `json:"gitSource"`
}

// IsComplete is used to determine whether a deployment is still processing, or whether it is fully done.
func (dr *DeploymentResponse) IsComplete() bool {
	return dr.AliasAssigned && dr.AliasError == nil
}

// DeploymentLogsURL provides a user friendly URL that links directly to the vercel UI for a particular deployment.
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

// CheckForError checks through the various failure modes of a deployment to see if any were hit.
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

// MissingFilesError is a sentinel error that indicates a deployment could not be created
// because additional files need to be uploaded first.
type MissingFilesError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Missing []string `json:"missing"`
}

// Error gives the MissingFilesError a user friendly error message.
func (e MissingFilesError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

func (c *Client) getGitSource(ctx context.Context, projectID, ref, teamID string) (gs gitSource, err error) {
	project, err := c.GetProject(ctx, projectID, teamID)
	if err != nil {
		return gs, fmt.Errorf("error getting project: %w", err)
	}
	if project.Link == nil {
		return gs, fmt.Errorf("unable to deploy project by ref: project has no linked git repository")
	}

	switch project.Link.Type {
	case "github":
		return gitSource{
			Org:  project.Link.Org,
			Ref:  ref,
			Repo: project.Link.Repo,
			Type: "github",
		}, nil
	case "gitlab":
		return gitSource{
			ProjectID: project.Link.ProjectID,
			Ref:       ref,
			Type:      "gitlab",
		}, nil
	case "bitbucket":
		return gitSource{
			Owner: project.Link.Owner,
			Ref:   ref,
			Slug:  project.Link.Slug,
			Type:  "bitbucket",
		}, nil
	default:
		return gs, fmt.Errorf("unable to deploy project by ref: project has no linked git repository")
	}
}

// CreateDeployment creates a deployment within Vercel.
func (c *Client) CreateDeployment(ctx context.Context, request CreateDeploymentRequest, teamID string) (r DeploymentResponse, err error) {
	request.Name = request.ProjectID                // Name is ignored if project is specified
	request.Build.Environment = request.Environment // Ensure they are both the same, as project environment variables are
	if request.Ref != "" {
		gitSource, err := c.getGitSource(ctx, request.ProjectID, request.Ref, teamID)
		if err != nil {
			return r, err
		}
		request.GitSource = &gitSource
	}
	url := fmt.Sprintf("%s/v12/now/deployments?skipAutoDetectionConfirmation=1", c.baseURL)
	if teamID != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, teamID)
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

	tflog.Trace(ctx, "creating deployment", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
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
