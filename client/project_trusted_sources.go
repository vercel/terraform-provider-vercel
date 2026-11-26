package client

import (
	"context"
	"fmt"
	"net/url"
)

// ProjectTrustedSources is a complete Trusted Sources configuration. Projects
// and OIDCProviders are always sent, so an empty map clears that list.
type ProjectTrustedSources struct {
	Projects                     map[string]TrustedSourcesProject        `json:"projects"`
	OIDCProviders                map[string][]TrustedSourcesOIDCProvider `json:"oidcProviders"`
	EnableVercelCiSameRepository *bool                                   `json:"enableVercelCiSameRepository,omitempty"`
}

type projectTrustedSourcesRequest struct {
	TrustedSources *ProjectTrustedSources `json:"trustedSources"`
}

// UpdateProjectTrustedSources writes only the project's trustedSources setting.
// A nil configuration sends null, which clears it.
func (c *Client) UpdateProjectTrustedSources(ctx context.Context, projectID, teamID string, trustedSources *ProjectTrustedSources) (ProjectResponse, error) {
	endpoint := fmt.Sprintf("%s/v9/projects/%s", c.baseURL, url.PathEscape(projectID))
	if teamID = c.TeamID(teamID); teamID != "" {
		endpoint += "?teamId=" + url.QueryEscape(teamID)
	}
	if trustedSources != nil {
		if trustedSources.Projects == nil {
			trustedSources.Projects = map[string]TrustedSourcesProject{}
		}
		if trustedSources.OIDCProviders == nil {
			trustedSources.OIDCProviders = map[string][]TrustedSourcesOIDCProvider{}
		}
	}
	var response ProjectResponse
	err := c.doRequest(clientRequest{ctx: ctx, method: "PATCH", url: endpoint, body: string(mustMarshal(projectTrustedSourcesRequest{TrustedSources: trustedSources})), sensitiveResponse: true}, &response)
	response.TeamID = teamID
	return response, err
}
