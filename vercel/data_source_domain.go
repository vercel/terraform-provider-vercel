package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &domainDataSource{}
	_ datasource.DataSourceWithConfigure = &domainDataSource{}
)

func newDomainDataSource() datasource.DataSource {
	return &domainDataSource{}
}

type domainDataSource struct {
	client *client.Client
}

func (d *domainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for a domain data source.
func (d *domainDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Domain that has been added to a Vercel account or team.

This is distinct from a ` + "`vercel_project_domain`" + `, which associates a domain name with a specific project.`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the domain.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the domain exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"zone": schema.BoolAttribute{
				Description: "Whether a DNS zone has been created for the domain on Vercel.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier of the domain.",
				Computed:    true,
			},
			"verified": schema.BoolAttribute{
				Description: "Whether the domain has its ownership verified.",
				Computed:    true,
			},
			"nameservers": schema.ListAttribute{
				Description: "A list of the current nameservers of the domain.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"intended_nameservers": schema.ListAttribute{
				Description: "A list of the intended nameservers for the domain to point to Vercel DNS.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"custom_nameservers": schema.ListAttribute{
				Description: "A list of custom nameservers for the domain to point to. Only applies to domains purchased with Vercel.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.Int64Attribute{
				Description: "Timestamp in milliseconds when the domain was created in the registry.",
				Computed:    true,
			},
			"expires_at": schema.Int64Attribute{
				Description: "Timestamp in milliseconds at which the domain is set to expire. Null if not bought with Vercel.",
				Computed:    true,
			},
			"bought_at": schema.Int64Attribute{
				Description: "Timestamp in milliseconds when the domain was purchased, if it was purchased through Vercel.",
				Computed:    true,
			},
		},
	}
}

// Read reads the domain information by requesting it from the Vercel API, and updates terraform
// with this information.
func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config Domain
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetDomain(ctx, config.Name.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Domain",
			fmt.Sprintf("Could not get Domain %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.Name.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToDomain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read Domain", map[string]any{
		"team_id": result.TeamID.ValueString(),
		"domain":  result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
