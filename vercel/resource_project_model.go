package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

type Project struct {
	BuildCommand    types.String      `tfsdk:"build_command"`
	DevCommand      types.String      `tfsdk:"dev_command"`
	Environment     []EnvironmentItem `tfsdk:"environment"`
	Framework       types.String      `tfsdk:"framework"`
	GitRepository   *GitRepository    `tfsdk:"git_repository"`
	ID              types.String      `tfsdk:"id"`
	InstallCommand  types.String      `tfsdk:"install_command"`
	Name            types.String      `tfsdk:"name"`
	OutputDirectory types.String      `tfsdk:"output_directory"`
	PublicSource    types.Bool        `tfsdk:"public_source"`
	RootDirectory   types.String      `tfsdk:"root_directory"`
	TeamID          types.String      `tfsdk:"team_id"`
}

func parseEnvironment(vars []EnvironmentItem) []client.EnvironmentVariable {
	out := []client.EnvironmentVariable{}
	for _, e := range vars {
		var target []string
		for _, t := range e.Target {
			target = append(target, t.Value)
		}

		out = append(out, client.EnvironmentVariable{
			Key:    e.Key.Value,
			Value:  e.Value.Value,
			Target: target,
			Type:   "encrypted",
			ID:     e.ID.Value,
		})
	}
	return out
}

func (p *Project) toCreateProjectRequest() client.CreateProjectRequest {
	return client.CreateProjectRequest{
		Name:                 p.Name.Value,
		BuildCommand:         toStrPointer(p.BuildCommand),
		DevCommand:           toStrPointer(p.DevCommand),
		EnvironmentVariables: parseEnvironment(p.Environment),
		Framework:            toStrPointer(p.Framework),
		GitRepository:        p.GitRepository.toCreateProjectRequest(),
		InstallCommand:       toStrPointer(p.InstallCommand),
		OutputDirectory:      toStrPointer(p.OutputDirectory),
		PublicSource:         toBoolPointer(p.PublicSource),
		RootDirectory:        toStrPointer(p.RootDirectory),
	}
}

func (p *Project) toUpdateProjectRequest(oldName string) client.UpdateProjectRequest {
	var name *string = nil
	if oldName != p.Name.Value {
		name = &p.Name.Value
	}
	return client.UpdateProjectRequest{
		Name:            name,
		BuildCommand:    toStrPointer(p.BuildCommand),
		DevCommand:      toStrPointer(p.DevCommand),
		Framework:       toStrPointer(p.Framework),
		InstallCommand:  toStrPointer(p.InstallCommand),
		OutputDirectory: toStrPointer(p.OutputDirectory),
		RootDirectory:   toStrPointer(p.RootDirectory),
		PublicSource:    toBoolPointer(p.PublicSource),
	}
}

type EnvironmentItem struct {
	Target []types.String `tfsdk:"target"`
	Key    types.String   `tfsdk:"key"`
	Value  types.String   `tfsdk:"value"`
	ID     types.String   `tfsdk:"id"`
}

func (e *EnvironmentItem) toUpsertEnvironmentVariableRequest() client.UpsertEnvironmentVariableRequest {
	var target []string
	for _, t := range e.Target {
		target = append(target, t.Value)
	}
	return client.UpsertEnvironmentVariableRequest{
		Key:    e.Key.Value,
		Value:  e.Value.Value,
		Target: target,
		Type:   "encrypted",
		ID:     e.ID.Value,
	}
}

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

func convertResponseToProject(response client.ProjectResponse, tid types.String) Project {
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
			Key:    types.String{Value: e.Key},
			Value:  types.String{Value: e.Value},
			Target: target,
			ID:     types.String{Value: e.ID},
		})
	}
	teamID := types.String{Value: tid.Value}
	if tid.Unknown || tid.Null {
		teamID.Null = true
	}

	return Project{
		TeamID:          teamID,
		ID:              types.String{Value: response.ID},
		Name:            types.String{Value: response.Name},
		BuildCommand:    fromStringPointer(response.BuildCommand),
		DevCommand:      fromStringPointer(response.DevCommand),
		Framework:       fromStringPointer(response.Framework),
		InstallCommand:  fromStringPointer(response.InstallCommand),
		OutputDirectory: fromStringPointer(response.OutputDirectory),
		PublicSource:    fromBoolPointer(response.PublicSource),
		RootDirectory:   fromStringPointer(response.RootDirectory),
		GitRepository:   gr,
		Environment:     env,
	}
}
