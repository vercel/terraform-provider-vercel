package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) ListSharedEnvironmentVariables(ctx context.Context, teamID string) ([]SharedEnvironmentVariableResponse, error) {
	url := fmt.Sprintf("%s/v1/env/all", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "listing shared environment variables", map[string]interface{}{
		"url": url,
	})
	res := struct {
		Data []SharedEnvironmentVariableResponse `json:"data"`
	}{}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &res)
	for _, v := range res.Data {
		v.TeamID = c.teamID(teamID)
	}
	return res.Data, err
}
