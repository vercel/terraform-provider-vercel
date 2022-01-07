package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

var ignores = map[string]struct{}{
	".hg":                  {},
	".git":                 {},
	".gitmodules":          {},
	".svn":                 {},
	".cache":               {},
	".next":                {},
	".now":                 {},
	".vercel":              {},
	".npmignore":           {},
	".dockerignore":        {},
	".gitignore":           {},
	".*.swp":               {},
	".DS_Store":            {},
	".wafpicke-*":          {},
	".lock-wscript":        {},
	".env.local":           {},
	".env.*.local":         {},
	".venv":                {},
	"npm-debug.log":        {},
	"config.gypi":          {},
	"node_modules":         {},
	"__pycache__":          {},
	"venv":                 {},
	"CVS":                  {},
	".vercel_build_output": {},
}

func dataSourceProjectDirectoryRead(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	files := map[string]interface{}{}
	err := filepath.WalkDir(
		d.Get("path").(string),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// TODO - ignores should use a `glob` pattern instead.
			// look into how Jared does this on turbo
			_, ignored := ignores[d.Name()]

			if d.IsDir() && ignored {
				return filepath.SkipDir
			}
			if ignored {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rawSha := sha1.Sum(content)
			sha := hex.EncodeToString(rawSha[:])

			files[path] = fmt.Sprintf("%d~%s", len(content), sha)
			return nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("files", files); err != nil {
		return diag.FromErr(err)
	}
	// Always read
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
