package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &domainConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &domainConfigDataSource{}
)

func newDomainConfigDataSource() datasource.DataSource {
	return &domainConfigDataSource{}
}

type domainConfigDataSource struct {
	client *client.Client
}

func (d *domainConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_config"
}

func (d *domainConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Schema returns the schema information for a domain config data source
func (d *domainConfigDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides domain configuration information for a Vercel project.

This data source returns configuration details for a domain associated with a specific project,
including recommended CNAME and IPv4 values.
		`,
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain name to get configuration for.",
			},
			"project_id_or_name": schema.StringAttribute{
				Required:    true,
				Description: "The project ID or name associated with the domain.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the domain config exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"recommended_cname": schema.StringAttribute{
				Computed:    true,
				Description: "The recommended CNAME value for the domain.",
			},
			"recommended_ipv4s": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "The recommended IPv4 values for the domain.",
			},
		},
	}
}

// DomainConfigDataSource reflects the state terraform stores internally for a domain config.
type DomainConfigDataSource struct {
	Domain            types.String `tfsdk:"domain"`
	ProjectIdOrName   types.String `tfsdk:"project_id_or_name"`
	TeamID            types.String `tfsdk:"team_id"`
	RecommendedCNAME  types.String `tfsdk:"recommended_cname"`
	RecommendedIPv4s  types.List   `tfsdk:"recommended_ipv4s"`
}

// Read will read domain config information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *domainConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainConfigDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetDomainConfig(ctx, config.Domain.ValueString(), config.ProjectIdOrName.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading domain config",
			fmt.Sprintf("Could not read domain config for domain %s and project %s, unexpected error: %s",
				config.Domain.ValueString(),
				config.ProjectIdOrName.ValueString(),
				err,
			),
		)
		return
	}

	var ipv4Values []attr.Value
	for _, ip := range out.RecommendedIPv4s {
		ipv4Values = append(ipv4Values, types.StringValue(ip))
	}

	result := DomainConfigDataSource{
		Domain:            config.Domain,
		ProjectIdOrName:   config.ProjectIdOrName,
		TeamID:            config.TeamID,
		RecommendedCNAME:  types.StringValue(out.RecommendedCNAME),
		RecommendedIPv4s:  types.ListValueMust(types.StringType, ipv4Values),
	}

	tflog.Info(ctx, "read domain config", map[string]any{
		"domain":            result.Domain.ValueString(),
		"projectIdOrName":   result.ProjectIdOrName.ValueString(),
		"teamId":            result.TeamID.ValueString(),
		"recommendedCNAME":  result.RecommendedCNAME.ValueString(),
		"recommendedIPv4s":  result.RecommendedIPv4s.Elements(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
