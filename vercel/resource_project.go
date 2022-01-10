package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type resourceProjectType struct{}

func (r resourceProjectType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"team_id": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The ID of the team the project should be created under",
			},
			"name": {
				Required: true,
				Type:     types.StringType,
				Validators: []tfsdk.AttributeValidator{
					stringLengthBetween(1, 52),
				},
				Description: "The desired name for the project",
			},
			"build_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The build command for this project. If omitted, this value will be automatically detected",
			},
			"dev_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The dev command for this project. If omitted, this value will be automatically detected",
			},
			"environment": {
				Description: "An environment variable for the project.",
				Optional:    true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"target": {
						Description: "The environments that the environment variable should be present on. Valid targets are be either `production`, `preview`, or `development`. If omitted, the variable will exist across all targets.",
						Type: types.SetType{
							ElemType: types.StringType,
						},
						Validators: []tfsdk.AttributeValidator{
							setMinSize(1),
							stringSetItemsIn("production", "preview", "development"),
						},
						Required: true,
					},
					"key": {
						Description: "The name of the environment variable",
						Type:        types.StringType,
						Required:    true,
					},
					"value": {
						Description: "The value of the environment variable",
						Type:        types.StringType,
						Required:    true,
					},
					"id": {
						Description: "The ID of the environment variable",
						Type:        types.StringType,
						Computed:    true,
					},
				}, tfsdk.ListNestedAttributesOptions{}),
			},
			"framework": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The framework that is being used for this project. If omitted, no framework is selected",
			},
			"git_repository": {
				Description:   "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type": {
						Description:   "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
						Type:          types.StringType,
						Required:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
					},
					"repo": {
						Description:   "The name of the git repository. For example: `vercel/next.js`",
						Type:          types.StringType,
						Required:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
					},
				}),
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"install_command": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The install command for this project. If omitted, this value will be automatically detected",
			},
			"output_directory": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The output directory of the project. When null is used this value will be automatically detected",
			},
			"public_source": {
				Optional:    true,
				Type:        types.BoolType,
				Description: "Specifies whether the source code and logs of the deployments for this project should be public or not",
			},
			"root_directory": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root",
			},
		},
	}, nil
}

// New resource instance
func (r resourceProjectType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceProject{
		p: *(p.(*provider)),
	}, nil
}

type resourceProject struct {
	p provider
}

func (r resourceProject) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
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

	out, err := r.p.client.CreateProject(ctx, plan.TeamID.Value, plan.toCreateProjectRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToProject(out, plan.TeamID)
	tflog.Trace(ctx, "created project", "team_id", result.TeamID.Value, "project_id", result.ID.Value)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceProject) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetProject(ctx, state.ID.Value, state.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project %s for team %s, unexpected error: %s",
				state.ID.Value,
				state.TeamID.Value,
				err.Error(),
			),
		)
		return
	}

	result := convertResponseToProject(out, state.TeamID)
	tflog.Trace(ctx, "created project", "team_id", result.TeamID.Value, "project_id", result.ID.Value)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceProject) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
}

func (r resourceProject) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
}

func (r resourceProject) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStateNotImplemented(ctx, "", resp)
}

/*
func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Reading Project")

	project, err := c.GetProject(ctx, d.Id(), d.Get("team_id").(string))
	var apiErr client.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.Errorf("error reading project: %s", err)
	}

	return updateProjectSchema(d, project)
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.Client)
	log.Printf("[DEBUG] Deleting Project")
	err := client.DeleteProject(ctx, d.Id(), d.Get("team_id").(string))
	if err != nil {
		return diag.Errorf("error deleting project: %s", err)
	}

	d.SetId("")
	return nil
}

func getStringPointerIfChanged(d *schema.ResourceData, key string) *string {
	if d.HasChange(key) {
		v := d.Get(key).(string)
		return &v
	}
	return nil
}

func getBoolPointerIfChanged(d *schema.ResourceData, key string) *bool {
	if d.HasChange(key) {
		v := d.Get(key).(bool)
		return &v
	}
	return nil
}

func containsEnvVar(env []client.EnvironmentVariable, v client.EnvironmentVariable) bool {
	for _, e := range env {
		if e.Key == v.Key &&
			e.Value == v.Value &&
			e.Type == v.Type &&
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

func diffEnvVars(oldVars, newVars []client.EnvironmentVariable) (toUpsert, toRemove []client.EnvironmentVariable) {
	toRemove = []client.EnvironmentVariable{}
	toUpsert = []client.EnvironmentVariable{}
	for _, e := range oldVars {
		if !containsEnvVar(newVars, e) {
			toRemove = append(toRemove, e)
		}
	}
	for _, e := range newVars {
		if !containsEnvVar(oldVars, e) {
			toUpsert = append(toUpsert, e)
		}
	}
	return toUpsert, toRemove
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Updating Project")
	teamID := d.Get("team_id").(string)

	if d.HasChange("environment") {
		oldVars, newVars := d.GetChange("environment")
		toUpsert, toRemove := diffEnvVars(
			parseEnvironmentVariables(oldVars.([]interface{})),
			parseEnvironmentVariables(newVars.([]interface{})),
		)
		for _, v := range toRemove {
			err := c.DeleteEnvironmentVariable(ctx, d.Id(), teamID, v.ID)
			if err != nil {
				return diag.Errorf("error deleting environment variable: %s", err)
			}
		}
		for _, v := range toUpsert {
			err := c.UpsertEnvironmentVariable(ctx, d.Id(), teamID, client.UpsertEnvironmentVariableRequest(v))
			if err != nil {
				return diag.Errorf("error creating or updating environment variable: %s", err)
			}
		}
	}

	project, err := c.UpdateProject(ctx, d.Id(), teamID, client.UpdateProjectRequest{
		Name:            getStringPointerIfChanged(d, "name"),
		BuildCommand:    getStringPointerIfChanged(d, "build_command"),
		DevCommand:      getStringPointerIfChanged(d, "dev_command"),
		Framework:       getStringPointerIfChanged(d, "framework"),
		InstallCommand:  getStringPointerIfChanged(d, "install_command"),
		OutputDirectory: getStringPointerIfChanged(d, "output_directory"),
		RootDirectory:   getStringPointerIfChanged(d, "root_directory"),
		PublicSource:    getBoolPointerIfChanged(d, "public_source"),
	})
	if err != nil {
		return diag.Errorf("error updating project: %s", err)
	}

	return updateProjectSchema(d, project)
}
*/
