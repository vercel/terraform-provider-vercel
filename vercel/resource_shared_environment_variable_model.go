package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// SharedEnvironmentVariable reflects the state terraform stores internally for a project environment variable.
type SharedEnvironmentVariable struct {
	Target     []types.String `tfsdk:"target"`
	Key        types.String   `tfsdk:"key"`
	Value      types.String   `tfsdk:"value"`
	TeamID     types.String   `tfsdk:"team_id"`
	ProjectIDs []types.String `tfsdk:"project_ids"`
	ID         types.String   `tfsdk:"id"`
	Sensitive  types.Bool     `tfsdk:"sensitive"`
}

func (e *SharedEnvironmentVariable) toCreateSharedEnvironmentVariableRequest() client.CreateSharedEnvironmentVariableRequest {
	target := []string{}
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}
	projectIDs := []string{}
	for _, t := range e.ProjectIDs {
		projectIDs = append(projectIDs, t.ValueString())
	}

	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.CreateSharedEnvironmentVariableRequest{
		EnvironmentVariable: client.SharedEnvironmentVariableRequest{
			Target:     target,
			Type:       envVariableType,
			ProjectIDs: projectIDs,
			EnvironmentVariables: []client.SharedEnvVarRequest{
				{
					Key:   e.Key.ValueString(),
					Value: e.Value.ValueString(),
				},
			},
		},
		TeamID: e.TeamID.ValueString(),
	}
}

func (e *SharedEnvironmentVariable) toUpdateSharedEnvironmentVariableRequest() client.UpdateSharedEnvironmentVariableRequest {
	target := []string{}
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}
	projectIDs := []string{}
	for _, t := range e.ProjectIDs {
		projectIDs = append(projectIDs, t.ValueString())
	}
	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}
	return client.UpdateSharedEnvironmentVariableRequest{
		Value:      e.Value.ValueString(),
		Target:     target,
		Type:       envVariableType,
		TeamID:     e.TeamID.ValueString(),
		EnvID:      e.ID.ValueString(),
		ProjectIDs: projectIDs,
	}
}

// convertResponseToSharedEnvironmentVariable is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToSharedEnvironmentVariable(response client.SharedEnvironmentVariableResponse) SharedEnvironmentVariable {
	target := []types.String{}
	for _, t := range response.Target {
		target = append(target, types.StringValue(t))
	}

	project_ids := []types.String{}
	for _, t := range response.ProjectIDs {
		project_ids = append(project_ids, types.StringValue(t))
	}

	return SharedEnvironmentVariable{
		Target:     target,
		Key:        types.StringValue(response.Key),
		Value:      types.StringValue(response.Value),
		ProjectIDs: project_ids,
		TeamID:     toTeamID(response.TeamID),
		ID:         types.StringValue(response.ID),
		Sensitive:  types.BoolValue(response.Type == "sensitive"),
	}
}
