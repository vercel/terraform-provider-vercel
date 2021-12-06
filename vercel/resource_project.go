package vercel

import (
	"context"
	"errors"
	"log"
	"net/http"

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
				Computed:    true,
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
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The build command for this project. If omitted, this value will be automatically detected",
			},
			"dev_command": {
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The dev command for this project. If omitted, this value will be automatically detected",
			},
			"environment": {
				Description: "An environment variable for the project.",
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": {
							Description: "The environments that the environment variable should be present on. Valid targets are be either `production`, `preview`, or `development`. If omitted, the variable will exist across all targets.",
							Type:        schema.TypeList,
							MinItems:    1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"production",
									"preview",
									"development",
								}, false),
							},
							Required: true,
						},
						"key": {
							Description: "The name of the environment variable",
							Type:        schema.TypeString,
							Required:    true,
						},
						"value": {
							Description: "The value of the environment variable",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"framework": {
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The framework that is being used for this project. If omitted, no framework is selected",
			},
			"git_repository": {
				Description: "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed",
				Optional:    true,
				Computed:    true,
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
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The install command for this project. If omitted, this value will be automatically detected",
			},
			"output_directory": {
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The output directory of the project. When null is used this value will be automatically detected",
			},
			"public_source": {
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeBool,
				Description: "Specifies whether the source code and logs of the deployments for this project should be public or not",
			},
			"root_directory": {
				Optional:    true,
				Computed:    true,
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

func getBoolPointer(d *schema.ResourceData, key string) *bool {
	if v, ok := d.GetOk(key); ok {
		value := v.(bool)
		return &value
	}
	return nil
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Creating Project")
	environmentVariables := []client.EnvironmentVariable{}
	for _, e := range d.Get("environment").([]interface{}) {
		env := e.(map[string]interface{})

		target := []string{}
		for _, t := range env["target"].([]interface{}) {
			target = append(target, t.(string))
		}
		if len(target) == 0 {
			target = []string{"production", "preview", "development"}
		}
		environmentVariables = append(environmentVariables, client.EnvironmentVariable{
			Key:    env["key"].(string),
			Value:  env["value"].(string),
			Target: target,
			Type:   "encrypted",
		})
	}

	out, err := c.CreateProject(ctx, d.Get("team_id").(string), client.CreateProjectRequest{
		Name:                 d.Get("name").(string),
		PublicSource:         getBoolPointer(d, "public_source"),
		EnvironmentVariables: environmentVariables,
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

func setStringPointers(d *schema.ResourceData, m map[string]*string) error {
	for k, v := range m {
		if _, ok := d.GetOk(k); !ok && v == nil {
			continue
		}
		if err := d.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func updateProjectSchema(d *schema.ResourceData, project client.ProjectResponse) diag.Diagnostics {
	if err := setStringPointers(d, map[string]*string{
		"build_command":    project.BuildCommand,
		"dev_command":      project.DevCommand,
		"framework":        project.Framework,
		"install_command":  project.InstallCommand,
		"name":             &project.Name,
		"output_directory": project.OutputDirectory,
		"root_directory":   project.RootDirectory,
	}); err != nil {
		return diag.FromErr(err)
	}
	// "environment":      project.Env,
	if err := d.Set("public_source", project.PublicSource); err != nil {
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

func diffEnvVars(oldVars, newVars map[string]interface{}) (toUpsert, toRemove map[string]interface{}) {
	toUpsert = map[string]interface{}{}
	toRemove = map[string]interface{}{}

	for key, value := range newVars {
		if oldValue, ok := oldVars[key]; !ok || oldValue != value {
			toUpsert[key] = value
		}
	}

	for key, value := range oldVars {
		if _, ok := newVars[key]; !ok {
			toRemove[key] = value
		}
	}

	return
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Updating Project")

	update := client.UpdateProjectRequest{
		Name:            getStringPointerIfChanged(d, "name"),
		BuildCommand:    getStringPointerIfChanged(d, "build_command"),
		DevCommand:      getStringPointerIfChanged(d, "dev_command"),
		Framework:       getStringPointerIfChanged(d, "framework"),
		InstallCommand:  getStringPointerIfChanged(d, "install_command"),
		OutputDirectory: getStringPointerIfChanged(d, "output_directory"),
		RootDirectory:   getStringPointerIfChanged(d, "root_directory"),
		PublicSource:    getBoolPointerIfChanged(d, "public_source"),
	}

	project, err := c.UpdateProject(ctx, d.Id(), d.Get("team_id").(string), update)
	if err != nil {
		return diag.Errorf("error updating project: %s", err)
	}

	return updateProjectSchema(d, project)
}
