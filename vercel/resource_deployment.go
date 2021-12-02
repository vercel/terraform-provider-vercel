package vercel

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vercel/terraform-provider-vercel/client"
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		DeleteContext: resourceDeploymentDelete,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"files": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeMap,
				ValidateFunc: func(i interface{}, k string) (warnings []string, errors []error) {
					v, ok := i.(map[string]interface{})
					if !ok {
						errors = append(errors, fmt.Errorf("expected files to be a map of strings"))
						return warnings, errors
					}

					if len(v) < 1 {
						errors = append(errors, fmt.Errorf("expected at least one file"))
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
			return diag.Errorf("error reading file %s: %s", filename, err)
		}

		files = append(files, client.DeploymentFile{
			File: filename,
			Sha:  sha,
			Size: len(content),
		})
	}

	out, err := c.CreateDeployment(ctx, client.CreateDeploymentRequest{
		Files:     files,
		ProjectID: d.Get("project_id").(string),
		Target:    target,
		Aliases:   []string{},
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
