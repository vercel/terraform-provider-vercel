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
			// "password_protection": {
			// 	// TODO - password protection??
			// },
			"public": {
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
			// "sso_protection": {
			// 	// TODO - sso protection??
			// },
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	log.Printf("[DEBUG] Creating Project")

	out, err := c.CreateProject(ctx, client.CreateProjectRequest{})
	if err != nil {
		return diag.Errorf("error creating project: %s", err)
	}

	d.SetId(out.ID)

	return nil
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Reading Project")
	return nil
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Deleting Project")
	d.SetId("")
	return nil
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Updating Project")
	return nil
}
