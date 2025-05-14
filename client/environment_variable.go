package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// create an environment variable.
type EnvironmentVariableRequest struct {
	Key                  string   `json:"key"`
	Value                string   `json:"value"`
	Target               []string `json:"target,omitempty"`
	CustomEnvironmentIDs []string `json:"customEnvironmentIds,omitempty"`
	GitBranch            *string  `json:"gitBranch,omitempty"`
	Type                 string   `json:"type"`
	Comment              string   `json:"comment"`
}

type CreateEnvironmentVariableRequest struct {
	EnvironmentVariable EnvironmentVariableRequest
	ProjectID           string
	TeamID              string
}

// CreateEnvironmentVariable will create a brand new environment variable if one does not exist.
func (c *Client) CreateEnvironmentVariable(ctx context.Context, request CreateEnvironmentVariableRequest) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/env", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request.EnvironmentVariable))

	tflog.Info(ctx, "creating environment variable", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var response CreateEnvironmentVariableResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)

	if conflictingEnv, isConflicting, err2 := conflictingEnvVar(err); isConflicting {
		if err2 != nil {
			return e, err2
		}
		envs, err3 := c.GetEnvironmentVariables(ctx, request.ProjectID, request.TeamID)
		if err3 != nil {
			return e, fmt.Errorf("%s: unable to list environment variables to detect conflict: %s", err, err3)
		}
		id, found := findConflictingEnvID(request.TeamID, request.ProjectID, conflictingEnv, envs)
		if found {
			return e, fmt.Errorf("%w the conflicting environment variable ID is %s", err, id)
		}
	}

	if err != nil {
		return e, fmt.Errorf("%w - %s", err, payload)
	}
	response.Created.Value = request.EnvironmentVariable.Value
	response.Created.TeamID = c.TeamID(request.TeamID)
	return response.Created, err
}

func overlaps(s []string, e []string) bool {
	set := make(map[string]struct{}, len(s))
	for _, a := range s {
		set[a] = struct{}{}
	}

	for _, b := range e {
		if _, exists := set[b]; exists {
			return true
		}
	}

	return false
}

func findConflictingEnvID(teamID, projectID string, envConflict EnvConflictError, envs []EnvironmentVariable) (string, bool) {
	checkTargetOverlap := len(envConflict.Target) != 0

	for _, env := range envs {
		if env.Key != envConflict.EnvVarKey || env.GitBranch != envConflict.GitBranch {
			continue
		}

		if checkTargetOverlap && !overlaps(env.Target, envConflict.Target) {
			continue
		}

		id := fmt.Sprintf("%s/%s", projectID, env.ID)
		if teamID != "" {
			id = fmt.Sprintf("%s/%s", teamID, id)
		}
		return id, true
	}

	return "", false
}

type CreateEnvironmentVariablesRequest struct {
	EnvironmentVariables []EnvironmentVariableRequest
	ProjectID            string
	TeamID               string
}

type CreateEnvironmentVariablesResponse struct {
	Created []EnvironmentVariable `json:"created"`
	Failed  []FailedItem          `json:"failed"`
}

type FailedItem struct {
	Error struct {
		Action    *string  `json:"action,omitempty"`
		Code      string   `json:"code"`
		EnvVarID  *string  `json:"envVarId,omitempty"`
		EnvVarKey *string  `json:"envVarKey,omitempty"`
		GitBranch *string  `json:"gitBranch,omitempty"`
		Key       *string  `json:"key,omitempty"`
		Link      *string  `json:"link,omitempty"`
		Message   string   `json:"message"`
		Project   *string  `json:"project,omitempty"`
		Target    []string `json:"target,omitempty"`
		Value     *string  `json:"value,omitempty"`
	} `json:"error"`
}

type CreateEnvironmentVariableResponse struct {
	Created EnvironmentVariable `json:"created"`
	Failed  []FailedItem        `json:"failed"`
}

