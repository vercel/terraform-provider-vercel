package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateFileRequest defines the information needed to upload a file to Vercel.
type CreateFileRequest struct {
	Filename string
	SHA      string
	Content  string
	TeamID   string
}

// CreateFile will upload a file to Vercel so that it can be later used for a Deployment.
func (c *Client) CreateFile(ctx context.Context, request CreateFileRequest) error {
	url := fmt.Sprintf("%s/v2/now/files", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(request.Content),
	)
	if err != nil {
		return err
	}

	req.Header.Add("x-vercel-digest", request.SHA)
	req.Header.Set("Content-Type", "application/octet-stream")

	tflog.Info(ctx, "uploading file", map[string]any{
		"url": url,
		"sha": request.SHA,
	})
	err = c._doRequest(req, nil, false)
	return err
}
