package vercel

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource               = &projectEnvironmentVariablesResource{}
	_ resource.ResourceWithConfigure  = &projectEnvironmentVariablesResource{}
	_ resource.ResourceWithModifyPlan = &projectEnvironmentVariablesResource{}
)

func newProjectEnvironmentVariablesResource() resource.Resource {
	return &projectEnvironmentVariablesResource{}
}

type projectEnvironmentVariablesResource struct {
	client *client.Client
}

func (r *projectEnvironmentVariablesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variables"
}

func (r *projectEnvironmentVariablesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a project environment variable resource.
func (r *projectEnvironmentVariablesResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a resource for managing a number of Project Environment Variables.

This resource defines multiple Environment Variables on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables).

~> Terraform currently provides this Project Environment Variables resource (multiple Environment Variables), a single Project Environment Variable Resource, and a Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variables` or `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
`,
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"variables": schema.SetNestedAttribute{
				Required:    true,
				Description: "A set of Environment Variables that should be configured for the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the Environment Variable.",
							Computed:    true,
						},
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The name of the Environment Variable.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value of the Environment Variable.",
							Sensitive:   true,
							WriteOnly:   true, // We don't want to show the value in the plan or state.
						},
						"target": schema.SetAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`. At least one of `target` or `custom_environment_ids` must be set.",
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(stringvalidator.OneOf("production", "preview", "development")),
								setvalidator.SizeAtLeast(1),
								setvalidator.AtLeastOneOf(
									path.MatchRelative().AtParent().AtName("custom_environment_ids"),
									path.MatchRelative().AtParent().AtName("target"),
								),
							},
						},
						"custom_environment_ids": schema.SetAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Description: "The IDs of Custom Environments that the Environment Variable should be present on. At least one of `target` or `custom_environment_ids` must be set.",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.AtLeastOneOf(
									path.MatchRelative().AtParent().AtName("custom_environment_ids"),
									path.MatchRelative().AtParent().AtName("target"),
								),
							},
						},
						"git_branch": schema.StringAttribute{
							Optional:    true,
							Description: "The git branch of the Environment Variable.",
						},
						"sensitive": schema.BoolAttribute{
							Description:   "Whether the Environment Variable is sensitive or not.",
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						},
						"comment": schema.StringAttribute{
							Description: "A comment explaining what the environment variable is for.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(0, 1000),
							},
						},
					},
				},
			},
		},
	}
}

// ProjectEnvironmentVariables reflects the state terraform stores internally for project environment variables.
type ProjectEnvironmentVariables struct {
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
	Variables types.Set    `tfsdk:"variables"`
}

func (p *ProjectEnvironmentVariables) environment(ctx context.Context) (EnvironmentItems, diag.Diagnostics) {
	if p.Variables.IsNull() {
		return nil, nil
	}

	var vars []EnvironmentItem
	diags := p.Variables.ElementsAs(ctx, &vars, true)
	return vars, diags
}

func (r *projectEnvironmentVariablesResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config ProjectEnvironmentVariables
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, diags := config.environment(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Should be at least one variable
	if len(environment) == 0 {
		return
	}

	// work out if there are any new env vars that are specifying sensitive = false
	var nonSensitiveEnvVars []path.Path
	for i, e := range environment {
		if e.ID.ValueString() != "" {
			continue
		}
		if e.Sensitive.IsUnknown() || e.Sensitive.IsNull() || e.Sensitive.ValueBool() {
			continue
		}
		nonSensitiveEnvVars = append(
			nonSensitiveEnvVars,
			path.Root("variables").
				AtSetValue(config.Variables.Elements()[i]).
				AtName("sensitive"),
		)
	}

	if len(nonSensitiveEnvVars) == 0 {
		return
	}

	// if sensitive is explicitly set to `false`, then validate that an env var can be created with the given
	// team sensitive environment variable policy.
	team, err := r.client.Team(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating project environment variables",
			"Could not validate project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	if team.SensitiveEnvironmentVariablePolicy == nil || *team.SensitiveEnvironmentVariablePolicy != "on" {
		// the policy isn't enabled
		return
	}

	for _, p := range nonSensitiveEnvVars {
		resp.Diagnostics.AddAttributeError(
			p,
			"Project Environment Variables Invalid",
			"This team has a policy that forces all environment variables to be sensitive. Please remove the `sensitive` field for your environment variables or set the `sensitive` field to `true` in your configuration.",
		)
	}
}

type EnvironmentItems []EnvironmentItem

func (e *EnvironmentItems) toCreateEnvironmentVariablesRequest(ctx context.Context, projectID types.String, teamID types.String) (r client.CreateEnvironmentVariablesRequest, diags diag.Diagnostics) {
	variables := []client.EnvironmentVariableRequest{}
	for _, env := range *e {
		var target []string
		diags = env.Target.ElementsAs(ctx, &target, true)
		if diags.HasError() {
			return r, diags
		}
		var customEnvironmentIDs []string
		diags = env.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
		if diags.HasError() {
			return r, diags
		}
		var envVariableType string
		if env.Sensitive.ValueBool() {
			envVariableType = "sensitive"
		} else {
			envVariableType = "encrypted"
		}
		variables = append(variables, client.EnvironmentVariableRequest{
			Key:                  env.Key.ValueString(),
			Value:                env.Value.ValueString(),
			Target:               target,
			CustomEnvironmentIDs: customEnvironmentIDs,
			Type:                 envVariableType,
			GitBranch:            env.GitBranch.ValueStringPointer(),
			Comment:              env.Comment.ValueString(),
		})
	}

	return client.CreateEnvironmentVariablesRequest{
		ProjectID:            projectID.ValueString(),
		TeamID:               teamID.ValueString(),
		EnvironmentVariables: variables,
	}, nil
}

// convertResponseToProjectEnvironmentVariables is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToProjectEnvironmentVariables(
	ctx context.Context,
	response []client.EnvironmentVariable,
	plan ProjectEnvironmentVariables,
	unchanged []EnvironmentItem,
) (ProjectEnvironmentVariables, diag.Diagnostics) {
	environment, diags := plan.environment(ctx)
	if diags.HasError() {
		return ProjectEnvironmentVariables{}, diags
	}

	var env []attr.Value
	alreadyPresent := map[string]struct{}{}
	for _, e := range response {
		var targetValue attr.Value
		if len(e.Target) > 0 {
			target := make([]attr.Value, 0, len(e.Target))
			for _, t := range e.Target {
				target = append(target, types.StringValue(t))
			}
			targetValue = types.SetValueMust(types.StringType, target)
		} else {
			targetValue = types.SetNull(types.StringType)
		}

		var customEnvIDsValue attr.Value
		if len(e.CustomEnvironmentIDs) > 0 {
			customEnvIDs := make([]attr.Value, 0, len(e.CustomEnvironmentIDs))
			for _, c := range e.CustomEnvironmentIDs {
				customEnvIDs = append(customEnvIDs, types.StringValue(c))
			}
			customEnvIDsValue = types.SetValueMust(types.StringType, customEnvIDs)
		} else {
			customEnvIDsValue = types.SetNull(types.StringType)
		}
		value := types.StringValue(e.Value)
		if e.Type == "sensitive" {
			value = types.StringNull()
		}
		if e.Decrypted != nil && !*e.Decrypted || e.Type == "sensitive" {
			for _, p := range environment {
				var target []string
				diags := p.Target.ElementsAs(ctx, &target, true)
				if diags.HasError() {
					return ProjectEnvironmentVariables{}, diags
				}
				var customEnvironmentIDs []string
				diags = p.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
				if diags.HasError() {
					return ProjectEnvironmentVariables{}, diags
				}
				if p.Key.ValueString() == e.Key && isSameStringSet(target, e.Target) && isSameStringSet(customEnvironmentIDs, e.CustomEnvironmentIDs) {
					value = p.Value
					break
				}
			}
		}

		// The Vercel API returns duplicate environment variables, so we need to filter them out.
		if _, ok := alreadyPresent[e.ID]; ok {
			continue
		}
		alreadyPresent[e.ID] = struct{}{}

		env = append(env, types.ObjectValueMust(
			envVariableElemType.AttrTypes,
			map[string]attr.Value{
				"key":                    types.StringValue(e.Key),
				"value":                  value,
				"target":                 targetValue,
				"custom_environment_ids": customEnvIDsValue,
				"git_branch":             types.StringPointerValue(e.GitBranch),
				"id":                     types.StringValue(e.ID),
				"sensitive":              types.BoolValue(e.Type == "sensitive"),
				"comment":                types.StringValue(e.Comment),
			},
		))
	}

	for _, e := range unchanged {
		env = append(env, e.toAttrValue())
	}

	return ProjectEnvironmentVariables{
		TeamID:    toTeamID(plan.TeamID.ValueString()),
		ProjectID: plan.ProjectID,
		Variables: types.SetValueMust(envVariableElemType, env),
	}, nil
}

