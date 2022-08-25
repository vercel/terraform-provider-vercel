package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UpdateProjectDomainRequest defines the information necessary to update a project domain.
type UpdateProjectDomainRequest struct {
	GitBranch          *string `json:"gitBranch"`
	Redirect           *string `json:"redirect"`
	RedirectStatusCode *int64  `json:"redirectStatusCode"`
}

// UpdateProjectDomain updates an existing project domain within Vercel.
func (c *Client) UpdateProjectDomain(ctx context.Context, projectID, domain, teamID string, request UpdateProjectDomainRequest) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}

	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "updating project domain", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &r)
	return r, err
}
