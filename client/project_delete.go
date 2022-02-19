package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// DeleteProject deletes a project within Vercel. Note that there is no need to explicitly
// remove every environment variable, as these cease to exist when a project is removed.
func (c *Client) DeleteProject(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
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
