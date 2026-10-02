package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &connectApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &connectApplicationDataSource{}
)

func newConnectApplicationDataSource() datasource.DataSource { return &connectApplicationDataSource{} }

type connectApplicationDataSource struct{ client *client.Client }

type connectApplicationData struct {
	ID     types.String `tfsdk:"id"`
	UID    types.String `tfsdk:"uid"`
	TeamID types.String `tfsdk:"team_id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
}

func (d *connectApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_application"
}

func (d *connectApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *connectApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing Vercel Connect application by its stable ID or team-scoped UID. Use the returned id as a Passport connector_id. This data source exposes application identity only; it does not expose OAuth credentials or runtime tokens.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The stable Connect application ID (scl_...). Specify exactly one of id or uid.",
				Optional:    true, Computed: true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1), stringvalidator.ExactlyOneOf(path.MatchRoot("uid"))},
			},
			"uid": schema.StringAttribute{
				Description: "The team's unique application identifier, such as oauth/company-sso. Specify the literal UID; the provider URL-encodes it.",
				Optional:    true, Computed: true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"team_id": schema.StringAttribute{
				Description: "The team that owns the application. Required unless the provider has a default team.",
				Optional:    true, Computed: true,
			},
			"name": schema.StringAttribute{Computed: true, Description: "The application's display name."},
			"type": schema.StringAttribute{Computed: true, Description: "The Connect application type. Passport uses an OAuth/OIDC application."},
		},
	}
}

func (d *connectApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config connectApplicationData
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	identifier := config.ID.ValueString()
	if identifier == "" {
		identifier = config.UID.ValueString()
	}
	out, err := d.client.GetConnectApplication(ctx, config.TeamID.ValueString(), identifier)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Connect application", fmt.Sprintf("Could not get Connect application %s: %s", identifier, err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, connectApplicationData{
		ID: types.StringValue(out.ID), UID: types.StringValue(out.UID), TeamID: types.StringValue(out.TeamID), Name: types.StringValue(out.Name), Type: types.StringValue(out.Type),
	})...)
}
