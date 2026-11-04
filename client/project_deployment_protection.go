package client

import (
	"context"
	"fmt"
	"net/url"
)

// ProjectProtectionTrustedIPs contains the allowlist fields supported by project PATCH.
// protectionMode is read-only in the public API contract.
type ProjectProtectionTrustedIPs struct {
	DeploymentType string             `json:"deploymentType"`
	Addresses      []TrustedIpAddress `json:"addresses"`
}

// ProjectDeploymentProtectionRequest distinguishes omitted settings from explicit nulls.
type ProjectDeploymentProtectionRequest struct {
	Authentication **VercelAuthentication           `json:"ssoProtection,omitempty"`
	Password       **PasswordProtectionWithPassword `json:"passwordProtection,omitempty"`
	TrustedIPs     **ProjectProtectionTrustedIPs    `json:"trustedIps,omitempty"`
}

// UpdateProjectDeploymentProtection changes only the selected protection settings.
func (c *Client) UpdateProjectDeploymentProtection(ctx context.Context, projectID, teamID string, request ProjectDeploymentProtectionRequest) (ProjectResponse, error) {
	endpoint := fmt.Sprintf("%s/v9/projects/%s", c.baseURL, url.PathEscape(projectID))
	if teamID = c.TeamID(teamID); teamID != "" {
		endpoint += "?teamId=" + url.QueryEscape(teamID)
	}
	var response ProjectResponse
	err := c.doRequest(clientRequest{ctx: ctx, method: "PATCH", url: endpoint, body: string(mustMarshal(request)), sensitiveResponse: true}, &response)
	response.TeamID = teamID
	return response, err
}
