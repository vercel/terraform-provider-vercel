package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFile() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFileRead,
		Schema: map[string]*schema.Schema{
			"path": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"file": {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceFileRead(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	path := d.Get("path").(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return diag.Errorf("error reading file %s: %s", path, err)
	}
	rawSha := sha1.Sum(content)
	sha := hex.EncodeToString(rawSha[:])

	d.SetId(path)
	d.Set("file", map[string]interface{}{
		path: sha,
	})
	return nil
}
