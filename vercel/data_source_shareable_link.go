package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &shareableLinkDataSource{}
	_ datasource.DataSourceWithConfigure = &shareableLinkDataSource{}
)

type shareableLinkDataSource struct{ client *client.Client }

func newShareableLinkDataSource() datasource.DataSource { return &shareableLinkDataSource{} }
func (d *shareableLinkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shareable_link"
}
func (d *shareableLinkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
	}
}
func (d *shareableLinkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Reads an existing shareable link for a Vercel alias without creating or revoking it. Ordinary deployment URLs without an alias record are not supported. The secret and URL are sensitive but are stored in Terraform state.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The non-secret ID of the alias owning the link.",
		},
		"alias": schema.StringAttribute{
			Required:    true,
			Description: "The alias hostname, without a scheme or path.",
			Validators:  shareableAliasValidators,
		},
		"team_id": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The team ID. Defaults to the team configured in the provider.",
		},
		"project_id": schema.StringAttribute{
			Computed:    true,
			Description: "The project owning the alias.",
		},
		"secret": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "The bearer secret. Stored in Terraform state.",
		},
		"url": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "The HTTPS shareable URL, including the secret. Stored in Terraform state.",
		},
		"created_at": schema.Int64Attribute{
			Computed:    true,
			Description: "Creation time in Unix milliseconds.",
		},
		"created_by": schema.StringAttribute{
			Computed:    true,
			Description: "The creator's ID.",
		},
		"expires_at": schema.Int64Attribute{
			Computed:    true,
			Description: "Expiry time in Unix seconds, or null for no expiry.",
		},
		"active": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the link has not expired, as of the last read.",
		},
	}}
}
func (d *shareableLinkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ShareableLink
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alias, err := d.client.GetAlias(ctx, config.Alias.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading shareable link", err.Error())
		return
	}
	secret, link, err := findShareableLink(alias.ProtectionBypass)
	if err != nil {
		resp.Diagnostics.AddError("Error reading shareable link", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, shareableLinkState(alias, secret, link))...)
}
