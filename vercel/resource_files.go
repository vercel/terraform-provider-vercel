package vercel

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vercel/terraform-provider-vercel/client"
)

func resourceFiles() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFilesCreate,
		ReadContext:   resourceFilesRead,
		DeleteContext: resourceFilesDelete,
		Schema: map[string]*schema.Schema{
			"files": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
			},
		},
	}
}

func resourceFilesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	log.Printf("[DEBUG] Creating Files")

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
	}

	// Always do something
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func resourceFilesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceFilesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
