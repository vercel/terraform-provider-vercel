package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type ProjectResponse struct {
	BuildCommand    *string            `json:"buildCommand"`
	DevCommand      *string            `json:"devCommand"`
	Env             *map[string]string `json:"env"`
	Framework       *string            `json:"framework"`
	ID              string             `json:"id"`
	InstallCommand  *string            `json:"installCommand"`
	Name            string             `json:"name"`
	OutputDirectory *string            `json:"outputDirectory"`
	PublicSource    *bool              `json:"publicSource"`
	RootDirectory   *string            `json:"rootDirectory"`
}

func (c *Client) GetProject(ctx context.Context, projectID, teamID string) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
