package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type ProjectDomainResponse struct {
	Name               string  `json:"name"`
	ProjectID          string  `json:"projectId"`
	Redirect           *string `json:"redirect"`
	RedirectStatusCode *int64  `json:"redirectStatusCode"`
	GitBranch          *string `json:"gitBranch"`
}

func (c *Client) GetProjectDomain(ctx context.Context, projectID, domain, teamID string) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
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
