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
	Sensitive types.Bool     `tfsdk:"sensitive"`
}

func (e *ProjectEnvironmentVariable) toCreateEnvironmentVariableRequest() client.CreateEnvironmentVariableRequest {
	target := []string{}
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}
	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.CreateEnvironmentVariableRequest{
		EnvironmentVariable: client.EnvironmentVariableRequest{
			Key:       e.Key.ValueString(),
			Value:     e.Value.ValueString(),
			Target:    target,
			GitBranch: toStrPointer(e.GitBranch),
			Type:      envVariableType,
		},
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
	}
}

func (e *ProjectEnvironmentVariable) toUpdateEnvironmentVariableRequest() client.UpdateEnvironmentVariableRequest {
	target := []string{}
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}

	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.UpdateEnvironmentVariableRequest{
		Key:       e.Key.ValueString(),
		Value:     e.Value.ValueString(),
		Target:    target,
		GitBranch: toStrPointer(e.GitBranch),
		Type:      envVariableType,
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
		EnvID:     e.ID.ValueString(),
	}
}

// convertResponseToProjectEnvironmentVariable is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToProjectEnvironmentVariable(response client.EnvironmentVariable, projectID types.String, v types.String) ProjectEnvironmentVariable {
	target := []types.String{}
	for _, t := range response.Target {
		target = append(target, types.StringValue(t))
	}

	value := types.StringValue(response.Value)
	if response.Type == "sensitive" {
		value = v
	}

	return ProjectEnvironmentVariable{
		Target:    target,
		GitBranch: fromStringPointer(response.GitBranch),
		Key:       types.StringValue(response.Key),
		Value:     value,
		TeamID:    toTeamID(response.TeamID),
		ProjectID: projectID,
		ID:        types.StringValue(response.ID),
		Sensitive: types.BoolValue(response.Type == "sensitive"),
	}
}
