package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ProjectDomainResponse defines the information that Vercel exposes about a domain that is
// associated with a vercel project.
type ProjectDomainResponse struct {
	Name               string  `json:"name"`
	ProjectID          string  `json:"projectId"`
	TeamID             string  `json:"-"`
	Redirect           *string `json:"redirect"`
	RedirectStatusCode *int64  `json:"redirectStatusCode"`
	GitBranch          *string `json:"gitBranch"`
}

// GetProjectDomain retrieves information about a project domain from Vercel.
func (c *Client) GetProjectDomain(ctx context.Context, projectID, domain, teamID string) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		nil,
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "getting project domain", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
