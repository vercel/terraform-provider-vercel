package client

import (
	"context"
	"fmt"

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
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project domain", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
