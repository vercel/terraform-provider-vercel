package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateFile will upload a file to Vercel so that it can be later used for a Deployment.
func (c *Client) CreateFile(ctx context.Context, filename, sha, content string) error {
	url := fmt.Sprintf("%s/v2/now/files", c.baseURL)
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(content),
	)
	if err != nil {
		return err
	}

	req.Header.Add("x-vercel-digest", sha)

	tflog.Trace(ctx, "uploading file", map[string]interface{}{
		"url":     url,
		"payload": mustMarshal(content),
		"sha":     sha,
	})
	err = c.doRequest(req, nil)
	return err
}
