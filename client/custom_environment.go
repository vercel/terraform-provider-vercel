package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type BranchMatcher struct {
	Pattern string `json:"pattern"`
	Type    string `json:"type"`
}

type CreateCustomEnvironmentRequest struct {
	TeamID        string         `json:"-"`
	ProjectID     string         `json:"-"`
	Slug          string         `json:"slug"`
	Description   string         `json:"description"`
	BranchMatcher *BranchMatcher `json:"branchMatcher,omitempty"`
}

type CustomEnvironmentResponse struct {
	ID            string         `json:"id"`
	Description   string         `json:"description"`
	Slug          string         `json:"slug"`
	BranchMatcher *BranchMatcher `json:"branchMatcher"`
	TeamID        string         `json:"-"`
	ProjectID     string         `json:"-"`
}

func (c *Client) CreateCustomEnvironment(ctx context.Context, request CreateCustomEnvironmentRequest) (res CustomEnvironmentResponse, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/custom-environments", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating custom environment", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &res)
	if err != nil {
		return res, err
	}
	res.TeamID = c.teamID(request.TeamID)
	res.ProjectID = request.ProjectID
	return res, nil
}

type GetCustomEnvironmentRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Slug      string `json:"-"`
}

func (c *Client) GetCustomEnvironment(ctx context.Context, request GetCustomEnvironmentRequest) (res CustomEnvironmentResponse, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/custom-environments/%s", c.baseURL, request.ProjectID, request.Slug)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	tflog.Info(ctx, "getting custom environment", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &res)
	if err != nil {
		return res, err
	}
	res.TeamID = c.teamID(request.TeamID)
	res.ProjectID = request.ProjectID
	return res, nil

}

type UpdateCustomEnvironmentRequest struct {
	TeamID        string         `json:"-"`
	ProjectID     string         `json:"-"`
	OldSlug       string         `json:"-"` // Needed to get the right URL
	Slug          string         `json:"slug"`
	Description   string         `json:"description"`
	BranchMatcher *BranchMatcher `json:"branchMatcher"`
}

func (c *Client) UpdateCustomEnvironment(ctx context.Context, request UpdateCustomEnvironmentRequest) (res CustomEnvironmentResponse, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/custom-environments/%s", c.baseURL, request.ProjectID, request.OldSlug)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating custom environment", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &res)
	if err != nil {
		return res, err
	}
	res.TeamID = c.teamID(request.TeamID)
	res.ProjectID = request.ProjectID
	return res, nil
}

type DeleteCustomEnvironmentRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Slug      string `json:"-"`
}

func (c *Client) DeleteCustomEnvironment(ctx context.Context, request DeleteCustomEnvironmentRequest) error {
	url := fmt.Sprintf("%s/v1/projects/%s/custom-environments/%s", c.baseURL, request.ProjectID, request.Slug)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	tflog.Info(ctx, "deleting custom environment", map[string]interface{}{
		"url": url,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "{ \"deleteUnassignedEnvironmentVariables\": true }",
	}, nil)
	return err
}
