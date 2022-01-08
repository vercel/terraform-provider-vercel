package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vercel/terraform-provider-vercel/glob"
)

func dataSourceProjectDirectory() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectDirectoryRead,
		Schema: map[string]*schema.Schema{
			"path": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"files": {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceProjectDirectoryRead(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Reading Project Directory")
	files := map[string]interface{}{}
	dir := d.Get("path").(string)
	ignoreRules, err := glob.GetIgnores(dir)
	if err != nil {
		return diag.Errorf("unable to get vercelignore rules: %s", err)
	}

	paths, err := glob.GetPaths(dir, ignoreRules)
	if err != nil {
		return diag.Errorf("unable to get files for directory %s: %s", dir, err)
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return diag.Errorf("unable to read file %s: %s", path, err)
		}
		rawSha := sha1.Sum(content)
		sha := hex.EncodeToString(rawSha[:])

		files[path] = fmt.Sprintf("%d~%s", len(content), sha)
	}

	if err := d.Set("files", files); err != nil {
		return diag.FromErr(err)
	}

	// Always read
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
