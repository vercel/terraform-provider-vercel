package vercel

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

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
				ValidateFunc: func(i interface{}, _ string) (warnings []string, errors []error) {
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

	// Build up files for the API, and files by SHA so we can easily identify which files to upload.
	var files []client.DeploymentFile
	filesBySha := map[string]client.DeploymentFile{}
	for filename, rawSizeSha := range d.Get("files").(map[string]interface{}) {
		sizeSha := strings.Split(rawSizeSha.(string), "~")
		size, err := strconv.Atoi(sizeSha[0])
		if err != nil {
			return diag.Errorf("error parsing file input for file '%s': %s", filename, rawSizeSha.(string))
		}
		sha := sizeSha[1]

		file := client.DeploymentFile{
			File: filename,
			Sha:  sha,
			Size: size,
		}
		files = append(files, file)
		filesBySha[sha] = file
	}

	cdr := client.CreateDeploymentRequest{
		Files:     files,
		ProjectID: d.Get("project_id").(string),
		Target:    target,
		Aliases:   []string{},
	}

	// First we attempt to create a deployment without bothering to upload any files.
	out, err := c.CreateDeployment(ctx, cdr)
	var mfErr client.MissingFilesError
	if errors.As(err, &mfErr) {
		// Then we need to upload the files, and create the deployment again.
		for _, sha := range mfErr.Missing {
			f := filesBySha[sha]
			content, err := os.ReadFile(f.File)
			if err != nil {
				return diag.Errorf("unable to read file '%s': %s", f.File, err)
			}

			err = c.CreateFile(ctx, f.File, f.Sha, string(content))
			if err != nil {
				return diag.Errorf("error uploading deployment file '%s': %s", f.File, err)
			}
		}

		out, err = c.CreateDeployment(ctx, cdr)
		if err != nil {
			return diag.Errorf("error creating deployment: %s", err)
		}
	} else if err != nil {
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

func resourceDeploymentDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
