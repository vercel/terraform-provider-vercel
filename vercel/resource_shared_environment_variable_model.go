package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// SharedEnvironmentVariable reflects the state terraform stores internally for a project environment variable.
type SharedEnvironmentVariable struct {
	Target     types.Set    `tfsdk:"target"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
	TeamID     types.String `tfsdk:"team_id"`
	ProjectIDs types.Set    `tfsdk:"project_ids"`
	ID         types.String `tfsdk:"id"`
	Sensitive  types.Bool   `tfsdk:"sensitive"`
}

func (e *SharedEnvironmentVariable) toCreateSharedEnvironmentVariableRequest(ctx context.Context, diags diag.Diagnostics) (req client.CreateSharedEnvironmentVariableRequest, ok bool) {
	var target []string
	ds := e.Target.ElementsAs(ctx, &target, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}

	var projectIDs []string
	ds = e.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
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
	}, true
}

func (e *SharedEnvironmentVariable) toUpdateSharedEnvironmentVariableRequest(ctx context.Context, diags diag.Diagnostics) (req client.UpdateSharedEnvironmentVariableRequest, ok bool) {
	var target []string
	ds := e.Target.ElementsAs(ctx, &target, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}

	var projectIDs []string
	ds = e.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
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
	}, true
}

// convertResponseToSharedEnvironmentVariable is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToSharedEnvironmentVariable(response client.SharedEnvironmentVariableResponse, v types.String) SharedEnvironmentVariable {
	target := []attr.Value{}
	for _, t := range response.Target {
		target = append(target, types.StringValue(t))
	}

	projectIDs := []attr.Value{}
	for _, t := range response.ProjectIDs {
		projectIDs = append(projectIDs, types.StringValue(t))
	}

	value := types.StringValue(response.Value)
	if response.Type == "sensitive" {
		value = v
	}

	return SharedEnvironmentVariable{
		Target:     types.SetValueMust(types.StringType, target),
		Key:        types.StringValue(response.Key),
		Value:      value,
		ProjectIDs: types.SetValueMust(types.StringType, projectIDs),
		TeamID:     toTeamID(response.TeamID),
		ID:         types.StringValue(response.ID),
		Sensitive:  types.BoolValue(response.Type == "sensitive"),
	}
}
