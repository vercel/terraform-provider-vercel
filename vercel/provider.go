package vercel

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vercel/terraform-provider-vercel/client"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_token": {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("VERCEL_API_TOKEN", nil),
				Description:  "The API key for operations.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("[0-9a-zA-Z]{24}"), "API key must only contain characters 0-9 and a-f (all lowercased)"),
				Sensitive:    true,
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"vercel_user":              dataSourceUser(),
			"vercel_file":              dataSourceFile(),
			"vercel_project_directory": dataSourceProjectDirectory(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"vercel_deployment": resourceDeployment(),
			"vercel_files":      resourceFiles(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	token, ok := d.GetOk("api_token")
	if !ok {
		return nil, fmt.Errorf("credentials are not set correctly: missing api_token or VERCEL_API_TOKEN")
	}

	return client.New(token.(string)), nil
}
