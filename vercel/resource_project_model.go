package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Project reflects the state terraform stores internally for a project.
type Project struct {
	BuildCommand             types.String      `tfsdk:"build_command"`
	DevCommand               types.String      `tfsdk:"dev_command"`
	Environment              []EnvironmentItem `tfsdk:"environment"`
	Framework                types.String      `tfsdk:"framework"`
	GitRepository            *GitRepository    `tfsdk:"git_repository"`
	ID                       types.String      `tfsdk:"id"`
	IgnoreCommand            types.String      `tfsdk:"ignore_command"`
	InstallCommand           types.String      `tfsdk:"install_command"`
	Name                     types.String      `tfsdk:"name"`
	OutputDirectory          types.String      `tfsdk:"output_directory"`
	PublicSource             types.Bool        `tfsdk:"public_source"`
	RootDirectory            types.String      `tfsdk:"root_directory"`
	ServerlessFunctionRegion types.String      `tfsdk:"serverless_function_region"`
	TeamID                   types.String      `tfsdk:"team_id"`
}

func parseEnvironment(vars []EnvironmentItem) []client.EnvironmentVariable {
	out := []client.EnvironmentVariable{}
	for _, e := range vars {
		var target []string
		for _, t := range e.Target {
			target = append(target, t.Value)
		}

		out = append(out, client.EnvironmentVariable{
			Key:       e.Key.Value,
			Value:     e.Value.Value,
			Target:    target,
			GitBranch: toStrPointer(e.GitBranch),
			Type:      "encrypted",
			ID:        e.ID.Value,
		})
	}
	return out
}

func (p *Project) toCreateProjectRequest() client.CreateProjectRequest {
	return client.CreateProjectRequest{
		BuildCommand:                toStrPointer(p.BuildCommand),
		CommandForIgnoringBuildStep: toStrPointer(p.IgnoreCommand),
		DevCommand:                  toStrPointer(p.DevCommand),
		EnvironmentVariables:        parseEnvironment(p.Environment),
		Framework:                   toStrPointer(p.Framework),
		GitRepository:               p.GitRepository.toCreateProjectRequest(),
		InstallCommand:              toStrPointer(p.InstallCommand),
		Name:                        p.Name.Value,
		OutputDirectory:             toStrPointer(p.OutputDirectory),
		PublicSource:                toBoolPointer(p.PublicSource),
		RootDirectory:               toStrPointer(p.RootDirectory),
		ServerlessFunctionRegion:    toStrPointer(p.ServerlessFunctionRegion),
	}
}

func (p *Project) toUpdateProjectRequest(oldName string) client.UpdateProjectRequest {
	var name *string = nil
	if oldName != p.Name.Value {
		name = &p.Name.Value
	}
	return client.UpdateProjectRequest{
		BuildCommand:                toStrPointer(p.BuildCommand),
		CommandForIgnoringBuildStep: toStrPointer(p.IgnoreCommand),
		DevCommand:                  toStrPointer(p.DevCommand),
		Framework:                   toStrPointer(p.Framework),
		InstallCommand:              toStrPointer(p.InstallCommand),
		Name:                        name,
		OutputDirectory:             toStrPointer(p.OutputDirectory),
		PublicSource:                toBoolPointer(p.PublicSource),
		RootDirectory:               toStrPointer(p.RootDirectory),
		ServerlessFunctionRegion:    toStrPointer(p.ServerlessFunctionRegion),
	}
}

// EnvironmentItem reflects the state terraform stores internally for a project's environment variable.
type EnvironmentItem struct {
	Target    []types.String `tfsdk:"target"`
	GitBranch types.String   `tfsdk:"git_branch"`
	Key       types.String   `tfsdk:"key"`
	Value     types.String   `tfsdk:"value"`
	ID        types.String   `tfsdk:"id"`
}

func (e *EnvironmentItem) toUpsertEnvironmentVariableRequest() client.UpsertEnvironmentVariableRequest {
	var target []string
	for _, t := range e.Target {
		target = append(target, t.Value)
	}
	return client.UpsertEnvironmentVariableRequest{
		Key:       e.Key.Value,
		Value:     e.Value.Value,
		Target:    target,
		GitBranch: toStrPointer(e.GitBranch),
		Type:      "encrypted",
		ID:        e.ID.Value,
	}
}

