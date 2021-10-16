package client

import (
	"net/http"
	"time"
)

type Client struct {
	token  string
	client *http.Client
}

func (c *Client) http() *http.Client {
	if c.client == nil {
		c.client = &http.Client{
			Timeout: 120 * time.Second,
		}
	}

	return c.client
}

func New(token string) *Client {
	return &Client{
		token: token,
	}
}
