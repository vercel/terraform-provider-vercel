package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type CreateProjectDomainRequest struct {
	Name               string `json:"name"`
	GitBranch          string `json:"gitBranch,omitempty"`
	Redirect           string `json:"redirect,omitempty"`
	RedirectStatusCode int64  `json:"redirectStatusCode,omitempty"`
}

func (c *Client) CreateProjectDomain(ctx context.Context, projectID, teamID string, request CreateProjectDomainRequest) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
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
	return r, err
}
