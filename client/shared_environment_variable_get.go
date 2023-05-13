package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) GetSharedEnvironmentVariable(ctx context.Context, teamID, envID string) (e SharedEnvironmentVariableResponse, err error) {
	url := fmt.Sprintf("%s/v1/env/%s", c.baseURL, envID)
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
		return e, err
	}
	tflog.Trace(ctx, "getting shared environment variable", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &e)
	e.TeamID = c.teamID(teamID)
	return e, err
}
