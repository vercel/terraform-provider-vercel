package client

import (
	"context"
	"fmt"
	"net/url"
)

// UpdateShareableLinkRequest creates, rotates, or revokes an alias shareable link.
type UpdateShareableLinkRequest struct {
	AliasID    string
	TeamID     string
	TTLSeconds *int64
	Secret     string
	Revoke     bool
	Regenerate bool
}

// UpdateShareableLink mutates only a shareable-link entry on the specified alias.
func (c *Client) UpdateShareableLink(ctx context.Context, req UpdateShareableLinkRequest) (map[string]ProtectionBypass, error) {
	endpoint := fmt.Sprintf("%s/aliases/%s/protection-bypass", c.baseURL, url.PathEscape(req.AliasID))
	if teamID := c.TeamID(req.TeamID); teamID != "" {
		endpoint += "?teamId=" + url.QueryEscape(teamID)
	}
	body := struct {
		TTL    *int64            `json:"ttl,omitempty"`
		Revoke *revokeBypassBody `json:"revoke,omitempty"`
	}{TTL: req.TTLSeconds}
	if req.Revoke {
		body.Revoke = &revokeBypassBody{Secret: req.Secret, Regenerate: req.Regenerate}
	}
	var response protectionBypassResponse
	err := c.doRequest(clientRequest{ctx: ctx, method: "PATCH", url: endpoint, body: string(mustMarshal(body)), sensitiveResponse: true}, &response)
	return response.ProtectionBypass, err
}
