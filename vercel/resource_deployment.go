package vercel

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vercel/terraform-provider-vercel/client"
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		DeleteContext: resourceDeploymentDelete,
		Schema: map[string]*schema.Schema{
			"project_name": {
				Optional:     true,
				ForceNew:     true,
				Type:         schema.TypeString,
				ValidateFunc: validation.StringLenBetween(1, 52),
				ExactlyOneOf: []string{"project_name", "project_id"},
			},
			"project_id": {
				Optional:     true,
				ForceNew:     true,
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"project_name", "project_id"},
			},
			"files": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeMap,
				ValidateFunc: func(i interface{}, k string) (warnings []string, errors []error) {
					v, ok := i.(map[string]interface{})
					if !ok {
						errors = append(errors, fmt.Errorf("Expected files to be a map of strings"))
						return warnings, errors
					}

					if len(v) > 1 {
						errors = append(errors, fmt.Errorf("Expected at least one file"))
						return warnings, errors
					}
					return warnings, errors
				},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"id": {
				Computed: true,
				Type:     schema.TypeString,
			},
			"url": {
				Computed: true,
				Type:     schema.TypeString,
			},
			"public": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeBool,
			},
			"is_staging": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeBool,
			},
			"is_production": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeBool,
			},
			"framework": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"dev_command": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"install_command": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"build_command": {
				ForceNew: true,
				Optional: true,
				Type:     schema.TypeString,
			},
			"output_directory": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"root_directory": {
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
		},
	}
}

func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	log.Printf("[DEBUG] Creating Deployment")

	target := ""
	if d.Get("is_staging").(bool) {
		target = "staging"
	}
	if d.Get("is_production").(bool) {
		target = "production"
	}

	var files []client.DeploymentFile
	for filename, isha := range d.Get("files").(map[string]interface{}) {
		sha := isha.(string)
		content, err := os.ReadFile(filename)
		if err != nil {
			return diag.Errorf("unable to read file %s: %s", filename, err)
		}

		err = c.CreateFile(ctx, filename, sha, string(content))
		if err != nil {
			return diag.FromErr(err)
		}

		files = append(files, client.DeploymentFile{
			File: filename,
			Sha:  sha,
			Size: len(content),
		})
	}

	out, err := c.CreateDeployment(ctx, client.CreateDeploymentRequest{
		ProjectName: d.Get("project_name").(string),
		Files:       files,
		ProjectID:   d.Get("project_id").(string),
		Public:      d.Get("public").(bool),
		Target:      target,
		Aliases:     []string{},
		ProjectSettings: client.ProjectSettings{
			Framework:       d.Get("framework").(string),
			DevCommand:      d.Get("dev_command").(string),
			BuildCommand:    d.Get("build_command").(string),
			InstallCommand:  d.Get("install_command").(string),
			OutputDirectory: d.Get("output_directory").(string),
			RootDirectory:   d.Get("root_directory").(string),
		},
	})
	if err != nil {
		return diag.Errorf("error creating deployment: %s", err)
	}

	if err := d.Set("url", out.URL); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)

	return nil
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
