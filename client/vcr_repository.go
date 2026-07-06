package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// VCRRepository represents a Vercel Container Registry repository. The VCR CRUD
// API lives under `/v1/vcr/repository` (see the api-vcr service) and is gated
// behind the `vercel-enable-vcr` feature flag; when the flag is disabled the
// API intentionally returns 404.
type VCRRepository struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"projectId"`
	URL       string `json:"url"`
	TeamID    string `json:"-"`
}

// vcrRepositoryResponse tolerates both a bare repository object and one wrapped
// in a `repository` envelope.
type vcrRepositoryResponse struct {
	VCRRepository
	Repository *VCRRepository `json:"repository"`
}

func (r vcrRepositoryResponse) repository() VCRRepository {
	if r.Repository != nil {
		return *r.Repository
	}
	return r.VCRRepository
}

type CreateVCRRepositoryRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
}

func (c *Client) CreateVCRRepository(ctx context.Context, request CreateVCRRepositoryRequest) (res VCRRepository, err error) {
	url := fmt.Sprintf("%s/v1/vcr/repository", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating vcr repository", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var out vcrRepositoryResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &out)
	if err != nil {
		return res, err
	}
	res = out.repository()
	if res.Name == "" {
		res.Name = request.Name
	}
	if res.ProjectID == "" {
		res.ProjectID = request.ProjectID
	}
	res.TeamID = c.TeamID(request.TeamID)
	return res, nil
}

type GetVCRRepositoryRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	// IDOrName is the repository id or, more commonly, its name. The VCR API
	// resolves a repository by `:idOrName`.
	IDOrName string `json:"-"`
}

func (c *Client) GetVCRRepository(ctx context.Context, request GetVCRRepositoryRequest) (res VCRRepository, err error) {
	url := fmt.Sprintf("%s/v1/vcr/repository/%s?projectId=%s", c.baseURL, request.IDOrName, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "getting vcr repository", map[string]any{
		"url": url,
	})
	var out vcrRepositoryResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &out)
	if err != nil {
		return res, err
	}
	res = out.repository()
	if res.Name == "" {
		res.Name = request.IDOrName
	}
	if res.ProjectID == "" {
		res.ProjectID = request.ProjectID
	}
	res.TeamID = c.TeamID(request.TeamID)
	return res, nil
}

type DeleteVCRRepositoryRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	IDOrName  string `json:"-"`
}

func (c *Client) DeleteVCRRepository(ctx context.Context, request DeleteVCRRepositoryRequest) error {
	url := fmt.Sprintf("%s/v1/vcr/repository/%s?projectId=%s", c.baseURL, request.IDOrName, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.TeamID(request.TeamID))
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
