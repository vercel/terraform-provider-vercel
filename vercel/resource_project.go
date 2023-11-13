package vercel

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

var (
	_ resource.Resource              = &projectResource{}
	_ resource.ResourceWithConfigure = &projectResource{}
)

func newProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *client.Client
}

func (r *projectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Schema returns the schema information for a deployment resource.
func (r *projectResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project resource.

A Project groups deployments and custom domains. To deploy on Vercel, you need to create a Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/overview).

~> Terraform currently provides both a standalone Project Environment Variable resource (a single Environment Variable), and a Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
        `,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
				Description:   "The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringLengthBetween(1, 52),
					stringRegex(
						regexp.MustCompile(`^[a-z0-9\-]{0,100}$`),
						"The name of a Project can only contain up to 100 alphanumeric lowercase characters and hyphens",
					),
				},
				Description: "The desired name for the project.",
			},
			"build_command": schema.StringAttribute{
				Optional:    true,
				Description: "The build command for this project. If omitted, this value will be automatically detected.",
			},
			"dev_command": schema.StringAttribute{
				Optional:    true,
				Description: "The dev command for this project. If omitted, this value will be automatically detected.",
			},
			"ignore_command": schema.StringAttribute{
				Optional:    true,
				Description: "When a commit is pushed to the Git repository that is connected with your Project, its SHA will determine if a new Build has to be issued. If the SHA was deployed before, no new Build will be issued. You can customize this behavior with a command that exits with code 1 (new Build needed) or code 0.",
			},
			"serverless_function_region": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The region on Vercel's network to which your Serverless Functions are deployed. It should be close to any data source your Serverless Function might depend on. A new Deployment is required for your changes to take effect. Please see [Vercel's documentation](https://vercel.com/docs/concepts/edge-network/regions) for a full list of regions.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					validateServerlessFunctionRegion(),
				},
			},
			"environment": schema.SetNestedAttribute{
				Description: "A set of Environment Variables that should be configured for the project.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.SetAttribute{
							Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
							ElementType: types.StringType,
							Validators: []validator.Set{
								stringSetItemsIn("production", "preview", "development"),
								stringSetMinCount(1),
							},
							Required: true,
						},
						"git_branch": schema.StringAttribute{
							Description: "The git branch of the Environment Variable.",
							Optional:    true,
						},
						"key": schema.StringAttribute{
							Description: "The name of the Environment Variable.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the Environment Variable.",
							Required:    true,
							Sensitive:   true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the Environment Variable.",
							Computed:    true,
						},
					},
				},
			},
			"framework": schema.StringAttribute{
				Optional:    true,
				Description: "The framework that is being used for this project. If omitted, no framework is selected.",
				Validators: []validator.String{
					validateFramework(),
				},
			},
			"git_repository": schema.SingleNestedAttribute{
				Description:   "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed. This requires the corresponding Vercel for [Github](https://vercel.com/docs/concepts/git/vercel-for-github), [Gitlab](https://vercel.com/docs/concepts/git/vercel-for-gitlab) or [Bitbucket](https://vercel.com/docs/concepts/git/vercel-for-bitbucket) plugins to be installed.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace(), objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
						Required:    true,
						Validators: []validator.String{
							stringOneOf("github", "gitlab", "bitbucket"),
						},
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"repo": schema.StringAttribute{
						Description:   "The name of the git repository. For example: `vercel/next.js`.",
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"production_branch": schema.StringAttribute{
						Description: "By default, every commit pushed to the main branch will trigger a Production Deployment instead of the usual Preview Deployment. You can switch to a different branch here.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"vercel_authentication": schema.SingleNestedAttribute{
				Description: "Ensures visitors to your Preview Deployments are logged into Vercel and have a minimum of Viewer access on your team.",
				Optional:    true,
				Computed:    true,
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"protect_production": types.BoolType,
					},
					map[string]attr.Value{
						"protect_production": types.BoolValue(false),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"protect_production": schema.BoolAttribute{
						Description: "If true, production deployments will also be protected",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			"password_protection": schema.SingleNestedAttribute{
				Description: "Ensures visitors of your Preview Deployments must enter a password in order to gain access.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"password": schema.StringAttribute{
						Description: "The password that visitors must enter to gain access to your Preview Deployments. Drift detection is not possible for this field.",
						Required:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringLengthBetween(1, 72),
						},
					},
					"protect_production": schema.BoolAttribute{
						Description: "If true, production deployments will also be protected",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"install_command": schema.StringAttribute{
				Optional:    true,
				Description: "The install command for this project. If omitted, this value will be automatically detected.",
			},
			"output_directory": schema.StringAttribute{
				Optional:    true,
				Description: "The output directory of the project. If omitted, this value will be automatically detected.",
			},
			"public_source": schema.BoolAttribute{
				Optional:    true,
				Description: "By default, visitors to the `/_logs` and `/_src` paths of your Production and Preview Deployments must log in with Vercel (requires being a member of your team) to see the Source, Logs and Deployment Status of your project. Setting `public_source` to `true` disables this behaviour, meaning the Source, Logs and Deployment Status can be publicly viewed.",
			},
			"root_directory": schema.StringAttribute{
				Optional:    true,
				Description: "The name of a directory or relative path to the source code of your project. If omitted, it will default to the project root.",
			},
			"protection_bypass_for_automation": schema.BoolAttribute{
				Optional:    true,
				Description: "Allow automation services to bypass Vercel Authentication and Password Protection for both Preview and Production Deployments on this project when using an HTTP header named `x-vercel-protection-bypass` with a value of the `password_protection_for_automation_secret` field.",
			},
			"protection_bypass_for_automation_secret": schema.StringAttribute{
				Computed:    true,
				Description: "If `protection_bypass_for_automation` is enabled, use this value in the `x-vercel-protection-bypass` header to bypass Vercel Authentication and Password Protection for both Preview and Production Deployments.",
			},
		},
	}
}

// Create will create a project within Vercel by calling the Vercel API.
// This is called automatically by the provider when a new resource should be created.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	out, err := r.client.CreateProject(ctx, plan.TeamID.ValueString(), plan.toCreateProjectRequest(environment))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProject(out, plan)
	tflog.Trace(ctx, "created project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.PasswordProtection != nil || plan.VercelAuthentication != nil {
		out, err = r.client.UpdateProject(ctx, result.ID.ValueString(), plan.TeamID.ValueString(), plan.toUpdateProjectRequest(plan.Name.ValueString()), !plan.Environment.IsNull())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project as part of creating project",
				"Could not update project, unexpected error: "+err.Error(),
			)
			return
		}

		result = convertResponseToProject(out, plan)
		tflog.Trace(ctx, "updated newly created project", map[string]interface{}{
			"team_id":    result.TeamID.ValueString(),
			"project_id": result.ID.ValueString(),
		})
		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if plan.ProtectionBypassForAutomation.ValueBool() {
		protectionBypassSecret, err := r.client.UpdateProtectionBypassForAutomation(ctx, client.UpdateProtectionBypassForAutomationRequest{
			ProjectID: result.ID.ValueString(),
			TeamID:    result.TeamID.ValueString(),
			NewValue:  true,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding protection bypass for automation",
				"Failed to create project, an error occurred adding Protection Bypass For Automation: "+err.Error(),
			)
			return
		}
		result.ProtectionBypassForAutomationSecret = types.StringValue(protectionBypassSecret)
		result.ProtectionBypassForAutomation = types.BoolValue(true)
		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if plan.GitRepository == nil || plan.GitRepository.ProductionBranch.IsNull() || plan.GitRepository.ProductionBranch.IsUnknown() {
		return
	}

	out, err = r.client.UpdateProductionBranch(ctx, client.UpdateProductionBranchRequest{
		ProjectID: out.ID,
		Branch:    plan.GitRepository.ProductionBranch.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Failed to create project, an error occurred setting the production branch: "+err.Error(),
		)
		return
	}

	result = convertResponseToProject(out, plan)
	tflog.Trace(ctx, "updated project production branch", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a project from the vercel API and provide terraform with information about it.
// It is called by the provider whenever values should be read to update state.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProject(ctx, state.ID.ValueString(), state.TeamID.ValueString(), !state.Environment.IsNull())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProject(out, state)
	tflog.Trace(ctx, "read project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
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
		if e.ID == v.ID {
			// we can rely on this because we use a known ID from state.
			// and new env vars have no ID
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
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	tflog.Trace(ctx, "planEnvs", map[string]interface{}{
		"plan_envs":  planEnvs,
		"state_envs": stateEnvs,
	})

	toCreate, toRemove := diffEnvVars(stateEnvs, planEnvs)
	for _, v := range toRemove {
		err := r.client.DeleteEnvironmentVariable(ctx, state.ID.ValueString(), state.TeamID.ValueString(), v.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.ValueString(),
					v.ID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Trace(ctx, "deleted environment variable", map[string]interface{}{
			"team_id":        plan.TeamID.ValueString(),
			"project_id":     plan.ID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}

	var items []client.EnvironmentVariableRequest
	for _, v := range toCreate {
		items = append(items, v.toEnvironmentVariableRequest())
	}

	if items != nil {
		err = r.client.CreateEnvironmentVariables(
			ctx,
			client.CreateEnvironmentVariablesRequest{
				ProjectID:            plan.ID.ValueString(),
				TeamID:               plan.TeamID.ValueString(),
				EnvironmentVariables: items,
			},
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not upsert environment variables for project %s, unexpected error: %s",
					plan.ID.ValueString(),
					err,
				),
			)
		}
		tflog.Trace(ctx, "upserted environment variables", map[string]interface{}{
			"team_id":    plan.TeamID.ValueString(),
			"project_id": plan.ID.ValueString(),
		})
	}

	if state.ProtectionBypassForAutomation != plan.ProtectionBypassForAutomation {
		_, err := r.client.UpdateProtectionBypassForAutomation(ctx, client.UpdateProtectionBypassForAutomationRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			NewValue:  plan.ProtectionBypassForAutomation.ValueBool(),
			Secret:    state.ProtectionBypassForAutomationSecret.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not update project %s %s, unexpected error setting Protection Bypass For Automation: %s",
					state.TeamID.ValueString(),
					state.ID.ValueString(),
					err,
				),
			)
			return
		}
	}

	out, err := r.client.UpdateProject(ctx, state.ID.ValueString(), state.TeamID.ValueString(), plan.toUpdateProjectRequest(state.Name.ValueString()), !plan.Environment.IsNull())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			fmt.Sprintf(
				"Could not update project %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	if plan.GitRepository != nil && !plan.GitRepository.ProductionBranch.IsNull() && (state.GitRepository == nil || state.GitRepository.ProductionBranch.ValueString() != plan.GitRepository.ProductionBranch.ValueString()) {
		out, err = r.client.UpdateProductionBranch(ctx, client.UpdateProductionBranchRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			Branch:    plan.GitRepository.ProductionBranch.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not update project %s %s, unexpected error: %s",
					state.TeamID.ValueString(),
					state.ID.ValueString(),
					err,
				),
			)
			return
		}
	}

	result := convertResponseToProject(out, plan)
	tflog.Trace(ctx, "updated project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete a project and any associated environment variables from within terraform.
// Environment variables do not need to be explicitly deleted, as Vercel will automatically prune them.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProject(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			fmt.Sprintf(
				"Could not delete project %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "deleted project", map[string]interface{}{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ID.ValueString(),
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
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetProject(ctx, projectID, teamID, false)
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

	result := convertResponseToProject(out, nullProject)
	tflog.Trace(ctx, "imported project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
