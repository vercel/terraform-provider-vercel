package vercel

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// ProjectSettings represents the terraform state for a nested deployment -> project_settings
// block. These are overrides specific to a single deployment.
type ProjectSettings struct {
	BuildCommand    types.String `tfsdk:"build_command"`
	Framework       types.String `tfsdk:"framework"`
	InstallCommand  types.String `tfsdk:"install_command"`
	OutputDirectory types.String `tfsdk:"output_directory"`
	RootDirectory   types.String `tfsdk:"root_directory"`
}

// Deployment represents the terraform state for a deployment resource.
type Deployment struct {
	Domains         types.List       `tfsdk:"domains"`
	Environment     types.Map        `tfsdk:"environment"`
	Files           types.Map        `tfsdk:"files"`
	ID              types.String     `tfsdk:"id"`
	Production      types.Bool       `tfsdk:"production"`
	ProjectID       types.String     `tfsdk:"project_id"`
	PathPrefix      types.String     `tfsdk:"path_prefix"`
	ProjectSettings *ProjectSettings `tfsdk:"project_settings"`
	TeamID          types.String     `tfsdk:"team_id"`
	URL             types.String     `tfsdk:"url"`
	DeleteOnDestroy types.Bool       `tfsdk:"delete_on_destroy"`
	Ref             types.String     `tfsdk:"ref"`
}

// setIfNotUnknown is a helper function to set a value in a map if it is not unknown.
// Null values are set as nil, and actual values are set directly.
func setIfNotUnknown(m map[string]interface{}, v types.String, name string) {
	if v.Null {
		m[name] = nil
	}
	if v.Value != "" {
		m[name] = &v.Value
	}
}

// toRequest takes a set of ProjectSettings and converts them into the required
// format for a CreateDeploymentRequest.
func (p *ProjectSettings) toRequest() map[string]interface{} {
	res := map[string]interface{}{
		/* Source files outside the root directory are required
		 * for a monorepo style codebase. This allows a root_directory
		 * to be set, but enables navigating upwards into a parent workspace.
		 *
		 * Surprisngly, even though this is the default setting for a project,
		 * it has to be explicitly passed for each request.
		 */
		"sourceFilesOutsideRootDirectory": true,
	}
	if p == nil {
		return res
	}

	setIfNotUnknown(res, p.BuildCommand, "buildCommand")
	setIfNotUnknown(res, p.Framework, "framework")
	setIfNotUnknown(res, p.InstallCommand, "installCommand")
	setIfNotUnknown(res, p.OutputDirectory, "outputDirectory")

	if p.RootDirectory.Null {
		res["rootDirectory"] = nil
	}

	return res
}

// fillStringNull is used to populate unknown resource values within state. Unknown values
// are coerced into null values. Explicitly set values are left unchanged.
func fillStringNull(t types.String) types.String {
	return types.String{
		Null:  t.Null || t.Unknown,
		Value: t.Value,
	}
}

// fillNulls takes a ProjectSettings and ensures that none of the values are unknown.
// Any unknown values are instead converted to nulls.
func (p *ProjectSettings) fillNulls() *ProjectSettings {
	if p == nil {
		return nil
	}
	return &ProjectSettings{
		BuildCommand:    fillStringNull(p.BuildCommand),
		Framework:       fillStringNull(p.Framework),
		InstallCommand:  fillStringNull(p.InstallCommand),
		OutputDirectory: fillStringNull(p.OutputDirectory),
		RootDirectory:   fillStringNull(p.RootDirectory),
	}
}

/*
 * The files uploaded to Vercel need to have some minor adjustments:
 * - Legacy behaviour was that any upward navigation ("../") was stripped from the
 * start of a file path.
 * - Newer behaviour introduced a `path_prefix` that could be specified, that would
 * control what part of a relative path to files should be removed prior to uploading
 * into Vercel.
 * - We want to support this regardless of path separator, the simplest way to do
 * this is to ensure all paths are converted to forward slashes, and settings should
 * be specified using forward slashes.
 * See https://github.com/vercel/terraform-provider-vercel/issues/14#issuecomment-1103973603
 * for additional context on the first two points.
 */
func normaliseFilename(filename string, pathPrefix types.String) string {
	filename = filepath.ToSlash(filename)
	if pathPrefix.Unknown || pathPrefix.Null {
		for strings.HasPrefix(filename, "../") {
			return strings.TrimPrefix(filename, "../")
		}
	}

	return strings.TrimPrefix(filename, filepath.ToSlash(pathPrefix.Value))
}

// getFiles is a helper for turning the terraform deployment state into a set of client.DeploymentFile
// structs, ready to hit the API with. It also returns a map of files by sha, which is used to quickly
// look up any missing SHAs from the create deployment resposnse.
func getFiles(unparsedFiles map[string]string, pathPrefix types.String) ([]client.DeploymentFile, map[string]client.DeploymentFile, error) {
	var files []client.DeploymentFile
	filesBySha := map[string]client.DeploymentFile{}

	for filename, rawSizeAndSha := range unparsedFiles {
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
			File: normaliseFilename(filename, pathPrefix),
			Sha:  sha,
			Size: size,
		}
		files = append(files, file)

		/* The API can return a set of missing files. When this happens, we want the path name
		 * complete with the original, untrimmed prefix. This also needs to use the hosts
		 * path separator. This is so we can read the file.
		 */
		filesBySha[sha] = client.DeploymentFile{
			File: filename,
			Sha:  sha,
			Size: size,
		}
	}
	return files, filesBySha, nil
}

// convertResponseToDeployment is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from the existing deployment state are used.
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

	if plan.Files.Unknown || plan.Files.Null {
		plan.Files.Unknown = false
		plan.Files.Null = true
	}

	return Deployment{
		Domains: types.List{
			ElemType: types.StringType,
			Elems:    domains,
		},
		TeamID:          toTeamID(response.TeamID),
		Environment:     plan.Environment,
		ProjectID:       types.String{Value: response.ProjectID},
		ID:              types.String{Value: response.ID},
		URL:             types.String{Value: response.URL},
		Production:      production,
		Files:           plan.Files,
		PathPrefix:      fillStringNull(plan.PathPrefix),
		ProjectSettings: plan.ProjectSettings.fillNulls(),
		DeleteOnDestroy: plan.DeleteOnDestroy,
		Ref:             types.String{Value: response.GitSource.Ref, Unknown: false, Null: response.GitSource.Ref == ""},
	}
}
