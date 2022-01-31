package vercel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

type ProjectSettings struct {
	BuildCommand    types.String `tfsdk:"build_command"`
	Framework       types.String `tfsdk:"framework"`
	InstallCommand  types.String `tfsdk:"install_command"`
	OutputDirectory types.String `tfsdk:"output_directory"`
	RootDirectory   types.String `tfsdk:"root_directory"`
}

type Deployment struct {
	Domains         types.List        `tfsdk:"domains"`
	Environment     types.Map         `tfsdk:"environment"`
	Files           map[string]string `tfsdk:"files"`
	ID              types.String      `tfsdk:"id"`
	Production      types.Bool        `tfsdk:"production"`
	ProjectID       types.String      `tfsdk:"project_id"`
	ProjectSettings *ProjectSettings  `tfsdk:"project_settings"`
	TeamID          types.String      `tfsdk:"team_id"`
	URL             types.String      `tfsdk:"url"`
}

func setIfNotUnknown(m map[string]*string, v types.String, name string) {
	if v.Null {
		m[name] = nil
	}
	if v.Value != "" {
		m[name] = &v.Value
	}
}

func (p *ProjectSettings) toRequest() map[string]*string {
	if p == nil {
		return nil
	}
	res := map[string]*string{}

	setIfNotUnknown(res, p.BuildCommand, "buildCommand")
	setIfNotUnknown(res, p.Framework, "framework")
	setIfNotUnknown(res, p.InstallCommand, "installCommand")
	setIfNotUnknown(res, p.OutputDirectory, "outputDirectory")
	setIfNotUnknown(res, p.RootDirectory, "rootDirectory")

	return res
}

func (p *ProjectSettings) fillNulls() *ProjectSettings {
	if p == nil {
		return nil
	}
	return &ProjectSettings{
		BuildCommand:    types.String{Null: p.BuildCommand.Null || p.BuildCommand.Unknown, Value: p.BuildCommand.Value},
		Framework:       types.String{Null: p.Framework.Null || p.Framework.Unknown, Value: p.Framework.Value},
		InstallCommand:  types.String{Null: p.InstallCommand.Null || p.InstallCommand.Unknown, Value: p.InstallCommand.Value},
		OutputDirectory: types.String{Null: p.OutputDirectory.Null || p.OutputDirectory.Unknown, Value: p.OutputDirectory.Value},
		RootDirectory:   types.String{Null: p.RootDirectory.Null || p.RootDirectory.Unknown, Value: p.RootDirectory.Value},
	}
}

func (d *Deployment) getFiles() ([]client.DeploymentFile, map[string]client.DeploymentFile, error) {
	var files []client.DeploymentFile
	filesBySha := map[string]client.DeploymentFile{}
	for filename, rawSizeAndSha := range d.Files {
		sizeSha := strings.Split(rawSizeAndSha, "~")
		if len(sizeSha) != 2 {
			return nil, nil, fmt.Errorf("expected file to have format `filename: size~sha`, but could not parse")
		}
		size, err := strconv.Atoi(sizeSha[0])
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse file size: %w", err)
		}
		sha := sizeSha[1]

		file := client.DeploymentFile{
			File: filename,
			Sha:  sha,
			Size: size,
		}
		files = append(files, file)
		filesBySha[sha] = file
	}
	return files, filesBySha, nil
}

func convertResponseToDeployment(response client.DeploymentResponse, plan Deployment) Deployment {
	production := types.Bool{Value: false}
	/*
	 * TODO - the first deployment to a new project is currently _always_ a
	 * production deployment, even if you ask it to be a preview deployment.
	 * In order to terraform complaining about an inconsistent output, we should only set
	 * the state back if it matches what we expect. The third part of this
	 * conditional ensures this, but can be removed if the behaviour is changed.
	 * see:
	 * https://github.com/vercel/customer-issues/issues/178#issuecomment-1012062345 and
	 * https://vercel.slack.com/archives/C01A2M9R8RZ/p1639594164360300
	 * for more context.
	 */
	if response.Target != nil && *response.Target == "production" && (plan.Production.Value || plan.Production.Unknown) {
		production.Value = true
	}

	var domains []attr.Value
	for _, a := range response.Aliases {
		domains = append(domains, types.String{Value: a})
	}

	if plan.Environment.Unknown || plan.Environment.Null {
		plan.Environment.Unknown = false
		plan.Environment.Null = true
	}

	return Deployment{
		Domains: types.List{
			ElemType: types.StringType,
			Elems:    domains,
		},
		TeamID:          plan.TeamID,
		Environment:     plan.Environment,
		ProjectID:       types.String{Value: response.ProjectID},
		ID:              types.String{Value: response.ID},
		URL:             types.String{Value: response.URL},
		Production:      production,
		Files:           plan.Files,
		ProjectSettings: plan.ProjectSettings.fillNulls(),
	}
}
