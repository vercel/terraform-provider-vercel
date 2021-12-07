package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) DeleteEnvironmentVariable(ctx context.Context, projectID, teamID, variableID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/env/%s", c.baseURL, projectID, variableID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil)
}