// GitRepository reflects the state terraform stores internally for a nested git_repository block on a project resource.
type GitRepository struct {
	Type types.String `tfsdk:"type"`
	Repo types.String `tfsdk:"repo"`
}

func (g *GitRepository) toCreateProjectRequest() *client.GitRepository {
	if g == nil {
		return nil
	}
	return &client.GitRepository{
		Type: g.Type.Value,
		Repo: g.Repo.Value,
	}
}

/*
* In the Vercel API the following fields are coerced to null during project creation

* This causes an issue when they are specified, but falsy, as the
* terraform configuration explicitly sets a value for them, but the Vercel
* API returns a different value. This causes an inconsistent plan error.

* We avoid this issue by choosing to use values from the terraform state,
* but only if they are _explicitly stated_ *and* they are _falsy_ values
* *and* the response value was null. This is important as drift detection
* would fail to work if the value was always selected, so this is as stringent
* as possible to allow drift-detection in the majority of scenarios.

* This is implemented in the below uncoerceString and uncoerceBool functions.
 */
type projectCoercedFields struct {
	BuildCommand    types.String
	DevCommand      types.String
	InstallCommand  types.String
	OutputDirectory types.String
	PublicSource    types.Bool
	TeamID          types.String
}

func (p *Project) coercedFields() projectCoercedFields {
	return projectCoercedFields{
		BuildCommand:    p.BuildCommand,
		DevCommand:      p.DevCommand,
		InstallCommand:  p.InstallCommand,
		OutputDirectory: p.OutputDirectory,
		PublicSource:    p.PublicSource,
		TeamID:          p.TeamID,
	}
}

func uncoerceString(plan, res types.String) types.String {
	if plan.Value == "" && !plan.Null && res.Null {
		return plan
	}
	return res
}
func uncoerceBool(plan, res types.Bool) types.Bool {
	if !plan.Value && !plan.Null && res.Null {
		return plan
	}
	return res
}

func convertResponseToProject(response client.ProjectResponse, fields projectCoercedFields) Project {
	var gr *GitRepository
	if repo := response.Repository(); repo != nil {
		gr = &GitRepository{
			Type: types.String{Value: repo.Type},
			Repo: types.String{Value: repo.Repo},
		}
	}
	var env []EnvironmentItem
	for _, e := range response.EnvironmentVariables {
		target := []types.String{}
		for _, t := range e.Target {
			target = append(target, types.String{Value: t})
		}
		env = append(env, EnvironmentItem{
			Key:       types.String{Value: e.Key},
			Value:     types.String{Value: e.Value},
			Target:    target,
			GitBranch: fromStringPointer(e.GitBranch),
			ID:        types.String{Value: e.ID},
		})
	}

	return Project{
		BuildCommand:             uncoerceString(fields.BuildCommand, fromStringPointer(response.BuildCommand)),
		DevCommand:               uncoerceString(fields.DevCommand, fromStringPointer(response.DevCommand)),
		Environment:              env,
		Framework:                fromStringPointer(response.Framework),
		GitRepository:            gr,
		ID:                       types.String{Value: response.ID},
		IgnoreCommand:            fromStringPointer(response.CommandForIgnoringBuildStep),
		InstallCommand:           uncoerceString(fields.InstallCommand, fromStringPointer(response.InstallCommand)),
		Name:                     types.String{Value: response.Name},
		OutputDirectory:          uncoerceString(fields.OutputDirectory, fromStringPointer(response.OutputDirectory)),
		PublicSource:             uncoerceBool(fields.PublicSource, fromBoolPointer(response.PublicSource)),
		RootDirectory:            fromStringPointer(response.RootDirectory),
		ServerlessFunctionRegion: fromStringPointer(response.ServerlessFunctionRegion),
		TeamID:                   types.String{Value: fields.TeamID.Value, Null: fields.TeamID.Null || fields.TeamID.Unknown},
	}
}
