package client

import (
	"net/http"
	"time"
)

// Client is an API wrapper, providing a high-level interface to the Vercel API.
type Client struct {
	token   string
	client  *http.Client
	_teamID string
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

func (c *Client) WithTeamID(teamID string) *Client {
	c._teamID = teamID
	return c
}

// teamID is a helper method to return one of two values based on specificity.
// It will return an explicitly passed teamID if it is defined. If not defined,
// it will fall back to the teamID configured on the client.
func (c *Client) teamID(teamID string) string {
	if teamID != "" {
		return teamID
	}
	return c._teamID
}
