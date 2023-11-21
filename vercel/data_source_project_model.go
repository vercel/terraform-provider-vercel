package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Project reflects the state terraform stores internally for a project.
type ProjectDataSource struct {
	BuildCommand             types.String          `tfsdk:"build_command"`
	DevCommand               types.String          `tfsdk:"dev_command"`
	Environment              types.Set             `tfsdk:"environment"`
	Framework                types.String          `tfsdk:"framework"`
	GitRepository            *GitRepository        `tfsdk:"git_repository"`
	ID                       types.String          `tfsdk:"id"`
	IgnoreCommand            types.String          `tfsdk:"ignore_command"`
	InstallCommand           types.String          `tfsdk:"install_command"`
	Name                     types.String          `tfsdk:"name"`
	OutputDirectory          types.String          `tfsdk:"output_directory"`
	PublicSource             types.Bool            `tfsdk:"public_source"`
	RootDirectory            types.String          `tfsdk:"root_directory"`
	ServerlessFunctionRegion types.String          `tfsdk:"serverless_function_region"`
	TeamID                   types.String          `tfsdk:"team_id"`
	VercelAuthentication     *VercelAuthentication `tfsdk:"vercel_authentication"`
	PasswordProtection       *PasswordProtection   `tfsdk:"password_protection"`
	TrustedIps               *TrustedIps           `tfsdk:"trusted_ips"`
}

func convertResponseToProjectDataSource(response client.ProjectResponse, plan Project) ProjectDataSource {
	project := convertResponseToProject(response, plan)

	return ProjectDataSource{
		BuildCommand:             project.BuildCommand,
		DevCommand:               project.DevCommand,
		Environment:              project.Environment,
		Framework:                project.Framework,
		GitRepository:            project.GitRepository,
		ID:                       project.ID,
		IgnoreCommand:            project.IgnoreCommand,
		InstallCommand:           project.InstallCommand,
		Name:                     project.Name,
		OutputDirectory:          project.OutputDirectory,
		PublicSource:             project.PublicSource,
		RootDirectory:            project.RootDirectory,
		ServerlessFunctionRegion: project.ServerlessFunctionRegion,
		TeamID:                   project.TeamID,
		VercelAuthentication:     project.VercelAuthentication,
		PasswordProtection:       project.PasswordProtection,
		TrustedIps:               project.TrustedIps,
	}
}
