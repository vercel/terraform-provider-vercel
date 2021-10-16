package client

import (
	"context"
	"net/http"
	"strings"
)

type User struct {
	UID      string `json:"uid"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Billing  struct {
		Plan string `json:"plan"`
	} `json:"billing"`
	Bio     string `json:"bio"`
	Website string `json:"website"`
}

func (c *Client) GetUser(ctx context.Context) (u User, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.vercel.com/www/user", strings.NewReader(""))
	if err != nil {
		return u, err
	}

	var response struct {
		User User `json:"user"`
	}
	err = c.doRequest(req, &response)
	if err != nil {
		return u, err
	}

	return response.User, nil
}
