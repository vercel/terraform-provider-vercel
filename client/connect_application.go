package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// ConnectApplication contains the public identity of a Vercel Connect application.
// OAuth credentials and runtime tokens are deliberately excluded.
type ConnectApplication struct {
	ID     string `json:"id"`
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	TeamID string `json:"-"`
}

// GetConnectApplication resolves a stable connector ID or team-scoped UID.
func (c *Client) GetConnectApplication(ctx context.Context, teamID, idOrUID string) (application ConnectApplication, err error) {
	teamID = c.TeamID(teamID)
	if teamID == "" {
		return application, fmt.Errorf("a team is required to read a Connect application")
	}
	endpoint := fmt.Sprintf("%s/v1/connect/connectors/%s?teamId=%s", c.baseURL, url.PathEscape(idOrUID), url.QueryEscape(teamID))
	err = c.doRequest(clientRequest{ctx: ctx, method: "GET", url: endpoint}, &application)
	if err != nil {
		// The endpoint also returns credential fields, so do not include raw
		// response bodies from decoding failures in Terraform diagnostics.
		var apiErr APIError
		if errors.As(err, &apiErr) {
			return application, fmt.Errorf("unable to get Connect application: %w", APIError{StatusCode: apiErr.StatusCode, Code: apiErr.Code, Message: "Connect application lookup failed"})
		}
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			return application, fmt.Errorf("unable to get Connect application: %w", requestErr)
		}
		return application, fmt.Errorf("unable to get Connect application: invalid or unreadable API response")
	}
	if application.ID == "" {
		return application, fmt.Errorf("connect application response is missing its ID")
	}
	application.TeamID = teamID
	return application, nil
}
