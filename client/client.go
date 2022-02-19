package client

import (
	"net/http"
	"time"
)

// Client is an API wrapper, providing a high-level interface to the Vercel API.
type Client struct {
	token   string
	client  *http.Client
	baseURL string
}

func (c *Client) http() *http.Client {
	if c.client == nil {
		c.client = &http.Client{
			// Hopefully it doesn't take more than 5 minutes
			// to upload a single file for a deployment.
			Timeout: 5 * 60 * time.Second,
		}
	}

	return c.client
}

// New creates a new instace of Client for a given API token.
func New(token string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://api.vercel.com",
	}
}
