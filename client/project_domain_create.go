package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// CreateProjectDomainRequest defines the information necessary to create a project domain.
// A project domain is an association of a specific domain name to a project. These are typically
// used to assign a domain name to any production deployments, but can also be used to configure
// redirects, or to give specific git branches a domain name.
type CreateProjectDomainRequest struct {
	Name               string `json:"name"`
	GitBranch          string `json:"gitBranch,omitempty"`
	Redirect           string `json:"redirect,omitempty"`
	RedirectStatusCode int64  `json:"redirectStatusCode,omitempty"`
}

// CreateProjectDomain creates a project domain within Vercel.
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
