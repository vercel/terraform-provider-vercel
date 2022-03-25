package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) getEnvironmentVariables(ctx context.Context, projectID, teamID string) ([]EnvironmentVariable, error) {
	url := fmt.Sprintf("%s/v8/projects/%s/env?decrypt=true", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return nil, err
	}

	envResponse := struct {
		Env []EnvironmentVariable `json:"envs"`
	}{}
	tflog.Trace(ctx, "getting environment variable", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &envResponse)
	return envResponse.Env, err
}
