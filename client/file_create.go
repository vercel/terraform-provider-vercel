package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateFile(ctx context.Context, filename, sha, content string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/v2/now/files", c.baseURL),
		strings.NewReader(content),
	)
	if err != nil {
		return err
	}

	req.Header.Add("x-vercel-digest", sha)

	err = c.doRequest(req, nil)
	return err
}