func (c *Client) CreateEnvironmentVariables(ctx context.Context, request CreateEnvironmentVariablesRequest) ([]EnvironmentVariable, error) {
	if len(request.EnvironmentVariables) == 1 {
		env, err := c.CreateEnvironmentVariable(ctx, CreateEnvironmentVariableRequest{
			EnvironmentVariable: request.EnvironmentVariables[0],
			ProjectID:           request.ProjectID,
			TeamID:              request.TeamID,
		})
		return []EnvironmentVariable{env}, err
	}
	url := fmt.Sprintf("%s/v10/projects/%s/env", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request.EnvironmentVariables))

	var response CreateEnvironmentVariablesResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)
	if err != nil {
		return nil, fmt.Errorf("%w - %s", err, payload)
	}

	decrypted := false
	for i := 0; i < len(response.Created); i++ {
		// When env vars are created, their values are encrypted
		response.Created[i].Decrypted = &decrypted
	}

	if len(response.Failed) > 0 {
		envs, err := c.GetEnvironmentVariables(ctx, request.ProjectID, request.TeamID)
		if err != nil {
			return response.Created, fmt.Errorf("failed to create environment variables. error detecting conflicting environment variables: %w", err)
		}
		for _, failed := range response.Failed {
			if failed.Error.Code == "ENV_CONFLICT" {
				id, found := findConflictingEnvID(request.TeamID, request.ProjectID, EnvConflictError{
					Key:       *failed.Error.EnvVarKey,
					Target:    failed.Error.Target,
					GitBranch: failed.Error.GitBranch,
				}, envs)
				if found {
					err = fmt.Errorf("%w, conflicting environment variable ID is %s", err, id)
				} else {
					err = fmt.Errorf("failed to create environment variables, %s", failed.Error.Message)
				}
			} else {
				err = fmt.Errorf("failed to create environment variables, %s", failed.Error.Message)
			}
		}
		return response.Created, err
	}

	return response.Created, err
}

// UpdateEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// update an environment variable.
type UpdateEnvironmentVariableRequest struct {
	Value                string   `json:"value"`
	Target               []string `json:"target"`
	CustomEnvironmentIDs []string `json:"customEnvironmentIds,omitempty"`
	GitBranch            *string  `json:"gitBranch,omitempty"`
	Type                 string   `json:"type"`
	Comment              string   `json:"comment"`
	ProjectID            string   `json:"-"`
	TeamID               string   `json:"-"`
	EnvID                string   `json:"-"`
}

// UpdateEnvironmentVariable will update an existing environment variable to the latest information.
func (c *Client) UpdateEnvironmentVariable(ctx context.Context, request UpdateEnvironmentVariableRequest) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/env/%s", c.baseURL, request.ProjectID, request.EnvID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating environment variable", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &e)
	// The API response returns an encrypted environment variable, but we want to return the decrypted version.
	e.Value = request.Value
	e.TeamID = c.TeamID(request.TeamID)
	return e, err
}

// DeleteEnvironmentVariable will remove an environment variable from Vercel.
func (c *Client) DeleteEnvironmentVariable(ctx context.Context, projectID, teamID, variableID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/env/%s", c.baseURL, projectID, variableID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting environment variable", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}

func (c *Client) GetEnvironmentVariables(ctx context.Context, projectID, teamID string) ([]EnvironmentVariable, error) {
	url := fmt.Sprintf("%s/v8/projects/%s/env?decrypt=true", c.baseURL, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.TeamID(teamID))
	}

	envResponse := struct {
		Env []EnvironmentVariable `json:"envs"`
	}{}
	tflog.Info(ctx, "getting environment variables", map[string]any{
		"url": url,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &envResponse)
	for i := 0; i < len(envResponse.Env); i++ {
		envResponse.Env[i].TeamID = c.TeamID(teamID)
	}
	return envResponse.Env, err
}

// GetEnvironmentVariable gets a singluar environment variable from Vercel based on its ID.
func (c *Client) GetEnvironmentVariable(ctx context.Context, projectID, teamID, envID string) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/env/%s", c.baseURL, projectID, envID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "getting environment variable", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &e)
	e.TeamID = c.TeamID(teamID)
	return e, err
}
