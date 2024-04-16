package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) GetSharedEnvironmentVariable(ctx context.Context, teamID, envID string) (e SharedEnvironmentVariableResponse, err error) {
	url := fmt.Sprintf("%s/v1/env/%s", c.baseURL, envID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting shared environment variable", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &e)
	e.TeamID = c.teamID(teamID)
	return e, err
}
