package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// ProjectEnvironmentVariable reflects the state terraform stores internally for a project environment variable.
type ProjectEnvironmentVariable struct {
	Target    []types.String `tfsdk:"target"`
	GitBranch types.String   `tfsdk:"git_branch"`
	Key       types.String   `tfsdk:"key"`
	Value     types.String   `tfsdk:"value"`
	TeamID    types.String   `tfsdk:"team_id"`
	ProjectID types.String   `tfsdk:"project_id"`
	ID        types.String   `tfsdk:"id"`
}

func (e *ProjectEnvironmentVariable) toCreateEnvironmentVariableRequest() client.CreateEnvironmentVariableRequest {
	var target []string
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}
	return client.CreateEnvironmentVariableRequest{
		Key:       e.Key.ValueString(),
		Value:     e.Value.ValueString(),
		Target:    target,
		GitBranch: toStrPointer(e.GitBranch),
		Type:      "encrypted",
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
	}
}

func (e *ProjectEnvironmentVariable) toUpdateEnvironmentVariableRequest() client.UpdateEnvironmentVariableRequest {
	var target []string
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}
	return client.UpdateEnvironmentVariableRequest{
		Key:       e.Key.ValueString(),
		Value:     e.Value.ValueString(),
		Target:    target,
		GitBranch: toStrPointer(e.GitBranch),
		Type:      "encrypted",
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
		EnvID:     e.ID.ValueString(),
	}
}

// convertResponseToProjectEnvironmentVariable is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToProjectEnvironmentVariable(response client.EnvironmentVariable, projectID types.String) ProjectEnvironmentVariable {
	target := []types.String{}
	for _, t := range response.Target {
		target = append(target, types.StringValue(t))
	}

	return ProjectEnvironmentVariable{
		Target:    target,
		GitBranch: fromStringPointer(response.GitBranch),
		Key:       types.StringValue(response.Key),
		Value:     types.StringValue(response.Value),
		TeamID:    toTeamID(response.TeamID),
		ProjectID: projectID,
		ID:        types.StringValue(response.ID),
	}
}
