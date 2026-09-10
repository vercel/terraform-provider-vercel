package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// importProjectTeam resolves ownership before reading project settings whose APIs
// do not return their owner. Explicit import and provider team scopes take priority.
func importProjectTeam(ctx context.Context, c *client.Client, projectID, teamID string, resp *resource.ImportStateResponse) (string, bool) {
	if teamID = c.TeamID(teamID); teamID != "" {
		return teamID, true
	}
	project, err := c.GetProject(ctx, projectID, "")
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ownership", fmt.Sprintf("Could not read project %s to determine its team: %s", projectID, err))
		return "", false
	}
	if project.AccountID == "" && project.TeamID == "" {
		resp.Diagnostics.AddError("Error resolving project ownership", fmt.Sprintf("Project %s did not return its owner. Include team_id in the import ID or configure a provider team.", projectID))
		return "", false
	}
	return project.TeamID, true
}
