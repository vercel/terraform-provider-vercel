package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// User represents the authenticated Vercel user.
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// GetUser returns the user associated with the current authentication token.
func (c *Client) GetUser(ctx context.Context) (u User, err error) {
	url := fmt.Sprintf("%s/v2/user", c.baseURL)
	tflog.Info(ctx, "getting authenticated user", map[string]any{
		"url": url,
	})

	var response struct {
		User User `json:"user"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	return response.User, err
}
