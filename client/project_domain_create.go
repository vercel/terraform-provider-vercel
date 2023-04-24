package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	url := fmt.Sprintf("%s/v10/projects/%s/domains", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "creating project domain", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
