package vercel

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vercel/terraform-provider-vercel/client"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		DeleteContext: resourceProjectDelete,
		UpdateContext: resourceProjectUpdate,
		Schema: map[string]*schema.Schema{
			"team_id": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The ID of the team the project should be created under",
			},
			"name": {
				Required:     true,
				Type:         schema.TypeString,
				ValidateFunc: validation.StringLenBetween(1, 52),
				Description:  "The desired name for the project",
			},
			"build_command": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The build command for this project. If omitted, this value will be automatically detected",
			},
			"dev_command": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The dev command for this project. If omitted, this value will be automatically detected",
			},
			"production_environment_variables": {
				Optional: true,
				Type:     schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Collection of environment variables the Project will use on production deployments",
			},
			"preview_environment_variables": {
				Optional: true,
				Type:     schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Collection of environment variables the Project will use on preview deployments",
			},
			"framework": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The framework that is being used for this project. If omitted, no framework is selected",
			},
			"git_repository": {
				Description: "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed",
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Description: "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"repo": {
							Description: "The name of the git repository. For example: `vercel/next.js`",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"id": {
				Computed: true,
				Type:     schema.TypeString,
			},
			"install_command": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The install command for this project. If omitted, this value will be automatically detected",
			},
			"output_directory": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The output directory of the project. When null is used this value will be automatically detected",
			},
			"public_source": {
				Optional:    true,
				Default:     false,
				Type:        schema.TypeBool,
				Description: "Specifies whether the source code and logs of the deployments for this project should be public or not",
			},
			"root_directory": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root",
			},
		},
	}
}

func getStringPointer(d *schema.ResourceData, key string) *string {
	if v, ok := d.GetOk(key); ok {
		value := v.(string)
		return &value
	}
	return nil
}

func getEnvVariables(d *schema.ResourceData) []client.EnvironmentVariable {
	envVars := []client.EnvironmentVariable{}
	if v, ok := d.GetOk("production_environment_variables"); ok {
		for key, value := range v.(map[string]interface{}) {
			envVars = append(envVars, client.EnvironmentVariable{
				Key:    key,
				Value:  value.(string),
				Target: "production",
			})
		}
	}
	if v, ok := d.GetOk("preview_environment_variables"); ok {
		for key, value := range v.(map[string]interface{}) {
			envVars = append(envVars, client.EnvironmentVariable{
				Key:    key,
				Value:  value.(string),
				Target: "preview",
			})
		}
	}
	return envVars
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Creating Project")

	out, err := c.CreateProject(ctx, d.Get("team_id").(string), client.CreateProjectRequest{
		Name:                 d.Get("name").(string),
		PublicSource:         d.Get("public_source").(bool),
		EnvironmentVariables: getEnvVariables(d),
		BuildCommand:         getStringPointer(d, "build_command"),
		DevCommand:           getStringPointer(d, "dev_command"),
		Framework:            getStringPointer(d, "framework"),
		InstallCommand:       getStringPointer(d, "install_command"),
		OutputDirectory:      getStringPointer(d, "output_directory"),
		RootDirectory:        getStringPointer(d, "root_directory"),
	})
	if err != nil {
		return diag.Errorf("error creating project: %s", err)
	}

	d.SetId(out.ID)

	return nil
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Reading Project")

	project, err := c.GetProject(ctx, d.Id(), d.Get("team_id").(string))
	if err != nil {
		return diag.Errorf("error reading project: %s", err)
	}

	return updateProjectSchema(d, project)
}

func setAll(d *schema.ResourceData, m map[string]interface{}) error {
	for k, v := range m {
		if err := d.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func getEnvironmentVariablesByTarget(vars []client.EnvironmentVariable, target string) map[string]interface{} {
	m := map[string]interface{}{}
	for _, v := range vars {
		if v.Target == target {
			m[v.Key] = v.Value
		}
	}
	return m
}

func updateProjectSchema(d *schema.ResourceData, project client.ProjectResponse) diag.Diagnostics {
	if err := setAll(d, map[string]interface{}{
		"build_command":                    project.BuildCommand,
		"dev_command":                      project.DevCommand,
		"production_environment_variables": getEnvironmentVariablesByTarget(project.Env, "production"),
		"preview_environment_variables":    getEnvironmentVariablesByTarget(project.Env, "preview"),
		"framework":                        project.Framework,
		"install_command":                  project.InstallCommand,
		"name":                             project.Name,
		"output_directory":                 project.OutputDirectory,
		"public_source":                    project.PublicSource,
		"root_directory":                   project.RootDirectory,
	}); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(project.ID)

	return nil
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

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Updating Project")

	var envVars []client.EnvironmentVariable
	if d.HasChange("production_environment_variables") || d.HasChange("preview_environment_variables") {
		envVars = getEnvVariables(d)
	}

	update := client.UpdateProjectRequest{
		EnvironmentVariables: envVars,
		Name:                 getStringPointerIfChanged(d, "name"),
		BuildCommand:         getStringPointerIfChanged(d, "build_command"),
		DevCommand:           getStringPointerIfChanged(d, "dev_command"),
		Framework:            getStringPointerIfChanged(d, "framework"),
		InstallCommand:       getStringPointerIfChanged(d, "install_command"),
		OutputDirectory:      getStringPointerIfChanged(d, "output_directory"),
		RootDirectory:        getStringPointerIfChanged(d, "root_directory"),
		PublicSource:         getBoolPointerIfChanged(d, "public_source"),
	}

	project, err := c.UpdateProject(ctx, d.Id(), d.Get("team_id").(string), update)
	if err != nil {
		return diag.Errorf("error updating project: %s", err)
	}

	return updateProjectSchema(d, project)
}
