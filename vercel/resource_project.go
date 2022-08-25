package vercel

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

type resourceProjectType struct{}

// GetSchema returns the schema information for a deployment resource.
func (r resourceProjectType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a Project resource.

A Project groups deployments and custom domains. To deploy on Vercel, you need to create a Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/overview).

~> Terraform currently provides both a standalone Project Environment Variable resource (a single Environment Variable), and a Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
        `,
		Attributes: map[string]tfsdk.Attribute{
			"team_id": {
				Optional:      true,
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Description:   "The team ID to add the project to.",
			},
			"name": {
				Required: true,
				Type:     types.StringType,
				Validators: []tfsdk.AttributeValidator{
					stringLengthBetween(1, 52),
					stringRegex(
						regexp.MustCompile(`^[a-z0-9\-]{0,100}$`),
						"The name of a Project can only contain up to 100 alphanumeric lowercase characters and hyphens",
					),
				},
				Description: "The desired name for the project.",
			},
			"build_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The build command for this project. If omitted, this value will be automatically detected.",
			},
			"dev_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The dev command for this project. If omitted, this value will be automatically detected.",
			},
			"ignore_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "When a commit is pushed to the Git repository that is connected with your Project, its SHA will determine if a new Build has to be issued. If the SHA was deployed before, no new Build will be issued. You can customize this behavior with a command that exits with code 1 (new Build needed) or code 0.",
			},
			"serverless_function_region": {
				Optional:    true,
				Computed:    true,
				Type:        types.StringType,
				Description: "The region on Vercel's network to which your Serverless Functions are deployed. It should be close to any data source your Serverless Function might depend on. A new Deployment is required for your changes to take effect. Please see [Vercel's documentation](https://vercel.com/docs/concepts/edge-network/regions) for a full list of regions.",
				Validators: []tfsdk.AttributeValidator{
					stringOneOf(
						"arn1",
						"bom1",
						"cdg1",
						"cle1",
						"dub1",
						"fra1",
						"gru1",
						"hkg1",
						"hnd1",
						"iad1",
						"icn1",
						"lhr1",
						"pdx1",
						"sfo1",
						"sin1",
						"syd1",
					),
				},
			},
			"environment": {
				Description: "A set of Environment Variables that should be configured for the project.",
				Optional:    true,
				Attributes: tfsdk.SetNestedAttributes(map[string]tfsdk.Attribute{
					"target": {
						Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
						Type: types.SetType{
							ElemType: types.StringType,
						},
						Validators: []tfsdk.AttributeValidator{
							stringSetItemsIn("production", "preview", "development"),
						},
						Required: true,
					},
					"git_branch": {
						Description: "The git branch of the Environment Variable.",
						Type:        types.StringType,
						Optional:    true,
					},
					"key": {
						Description: "The name of the Environment Variable.",
						Type:        types.StringType,
						Required:    true,
					},
					"value": {
						Description: "The value of the Environment Variable.",
						Type:        types.StringType,
						Required:    true,
						Sensitive:   true,
					},
					"id": {
						Description:   "The ID of the Environment Variable.",
						Type:          types.StringType,
						PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown()},
						Computed:      true,
					},
				}),
			},
			"framework": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The framework that is being used for this project. If omitted, no framework is selected.",
				Validators: []tfsdk.AttributeValidator{
					validateFramework(),
				},
			},
			"git_repository": {
				Description:   "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed. This requires the corresponding Vercel for [Github](https://vercel.com/docs/concepts/git/vercel-for-github), [Gitlab](https://vercel.com/docs/concepts/git/vercel-for-gitlab) or [Bitbucket](https://vercel.com/docs/concepts/git/vercel-for-bitbucket) plugins to be installed.",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type": {
						Description: "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
						Type:        types.StringType,
						Required:    true,
						Validators: []tfsdk.AttributeValidator{
							stringOneOf("github", "gitlab", "bitbucket"),
						},
						PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
					},
					"repo": {
						Description:   "The name of the git repository. For example: `vercel/next.js`.",
						Type:          types.StringType,
						Required:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
					},
				}),
			},
			"id": {
				Computed:      true,
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown()},
			},
			"install_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The install command for this project. If omitted, this value will be automatically detected.",
			},
			"output_directory": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The output directory of the project. If omitted, this value will be automatically detected.",
			},
			"public_source": {
				Optional:    true,
				Type:        types.BoolType,
				Description: "By default, visitors to the `/_logs` and `/_src` paths of your Production and Preview Deployments must log in with Vercel (requires being a member of your team) to see the Source, Logs and Deployment Status of your project. Setting `public_source` to `true` disables this behaviour, meaning the Source, Logs and Deployment Status can be publicly viewed.",
			},
			"root_directory": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The name of a directory or relative path to the source code of your project. If omitted, it will default to the project root.",
			},
		},
	}, nil
}

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceProjectType) NewResource(_ context.Context, p provider.Provider) (resource.Resource, diag.Diagnostics) {
	return resourceProject{
		p: *(p.(*vercelProvider)),
	}, nil
}

type resourceProject struct {
	p vercelProvider
}

// Create will create a project within Vercel by calling the Vercel API.
// This is called automatically by the provider when a new resource should be created.
func (r resourceProject) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var plan Project
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := plan.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	out, err := r.p.client.CreateProject(ctx, plan.TeamID.Value, plan.toCreateProjectRequest(environment))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProject(out, plan.coercedFields(), plan.Environment)
	tflog.Trace(ctx, "created project", map[string]interface{}{
		"team_id":    result.TeamID.Value,
		"project_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a project from the vercel API and provide terraform with information about it.
// It is called by the provider whenever values should be read to update state.
func (r resourceProject) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetProject(ctx, state.ID.Value, state.TeamID.Value, !state.Environment.Null)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.ID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProject(out, state.coercedFields(), state.Environment)
	tflog.Trace(ctx, "read project", map[string]interface{}{
		"team_id":    result.TeamID.Value,
		"project_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// containsEnvVar is a helper function for working out whether a specific environment variable
// is present within a slice. It ensures that all properties of the environment variable match.
func containsEnvVar(env []EnvironmentItem, v EnvironmentItem) bool {
	for _, e := range env {
		if e.Key == v.Key &&
			e.Value == v.Value &&
			e.GitBranch == v.GitBranch &&
			len(e.Target) == len(v.Target) {
			for i, t := range e.Target {
				if t != v.Target[i] {
					continue
				}
			}
			return true
		}
	}
	return false
}

// diffEnvVars is used to determine the set of environment variables that need to be updated,
// and the set of environment variables that need to be removed.
func diffEnvVars(oldVars, newVars []EnvironmentItem) (toCreate, toRemove []EnvironmentItem) {
	toRemove = []EnvironmentItem{}
	toCreate = []EnvironmentItem{}
	for _, e := range oldVars {
		if !containsEnvVar(newVars, e) {
			toRemove = append(toRemove, e)
		}
	}
	for _, e := range newVars {
		if !containsEnvVar(oldVars, e) {
			toCreate = append(toCreate, e)
		}
	}
	return toCreate, toRemove
}

// Update will update a project and it's associated environment variables via the vercel API.
// Environment variables are manually diffed and updated individually. Once the environment
// variables are all updated, the project is updated too.
func (r resourceProject) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Project
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state Project
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	/* Update the environment variables first */
	planEnvs, err := plan.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}
	stateEnvs, err := state.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables from state",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Error(ctx, "planEnvs", map[string]interface{}{
		"plan_envs":  planEnvs,
		"state_envs": stateEnvs,
	})

	toCreate, toRemove := diffEnvVars(stateEnvs, planEnvs)
	for _, v := range toRemove {
		err := r.p.client.DeleteEnvironmentVariable(ctx, state.ID.Value, state.TeamID.Value, v.ID.Value)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.Value,
					v.ID.Value,
					err,
				),
			)
			return
		}
		tflog.Trace(ctx, "deleted environment variable", map[string]interface{}{
			"team_id":        plan.TeamID.Value,
			"project_id":     plan.ID.Value,
			"environment_id": v.ID.Value,
		})
	}
	for _, v := range toCreate {
		result, err := r.p.client.CreateEnvironmentVariable(
			ctx,
			v.toCreateEnvironmentVariableRequest(plan.ID.Value, plan.TeamID.Value),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not upsert environment variable %s (%s), unexpected error: %s",
					v.Key.Value,
					v.ID.Value,
					err,
				),
			)
		}
		tflog.Trace(ctx, "upserted environment variable", map[string]interface{}{
			"team_id":        plan.TeamID.Value,
			"project_id":     plan.ID.Value,
			"environment_id": result.ID,
		})
	}

	out, err := r.p.client.UpdateProject(ctx, state.ID.Value, state.TeamID.Value, plan.toUpdateProjectRequest(state.Name.Value))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			fmt.Sprintf(
				"Could not update project %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.ID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProject(out, plan.coercedFields(), plan.Environment)
	tflog.Trace(ctx, "updated project", map[string]interface{}{
		"team_id":    result.TeamID.Value,
		"project_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete a project and any associated environment variables from within terraform.
// Environment variables do not need to be explicitly deleted, as Vercel will automatically prune them.
func (r resourceProject) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.p.client.DeleteProject(ctx, state.ID.Value, state.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			fmt.Sprintf(
				"Could not delete project %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.ID.Value,
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "deleted project", map[string]interface{}{
		"team_id":    state.TeamID.Value,
		"project_id": state.ID.Value,
	})
}

// splitID is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitID(id string) (teamID, _id string, ok bool) {
	if strings.Contains(id, "/") {
		attributes := strings.Split(id, "/")
		if len(attributes) != 2 {
			return "", "", false
		}
		return attributes[0], attributes[1], true
	}
	return "", id, true
}

// ImportState takes an identifier and reads all the project information from the Vercel API.
// Note that environment variables are also read. The results are then stored in terraform state.
func (r resourceProject) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.p.client.GetProject(ctx, projectID, teamID, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not get project %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	result := convertResponseToProject(out, projectCoercedFields{
		/* As this is import, none of these fields are specified - so treat them all as Null */
		BuildCommand:    types.String{Null: true},
		DevCommand:      types.String{Null: true},
		InstallCommand:  types.String{Null: true},
		OutputDirectory: types.String{Null: true},
		PublicSource:    types.Bool{Null: true},
		TeamID:          types.String{Value: teamID, Null: teamID == ""},
	}, types.Set{Null: true})
	tflog.Trace(ctx, "imported project", map[string]interface{}{
		"team_id":    result.TeamID.Value,
		"project_id": result.ID.Value,
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
