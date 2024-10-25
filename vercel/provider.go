package vercel

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

type vercelProvider struct{}

// New instantiates a new instance of a vercel terraform provider.
func New() provider.Provider {
	return &vercelProvider{}
}

func (p *vercelProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "vercel"
}

// Schema returns the schema information for the provider configuration itself.
func (p *vercelProvider) Schema(_ context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
The Vercel provider is used to interact with resources supported by Vercel.
The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.
        `,
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Optional:    true,
				Description: "The Vercel API Token to use. This can also be specified with the `VERCEL_API_TOKEN` shell environment variable. Tokens can be created from your [Vercel settings](https://vercel.com/account/tokens).",
				Sensitive:   true,
			},
			"team": schema.StringAttribute{
				Optional:    true,
				Description: "The default Vercel Team to use when creating resources or reading data sources. This can be provided as either a team slug, or team ID. The slug and ID are both available from the Team Settings page in the Vercel dashboard.",
			},
		},
	}
}

func (p *vercelProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAliasResource,
		newAttackChallengeModeResource,
		newDNSRecordResource,
		newDeploymentResource,
		newEdgeConfigResource,
		newEdgeConfigItemResource,
		newEdgeConfigSchemaResource,
		newEdgeConfigTokenResource,
		newFirewallConfigResource,
		newLogDrainResource,
		newProjectDeploymentRetentionResource,
		newProjectDomainResource,
		newProjectEnvironmentVariableResource,
		newProjectResource,
		newSharedEnvironmentVariableResource,
		newTeamConfigResource,
		newWebhookResource,
	}
}

func (p *vercelProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newAliasDataSource,
		newAttackChallengeModeDataSource,
		newDeploymentDataSource,
		newEdgeConfigDataSource,
		newEdgeConfigItemDataSource,
		newEdgeConfigSchemaDataSource,
		newEdgeConfigTokenDataSource,
		newEndpointVerificationDataSource,
		newFileDataSource,
		newLogDrainDataSource,
		newPrebuiltProjectDataSource,
		newProjectDataSource,
		newProjectDeploymentRetentionDataSource,
		newProjectDirectoryDataSource,
		newSharedEnvironmentVariableDataSource,
		newTeamConfigDataSource,
	}
}

type providerData struct {
	APIToken types.String `tfsdk:"api_token"`
	Team     types.String `tfsdk:"team"`
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
	if config.APIToken.IsUnknown() {
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as api_token",
		)
		return
	}

	if config.APIToken.IsNull() {
		apiToken = os.Getenv("VERCEL_API_TOKEN")
	} else {
		apiToken = config.APIToken.ValueString()
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

	vercelClient := client.New(apiToken)
	if config.Team.ValueString() != "" {
		res, err := vercelClient.GetTeam(ctx, config.Team.ValueString())
		if client.NotFound(err) {
			resp.Diagnostics.AddError(
				"Vercel Team not found",
				"You provided a `team` field on the Vercel provider, but the team could not be found. Please check the team slug or ID is correct and that your api_token has access to the team.",
			)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Unexpected error reading Vercel Team",
				fmt.Sprintf("Could not read Vercel Team %s, unexpected error: %s", config.Team.ValueString(), err),
			)
			return
		}
		vercelClient = vercelClient.WithTeam(res)
	}

	resp.DataSourceData = vercelClient
	resp.ResourceData = vercelClient
}
