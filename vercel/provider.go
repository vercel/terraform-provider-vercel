package vercel

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/providervalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ provider.ProviderWithListResources    = &vercelProvider{}
	_ provider.ProviderWithConfigValidators = &vercelProvider{}
)

const (
	oidcTokenEnv         = "VERCEL_TERRAFORM_OIDC_TOKEN"
	oidcTokenFilePathEnv = "VERCEL_TERRAFORM_OIDC_TOKEN_FILE_PATH"
	oidcPolicyIDEnv      = "VERCEL_TERRAFORM_OIDC_POLICY_ID"
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
			"oidc_token": schema.StringAttribute{
				Optional:    true,
				Description: "An OIDC token issued by an external identity provider, such as your CI/CD provider, to exchange for a short-lived Vercel access token instead of using an `api_token`. This can also be specified with the `" + oidcTokenEnv + "` shell environment variable. Requires `team`, and an OIDC federation policy for the Vercel CLI on that team that matches the token.",
				Sensitive:   true,
			},
			"oidc_token_file_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path to a file containing the OIDC token to exchange for a short-lived Vercel access token. The file is read again every time the access token is about to expire, so prefer it over `oidc_token` when the OIDC token is rotated during a run. This can also be specified with the `" + oidcTokenFilePathEnv + "` shell environment variable.",
			},
			"oidc_policy_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the OIDC federation policy to exchange the OIDC token with. Only needed when more than one policy of the team matches the token. This can also be specified with the `" + oidcPolicyIDEnv + "` shell environment variable.",
			},
			"team": schema.StringAttribute{
				Optional:    true,
				Description: "The default Vercel Team to use when creating resources or reading data sources. This can be provided as either a team slug, or team ID. The slug and ID are both available from the Team Settings page in the Vercel dashboard.",
			},
		},
	}
}

func (p *vercelProvider) ConfigValidators(_ context.Context) []provider.ConfigValidator {
	return []provider.ConfigValidator{
		providervalidator.Conflicting(
			path.MatchRoot("api_token"),
			path.MatchRoot("oidc_token"),
			path.MatchRoot("oidc_token_file_path"),
		),
	}
}

func (p *vercelProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAccessGroupMemberResource,
		newAccessGroupProjectResource,
		newAccessGroupResource,
		newAIGatewayAPIKeyResource,
		newAlertRuleResource,
		newAlertRuleSlackNotificationResource,
		newAlertRuleWebhookNotificationResource,
		newAliasResource,
		newAttackChallengeModeResource,
		newAuditLogDrainResource,
		newBulkRedirectsResource,
		newBlobObjectResource,
		newBlobProjectConnectionResource,
		newBlobStoreResource,
		newCustomAlertRuleResource,
		newCustomCertificateResource,
		newCustomEnvironmentResource,
		newCustomMonitorResource,
		newDeploymentProtectionExceptionResource,
		newDeploymentResource,
		newDNSRecordResource,
		newEdgeConfigItemResource,
		newEdgeConfigResource,
		newEdgeConfigSchemaResource,
		newEdgeConfigTokenResource,
		newFirewallBypassResource,
		newFirewallConfigResource,
		newFeatureFlagDefinitionResource,
		newFeatureFlagConnectionResource,
		newFeatureFlagSDKKeyResource,
		newFeatureFlagSegmentResource,
		newFeatureFlagConfigResource,
		newIntegrationProjectAccessResource,
		newKMSIssuerResource,
		newKMSProjectGrantResource,
		newKMSSigningKeyResource,
		newLogDrainResource,
		newMicrofrontendGroupMembershipResource,
		newMicrofrontendGroupResource,
		newMonitorResource,
		newMonitorSlackNotificationResource,
		newMonitorWebhookNotificationResource,
		newNetworkResource,
		newOAuthAppClientSecretResource,
		newOAuthAppResource,
		newOIDCFederationPolicyResource,
		newProjectDeploymentRetentionResource,
		newProjectDeploymentCheckResource,
		newProjectCronsResource,
		newProjectDomainResource,
		newProjectEnvironmentVariableResource,
		newProjectEnvironmentVariablesResource,
		newProjectMembersResource,
		newProjectProtectionBypassResource,
		newProjectPassportResource,
		newProjectDeploymentProtectionResource,
		newProjectRouteResource,
		newProjectResource,
		newProjectTracingResource,
		newSharedEnvironmentVariableProjectLinkResource,
		newShareableLinkResource,
		newSharedEnvironmentVariableResource,
		newOrganizationTeamResource,
		newTeamConfigResource,
		newTeamMemberResource,
		newTraceDrainResource,
		newUserTokenResource,
		newVCRRepositoryResource,
		newVCRRepositoryPermissionResource,
		newWebhookResource,
		newProjectRollingReleaseResource,
	}
}

func (p *vercelProvider) ListResources(context.Context) []func() list.ListResource {
	return []func() list.ListResource{newProjectListResource}
}