// Create will create a new project environment variable for a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r *projectEnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectEnvironmentVariables
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project environment variables",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	envs, diags := plan.environment(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	request, diags := envs.toCreateEnvironmentVariablesRequest(ctx, plan.ProjectID, plan.TeamID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	created, err := r.client.CreateEnvironmentVariables(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment variables",
			"Could not create project environment variables, unexpected error: "+err.Error(),
		)
	}

	result, diags := convertResponseToProjectEnvironmentVariables(ctx, created, plan, nil)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Set the hash of the environment variable values in the private state.
	prefix := fmt.Sprintf("vercel_env_%s_%s_", plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	for _, env := range request.EnvironmentVariables { 
		hash := sha256.Sum256([]byte(env.Value))
		privateKey := prefix + env.Key
		resp.Private.SetKey(ctx, privateKey, hash[:])
	}

	tflog.Info(ctx, "created project environment variables", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an environment variable of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectEnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectEnvironmentVariables
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, diags := state.environment(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	existingIDs := map[string]struct{}{}
	for _, e := range existing {
		if e.ID.ValueString() != "" {
			existingIDs[e.ID.ValueString()] = struct{}{}
		}
	}
	if len(existingIDs) == 0 {
		// no existing environment variables, nothing to do
		return
	}

	envs, err := r.client.GetEnvironmentVariables(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	var toUse []client.EnvironmentVariable
	for _, e := range envs {
		if _, ok := existingIDs[e.ID]; ok {
			// This ID exists in the env vars we have already. So use it.
			toUse = append(toUse, e)
		}
	}
	for _, e := range envs {
		if _, ok := existingIDs[e.ID]; !ok {
			// The env var exists at the moment, but not in TF state (the ID isn't present).
			// Check if it has the same `key`, `target` and `custom_environment_ids` as an existing env var.
			// This detects drift for stuff like: deleting an env var and then creating it again (the ID changes).
			for _, ee := range existing {
				var target []string
				diags := ee.Target.ElementsAs(ctx, &target, true)
				if diags.HasError() {
					resp.Diagnostics.Append(diags...)
					return
				}
				var customEnvironmentIDs []string
				diags = ee.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
				if diags.HasError() {
					resp.Diagnostics.Append(diags...)
					return
				}
				if ee.Key.ValueString() == e.Key && isSameStringSet(target, e.Target) && isSameStringSet(customEnvironmentIDs, e.CustomEnvironmentIDs) {
					toUse = append(toUse, e)
				}
			}
		}
	}

	result, diags := convertResponseToProjectEnvironmentVariables(ctx, toUse, state, nil)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Info(ctx, "read project environment variables", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the project environment variable of a Vercel project state.
func (r *projectEnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectEnvironmentVariables
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectEnvironmentVariables
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateEnvs, diags := state.environment(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	planEnvs, diags := plan.environment(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	plannedEnvsByID := map[string]EnvironmentItem{}
	var toAdd EnvironmentItems
	for _, e := range planEnvs {
		if e.ID.ValueString() != "" {
			plannedEnvsByID[e.ID.ValueString()] = e
		} else {
			toAdd = append(toAdd, e)
		}
	}

	var toRemove EnvironmentItems
	var unchanged EnvironmentItems
	for _, e := range stateEnvs {
		plannedEnv, ok := plannedEnvsByID[e.ID.ValueString()]
		if !ok {
			toRemove = append(toRemove, e)
			continue
		}
		if !plannedEnv.equal(&e) {
			toRemove = append(toRemove, e)
			toAdd = append(toAdd, plannedEnv)
			continue
		}
		unchanged = append(unchanged, e)
	}

	envsFromAPI, err := r.client.GetEnvironmentVariables(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables as part of environment variable update",
			"Could not read environment variables as part of updating, unexpected error: "+err.Error(),
		)
		return
	}
	skipAdding := map[int]bool{}
	for _, e := range envsFromAPI {
		// The env var exists at the moment, but not in TF state (the ID isn't present).
		// Check if it has the same `key`, `target` and `custom_environment_ids` and value as any env var we are adding.
		// This detects drift for stuff like: deleting an env var and then creating it again (the ID changes, but
		// nothing else).
		if _, ok := plannedEnvsByID[e.ID]; !ok { // env isn't in the planned envs
			for i, ee := range toAdd { // look for a matching env var in the toAdd list
				var target []string
				diags := ee.Target.ElementsAs(ctx, &target, true)
				if diags.HasError() {
					resp.Diagnostics.Append(diags...)
					return
				}
				var customEnvironmentIDs []string
				diags = ee.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
				if diags.HasError() {
					resp.Diagnostics.Append(diags...)
					return
				}
				if ee.Key.ValueString() == e.Key && isSameStringSet(target, e.Target) && isSameStringSet(customEnvironmentIDs, e.CustomEnvironmentIDs) {
					if e.Decrypted != nil && !*e.Decrypted {
						continue // We don't know if it's value is encrypted.
					}
					if e.Type == "sensitive" {
						continue // We don't know if it's the same env var if sensitive
					}
					if e.Value != ee.Value.ValueString() {
						continue // Value mismatches, so we need to update it.
					}

					var targetValue types.Set
					if len(e.Target) > 0 {
						target := make([]attr.Value, 0, len(e.Target))
						for _, t := range e.Target {
							target = append(target, types.StringValue(t))
						}
						targetValue = types.SetValueMust(types.StringType, target)
					} else {
						targetValue = types.SetNull(types.StringType)
					}

					var customEnvIDsValue types.Set
					if len(e.CustomEnvironmentIDs) > 0 {
						customEnvIDs := make([]attr.Value, 0, len(e.CustomEnvironmentIDs))
						for _, c := range e.CustomEnvironmentIDs {
							customEnvIDs = append(customEnvIDs, types.StringValue(c))
						}
						customEnvIDsValue = types.SetValueMust(types.StringType, customEnvIDs)
					} else {
						customEnvIDsValue = types.SetNull(types.StringType)
					}
					unchanged = append(unchanged, EnvironmentItem{
						Key:                  types.StringValue(e.Key),
						Value:                types.StringValue(e.Value),
						Target:               targetValue,
						CustomEnvironmentIDs: customEnvIDsValue,
						GitBranch:            types.StringPointerValue(e.GitBranch),
						ID:                   types.StringValue(e.ID),
						Sensitive:            types.BoolValue(e.Type == "sensitive"),
						Comment:              types.StringValue(e.Comment),
					})
					skipAdding[i] = true
				}
			}
		}
	}
	var filteredToAdd EnvironmentItems
	for i, e := range toAdd {
		if _, ok := skipAdding[i]; ok {
			continue
		}
		filteredToAdd = append(filteredToAdd, e)
	}
	toAdd = filteredToAdd

	tflog.Info(ctx, "Updating environment variables", map[string]any{
		"to_remove": len(toRemove),
		"to_add":    len(toAdd),
		"unchanged": len(unchanged),
	})

	for _, v := range toRemove {
		err := r.client.DeleteEnvironmentVariable(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), v.ID.ValueString())
		if client.NotFound(err) {
			continue
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Project Environment Variables",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.ValueString(),
					v.ID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Info(ctx, "deleted environment variable", map[string]any{
			"team_id":        plan.TeamID.ValueString(),
			"project_id":     plan.ProjectID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}

	var response []client.EnvironmentVariable
	if len(toAdd) > 0 {
		if len(toRemove) > 0 {
			// Sleep a bit to ensure the environment variables are fully propagated before we try to create them
			// This is disgusting, but what you gonna do?
			time.Sleep(time.Second * 5)
		}
		request, diags := toAdd.toCreateEnvironmentVariablesRequest(ctx, plan.ProjectID, plan.TeamID)

		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		response, err = r.client.CreateEnvironmentVariables(ctx, request)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project environment variables",
				"Could not update project environment variable, unexpected error: "+err.Error(),
			)
			return
		}
	}

	result, diags := convertResponseToProjectEnvironmentVariables(ctx, response, plan, unchanged)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Info(ctx, "updated project environment variables", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel project environment variable.
func (r *projectEnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectEnvironmentVariables
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envs, diags := state.environment(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	for _, v := range envs {
		err := r.client.DeleteEnvironmentVariable(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), v.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Project Environment Variables",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.ValueString(),
					v.ID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Info(ctx, "deleted environment variable", map[string]any{
			"team_id":        state.TeamID.ValueString(),
			"project_id":     state.ProjectID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}
}
