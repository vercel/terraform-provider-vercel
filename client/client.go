package client

import (
	"context"
	"net/http"
	"time"
)

// Client is an API wrapper, providing a high-level interface to the Vercel API.
type Client struct {
	token   string
	client  *http.Client
	team    Team
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

func (c *Client) WithTeam(team Team) *Client {
	c.team = team
	return c
}

func (c *Client) Team(ctx context.Context, teamID string) (Team, error) {
	if teamID != "" {
		return c.GetTeam(ctx, teamID)
	}
	return c.team, nil
}

// TeamID is a helper method to return one of two values based on specificity.
// It will return an explicitly passed TeamID if it is defined. If not defined,
// it will fall back to the TeamID configured on the client.
func (c *Client) TeamID(teamID string) string {
	if teamID != "" {
		return teamID
	}
	return c.team.ID
}
