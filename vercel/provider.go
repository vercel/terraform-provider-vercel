package vercel

import (
	"context"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

type provider struct {
	configured bool
	client     *client.Client
}

func New() tfsdk.Provider {
	return &provider{}
}

func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"api_token": {
				Type:        types.StringType,
				Optional:    true,
				Description: "The API Key to use",
				Sensitive:   true,
			},
		},
	}, nil
}

func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"vercel_deployment":     resourceDeploymentType{},
		"vercel_project":        resourceProjectType{},
		"vercel_project_domain": resourceProjectDomainType{},
	}, nil
}

func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"vercel_file":              dataSourceFileType{},
		"vercel_project_directory": dataSourceProjectDirectoryType{},
	}, nil
}

type providerData struct {
	APIToken types.String `tfsdk:"api_token"`
}

var apiTokenRe = regexp.MustCompile("[0-9a-zA-Z]{24}")

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
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
