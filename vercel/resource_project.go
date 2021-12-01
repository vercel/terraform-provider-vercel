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
				Required:    false,
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
			"environment_variables": {
				Optional: true,
				Type:     schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Collection of environment variables the Project will use",
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
				ForceNew:    true,
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

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Creating Project")
	request := client.CreateProjectRequest{
		Name:         d.Get("name").(string),
		PublicSource: d.Get("public_source").(bool),
	}

	if v, ok := d.GetOk("build_command"); ok {
		buildCommand := v.(string)
		request.BuildCommand = &buildCommand
	}

	if v, ok := d.GetOk("dev_command"); ok {
		devCommand := v.(string)
		request.DevCommand = &devCommand
	}

	if v, ok := d.GetOk("environment_variables"); ok {
		env := map[string]string{}
		for k, v := range v.(map[string]interface{}) {
			env[k] = v.(string)
		}
		request.EnvironmentVariables = env
	}

	if v, ok := d.GetOk("framework"); ok {
		framework := v.(string)
		request.Framework = &framework
	}

	if v, ok := d.GetOk("install_command"); ok {
		installCommand := v.(string)
		request.InstallCommand = &installCommand
	}

	if v, ok := d.GetOk("output_directory"); ok {
		outputDir := v.(string)
		request.OutputDirectory = &outputDir
	}

	if v, ok := d.GetOk("root_directory"); ok {
		rootDir := v.(string)
		request.RootDirectory = &rootDir
	}

	out, err := c.CreateProject(ctx, request, d.Get("team_id").(string))
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

func updateProjectSchema(d *schema.ResourceData, project client.ProjectResponse) diag.Diagnostics {
	if err := d.Set("build_command", project.BuildCommand); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("dev_command", project.DevCommand); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("environment_variables", project.Env); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("framework", project.Framework); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(project.ID)
	if err := d.Set("install_command", project.InstallCommand); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", project.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("output_directory", project.OutputDirectory); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("public_source", project.PublicSource); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("root_directory", project.RootDirectory); err != nil {
		return diag.FromErr(err)
	}

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

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Updating Project")
	update := client.UpdateProjectRequest{}

	if d.HasChange("build_command") {
		buildCommand := d.Get("build_command").(string)
		update.BuildCommand = &buildCommand
	}
	if d.HasChange("dev_command") {
		devCommand := d.Get("dev_command").(string)
		update.DevCommand = &devCommand
	}
	if d.HasChange("environment_variables") {
		env := map[string]string{}
		for k, v := range d.Get("environment_variables").(map[string]interface{}) {
			env[k] = v.(string)
		}
		update.EnvironmentVariables = env
	}
	if d.HasChange("framework") {
		framework := d.Get("framework").(string)
		update.Framework = &framework
	}
	if d.HasChange("install_command") {
		installCommand := d.Get("install_command").(string)
		update.InstallCommand = &installCommand
	}
	if d.HasChange("name") {
		update.Name = d.Get("name").(string)
	}
	if d.HasChange("output_directory") {
		outputDir := d.Get("output_directory").(string)
		update.OutputDirectory = &outputDir
	}
	if d.HasChange("public_source") {
		update.PublicSource = d.Get("public_source").(bool)
	}
	if d.HasChange("root_directory") {
		rootDir := d.Get("root_directory").(string)
		update.RootDirectory = &rootDir
	}

	project, err := c.UpdateProject(ctx, update, d.Id(), d.Get("team_id").(string))
	if err != nil {
		return diag.Errorf("error updating project: %s", err)
	}

	return updateProjectSchema(d, project)
}
