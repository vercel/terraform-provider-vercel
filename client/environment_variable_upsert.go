package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type UpsertEnvironmentVariableRequest EnvironmentVariable

func (c *Client) UpsertEnvironmentVariable(ctx context.Context, projectID, teamID string, request UpsertEnvironmentVariableRequest) error {
	log.Printf("[DEBUG], %s", string(mustMarshal(request)))
	url := fmt.Sprintf("%s/v8/projects/%s/env", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil)
}
