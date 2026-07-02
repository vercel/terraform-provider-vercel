package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type VCRRepository struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
}

type CreateVCRRepositoryRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Name      string `json:"name"`
}

func (c *Client) CreateVCRRepository(ctx context.Context, request CreateVCRRepositoryRequest) (res VCRRepository, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/vcr/repositories", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating vcr repository", map[string]any{
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
	if res.Name == "" {
		res.Name = request.Name
	}
	res.TeamID = c.TeamID(request.TeamID)
	res.ProjectID = request.ProjectID
	return res, nil
}

type GetVCRRepositoryRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Name      string `json:"-"`
}

func (c *Client) GetVCRRepository(ctx context.Context, request GetVCRRepositoryRequest) (res VCRRepository, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/vcr/repositories/%s", c.baseURL, request.ProjectID, request.Name)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "getting vcr repository", map[string]any{
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
	if res.Name == "" {
		res.Name = request.Name
	}
	res.TeamID = c.TeamID(request.TeamID)
	res.ProjectID = request.ProjectID
	return res, nil
}

type DeleteVCRRepositoryRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Name      string `json:"-"`
}

func (c *Client) DeleteVCRRepository(ctx context.Context, request DeleteVCRRepositoryRequest) error {
	url := fmt.Sprintf("%s/v1/projects/%s/vcr/repositories/%s", c.baseURL, request.ProjectID, request.Name)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "deleting vcr repository", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