func (p *vercelProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newAccessGroupDataSource,
		newAccessGroupProjectDataSource,
		newAliasDataSource,
		newAttackChallengeModeDataSource,
		newBulkRedirectsDataSource,
		newBlobObjectDataSource,
		newBlobProjectConnectionsDataSource,
		newBlobStoreDataSource,
		newBlobStoreSecretsDataSource,
		newBlobStoresDataSource,
		newCustomEnvironmentDataSource,
		newConnectApplicationDataSource,
		newDeploymentDataSource,
		newDomainConfigDataSource,
		newEdgeConfigDataSource,
		newEdgeConfigItemDataSource,
		newEdgeConfigSchemaDataSource,
		newEdgeConfigTokenDataSource,
		newEndpointVerificationDataSource,
		newFileDataSource,
		newFeatureFlagDataSource,
		newFeatureFlagSDKKeyDataSource,
		newFeatureFlagSegmentDataSource,
		newKMSIssuerDataSource,
		newLogDrainDataSource,
		newNetworkDataSource,
		newPrebuiltProjectDataSource,
		newProjectDataSource,
		newProjectsDataSource,
		newProjectDeploymentRetentionDataSource,
		newProjectDirectoryDataSource,
		newProjectMembersDataSource,
		newProjectProtectionInventoryDataSource,
		newProjectRoutesDataSource,
		newShareableLinkDataSource,
		newSharedEnvironmentVariableDataSource,
		newTeamConfigDataSource,
		newTeamMemberDataSource,
		newTraceDrainDataSource,
		newVCRRepositoryDataSource,
		newMicrofrontendGroupDataSource,
		newMicrofrontendGroupMembershipDataSource,
		newProjectRollingReleaseDataSource,
		newDsyncGroupsDataSource,
	}
}

type providerData struct {
	APIToken          types.String `tfsdk:"api_token"`
	OIDCToken         types.String `tfsdk:"oidc_token"`
	OIDCTokenFilePath types.String `tfsdk:"oidc_token_file_path"`
	OIDCPolicyID      types.String `tfsdk:"oidc_policy_id"`
	Team              types.String `tfsdk:"team"`
}

// providerCredentials is what the provider authenticates with: an API token,
// or an OIDC token given either inline or as the path to a file containing it.
type providerCredentials struct {
	apiToken          string
	oidcToken         string
	oidcTokenFilePath string
}

// credentials returns the credentials set in the provider configuration. The
// environment is only used when the configuration sets none, so that a leftover
// environment variable never overrides the configured way to authenticate.
func (config providerData) credentials() providerCredentials {
	if !config.APIToken.IsNull() || !config.OIDCToken.IsNull() || !config.OIDCTokenFilePath.IsNull() {
		return providerCredentials{
			apiToken:          config.APIToken.ValueString(),
			oidcToken:         config.OIDCToken.ValueString(),
			oidcTokenFilePath: config.OIDCTokenFilePath.ValueString(),
		}
	}
	if apiToken := os.Getenv("VERCEL_API_TOKEN"); apiToken != "" {
		return providerCredentials{apiToken: apiToken}
	}
	if oidcToken := os.Getenv(oidcTokenEnv); oidcToken != "" {
		return providerCredentials{oidcToken: oidcToken}
	}
	return providerCredentials{oidcTokenFilePath: os.Getenv(oidcTokenFilePathEnv)}
}

func (c providerCredentials) useOIDC() bool {
	return c.oidcToken != "" || c.oidcTokenFilePath != ""
}

// oidcSubjectToken returns the OIDC token to exchange. A token file is read on
// every call, because its content changes when the token is rotated.
func (c providerCredentials) oidcSubjectToken() (string, error) {
	if c.oidcTokenFilePath == "" {
		return c.oidcToken, nil
	}
	content, err := os.ReadFile(c.oidcTokenFilePath)
	if err != nil {
		return "", fmt.Errorf("reading the OIDC token file: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

// apiTokenRe is a regex for an API access token. We use this to validate that the
// token provided matches the expected format.
var apiTokenRe = regexp.MustCompile("[0-9a-zA-Z]{24}")

// Configure takes a provider and applies any configuration. In the context of Vercel
// this allows us to set up an API token, or to exchange an OIDC token for one.
func (p *vercelProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, attribute := range []struct {
		name  string
		value types.String
	}{
		{"api_token", config.APIToken},
		{"oidc_token", config.OIDCToken},
		{"oidc_token_file_path", config.OIDCTokenFilePath},
		{"oidc_policy_id", config.OIDCPolicyID},
	} {
		if attribute.value.IsUnknown() {
			resp.Diagnostics.AddWarning(
				"Unable to create client",
				"Cannot use unknown value as "+attribute.name,
			)
			return
		}
	}

	// User must provide an api_token, or an OIDC token to exchange for one, to the provider
	credentials := config.credentials()
	var vercelClient *client.Client
	switch {
	case credentials.useOIDC():
		if config.Team.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Unable to find team",
				"team is required to authenticate with an OIDC token, because the Vercel access token it is exchanged for only gives access to one team",
			)
			return
		}

		policyID := config.OIDCPolicyID.ValueString()
		if config.OIDCPolicyID.IsNull() {
			policyID = os.Getenv(oidcPolicyIDEnv)
		}
		vercelClient = client.NewWithOIDC(client.OIDCConfig{
			ClientID:     vercelCLIClientID,
			Team:         config.Team.ValueString(),
			PolicyID:     policyID,
			SubjectToken: credentials.oidcSubjectToken,
		})
		if err := vercelClient.Authenticate(ctx); err != nil {
			resp.Diagnostics.AddError(
				"Unable to authenticate with OIDC",
				fmt.Sprintf("Could not authenticate with OIDC, unexpected error: %s", err),
			)
			return
		}
	case credentials.apiToken == "":
		resp.Diagnostics.AddError(
			"Unable to find api_token",
			"api_token cannot be an empty string. Set api_token, or set oidc_token or oidc_token_file_path to authenticate with OIDC instead",
		)
		return
	case !apiTokenRe.MatchString(credentials.apiToken):
		resp.Diagnostics.AddError(
			"Invalid api_token",
			"api_token (VERCEL_API_TOKEN) must be 24 characters and only contain characters 0-9 and a-f (all lowercased)",
		)
		return
	default:
		vercelClient = client.New(credentials.apiToken)
	}

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
	resp.ListResourceData = vercelClient
}
