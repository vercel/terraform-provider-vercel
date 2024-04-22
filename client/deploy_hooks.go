package client

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type DeployHook struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
	URL  string `json:"url"`
	ID   string `json:"id"`
}

type CreateDeployHookRequest struct {
	ProjectID string `json:"-"`
	TeamID    string `json:"-"`
	Name      string `json:"name"`
	Ref       string `json:"ref"`
}

func (c *Client) CreateDeployHook(ctx context.Context, request CreateDeployHookRequest) (h DeployHook, err error) {
	url := fmt.Sprintf("%s/v2/projects/%s/deploy-hooks", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating deploy hook", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})

	var r ProjectResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return h, fmt.Errorf("error creating deploy hook: %w", err)
	}

	// Reverse the list as newest created are at the end
	slices.Reverse(r.Link.DeployHooks)
	for _, hook := range r.Link.DeployHooks {
		if hook.Name == request.Name && hook.Ref == request.Ref {
			return hook, nil
		}
	}

	return h, fmt.Errorf("deploy hook was created successfully, but could not be found")
}

type DeleteDeployHookRequest struct {
	ProjectID string
	TeamID    string
	ID        string
}

func (c *Client) DeleteDeployHook(ctx context.Context, request DeleteDeployHookRequest) error {
	url := fmt.Sprintf("%s/v2/projects/%s/deploy-hooks/%s", c.baseURL, request.ProjectID, request.ID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating deploy hook", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})

	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
	if err != nil {
		return fmt.Errorf("error deleting deploy hook: %w", err)
	}
	return nil
}
