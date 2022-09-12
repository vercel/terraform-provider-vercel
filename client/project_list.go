package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ListProjects lists the top 100 projects (no pagination) from within Vercel.
func (c *Client) ListProjects(ctx context.Context, teamID string) (r []ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects?limit=100", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.teamID(teamID))
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

	pr := struct {
		Projects []ProjectResponse `json:"projects"`
	}{}
	tflog.Trace(ctx, "listing projects", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &pr)
	for _, p := range pr.Projects {
		p.TeamID = c.teamID(teamID)
	}
	return pr.Projects, err
}
