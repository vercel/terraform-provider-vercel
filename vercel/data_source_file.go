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
			"file": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"sha": {
				Computed: true,
				Type:     schema.TypeString,
			},
		},
	}
}

func dataSourceFileRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	filename := d.Get("file").(string)
	content, err := os.ReadFile(d.Get("file").(string))
	if err != nil {
		return diag.Errorf("error reading file %s: %w", filename, err)
	}
	rawSha := sha1.Sum(content)
	sha := hex.EncodeToString(rawSha[:])

	d.SetId(filename)
	d.Set("sha", sha)
	return nil
}
