package client

import (
	"context"
	"fmt"
	"net/url"
)

// UpdateProjectPassport changes only Passport; nil disables it without deleting the project.
func (c *Client) UpdateProjectPassport(ctx context.Context, projectID, teamID string, passport *Passport) (ProjectResponse, error) {
	endpoint := fmt.Sprintf("%s/v9/projects/%s", c.baseURL, url.PathEscape(projectID))
	if teamID = c.TeamID(teamID); teamID != "" {
		endpoint += "?teamId=" + url.QueryEscape(teamID)
	}
	var response ProjectResponse
	err := c.doRequest(clientRequest{
		ctx: ctx, method: "PATCH", url: endpoint,
		body: string(mustMarshal(struct {
			Passport *Passport `json:"passport"`
		}{passport})),
		sensitiveResponse: true,
	}, &response)
	response.TeamID = teamID
	return response, err
}
