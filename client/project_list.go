package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ListProjects lists the top 100 projects (no pagination) from within Vercel.
func (c *Client) ListProjects(ctx context.Context, teamID string) (r []ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects?limit=100", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.teamID(teamID))
	}

	pr := struct {
		Projects []ProjectResponse `json:"projects"`
	}{}
	tflog.Info(ctx, "listing projects", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &pr)
	for _, p := range pr.Projects {
		p.TeamID = c.teamID(teamID)
	}
	return pr.Projects, err
}
