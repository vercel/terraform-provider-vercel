package vercel

import (
	"context"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

type vercelProvider struct {
	configured bool
	client     *client.Client
}

// New instantiates a new instance of a vercel terraform provider.
func New() provider.Provider {
	return &vercelProvider{}
}

// GetSchema returns the schema information for the provider configuration itself.
func (p *vercelProvider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
The Vercel provider is used to interact with resources supported by Vercel.
The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.
        `,
		Attributes: map[string]tfsdk.Attribute{
			"api_token": {
				Type:        types.StringType,
				Optional:    true,
				Description: "The Vercel API Token to use. This can also be specified with the `VERCEL_API_TOKEN` shell environment variable. Tokens can be created from your [Vercel settings](https://vercel.com/account/tokens).",
				Sensitive:   true,
			},
		},
	}, nil
}

// GetResources shows the available resources for the vercel provider
func (p *vercelProvider) GetResources(_ context.Context) (map[string]provider.ResourceType, diag.Diagnostics) {
	return map[string]provider.ResourceType{
		"vercel_alias":                        resourceAliasType{},
		"vercel_deployment":                   resourceDeploymentType{},
		"vercel_project":                      resourceProjectType{},
		"vercel_project_domain":               resourceProjectDomainType{},
		"vercel_project_environment_variable": resourceProjectEnvironmentVariableType{},
		"vercel_dns_record":                   resourceDNSRecordType{},
	}, nil
}

// GetDataSources shows the available data sources for the vercel provider
func (p *vercelProvider) GetDataSources(_ context.Context) (map[string]provider.DataSourceType, diag.Diagnostics) {
	return map[string]provider.DataSourceType{
		"vercel_file":              dataSourceFileType{},
		"vercel_project":           dataSourceProjectType{},
		"vercel_project_directory": dataSourceProjectDirectoryType{},
		"vercel_alias":             dataSourceAliasType{},
	}, nil
}

type providerData struct {
	APIToken types.String `tfsdk:"api_token"`
}

// apiTokenRe is a regex for an API access token. We use this to validate that the
// token provided matches the expected format.
var apiTokenRe = regexp.MustCompile("[0-9a-zA-Z]{24}")

// Configure takes a provider and applies any configuration. In the context of Vercel
// this allows us to set up an API token.
func (p *vercelProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// User must provide an api_token to the provider
	var apiToken string
	if config.APIToken.Unknown {
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as api_token",
		)
		return
	}

	if config.APIToken.Null {
		apiToken = os.Getenv("VERCEL_API_TOKEN")
	} else {
		apiToken = config.APIToken.Value
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Unable to find api_token",
			"api_token cannot be an empty string",
		)
		return
	}

	if !apiTokenRe.MatchString(apiToken) {
		resp.Diagnostics.AddError(
			"Invalid api_token",
			"api_token (VERCEL_API_TOKEN) must be 24 characters and only contain characters 0-9 and a-f (all lowercased)",
		)
		return
	}

	p.client = client.New(apiToken)
	p.configured = true
}
