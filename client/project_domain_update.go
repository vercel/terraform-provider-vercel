package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type UpdateProjectDomainRequest struct {
	GitBranch          *string `json:"gitBranch"`
	Redirect           *string `json:"redirect"`
	RedirectStatusCode *int64  `json:"redirectStatusCode"`
}

func (c *Client) UpdateProjectDomain(ctx context.Context, projectID, domain, teamID string, request UpdateProjectDomainRequest) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
