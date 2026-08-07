package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &kmsIssuerDataSource{}
	_ datasource.DataSourceWithConfigure = &kmsIssuerDataSource{}
)

func newKMSIssuerDataSource() datasource.DataSource {
	return &kmsIssuerDataSource{}
}

type kmsIssuerDataSource struct {
	client *client.Client
}

type kmsIssuerDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	TeamID      types.String `tfsdk:"team_id"`
	Name        types.String `tfsdk:"name"`
	Algorithm   types.String `tfsdk:"algorithm"`
	OwnerID     types.String `tfsdk:"owner_id"`
	Origin      types.String `tfsdk:"origin"`
	ManagedBy   types.String `tfsdk:"managed_by"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	SigningKeys types.List   `tfsdk:"signing_keys"`
	Policies    types.List   `tfsdk:"policies"`
}

func (d *kmsIssuerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_issuer"
}

func (d *kmsIssuerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func kmsSigningKeysDataSourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "The signing keys belonging to the issuer.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key_id":                 schema.StringAttribute{Computed: true, Description: "The ID of the signing key. Used as the JWT/JWKS `kid`."},
				"issuer_id":              schema.StringAttribute{Computed: true, Description: "The ID of the issuer the key belongs to."},
				"algorithm":              schema.StringAttribute{Computed: true, Description: "The signing algorithm of the key."},
				"status":                 schema.StringAttribute{Computed: true, Description: "The status of the key, either `active` or `revoking`."},
				"public_key":             schema.StringAttribute{Computed: true, Description: "The public key as a JSON-encoded JWK."},
				"public_key_fingerprint": schema.StringAttribute{Computed: true, Description: "The fingerprint of the public key."},
				"created_at":             schema.StringAttribute{Computed: true, Description: "The time the key was created."},
				"updated_at":             schema.StringAttribute{Computed: true, Description: "The time the key was last updated."},
				"revoke_at":              schema.StringAttribute{Computed: true, Description: "The time at which the key is scheduled to be revoked, if any."},
			},
		},
	}
}

func kmsPoliciesDataSourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "The authorization policies attached to the issuer.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"kind":       schema.StringAttribute{Computed: true, Description: "The policy kind, either `project-grant` or `connex-grant`."},
				"team_id":    schema.StringAttribute{Computed: true, Description: "The team ID associated with the policy, for project grants."},
				"project_id": schema.StringAttribute{Computed: true, Description: "The project ID associated with the policy, for project grants."},
				"client_id":  schema.StringAttribute{Computed: true, Description: "The client ID associated with the policy, for connex grants."},
				"environments": schema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "The environments the policy applies to, for project grants.",
				},
				"token_claims": schema.StringAttribute{Computed: true, Description: "The claims KMS includes in signed JWTs for this policy, as a JSON-encoded object."},
				"created_at":   schema.StringAttribute{Computed: true, Description: "The time the policy was created."},
				"updated_at":   schema.StringAttribute{Computed: true, Description: "The time the policy was last updated."},
			},
		},
	}
}

func (d *kmsIssuerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Vercel KMS Issuer.

A KMS Issuer signs JSON Web Tokens (JWTs) and exposes a JWKS endpoint at
` + "`https://kms.vercel.com/{id}/jwks.json`" + `.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the issuer.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the issuer exists under. Required when reading a team resource if a default team has not been set in the provider.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the issuer.",
			},
			"algorithm": schema.StringAttribute{
				Computed:    true,
				Description: "The signing algorithm used by the issuer.",
			},
			"owner_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the team that owns the issuer.",
			},
			"origin": schema.StringAttribute{
				Computed:    true,
				Description: "The origin of the issuer's key material, either `vercel` or `external`.",
			},
			"managed_by": schema.StringAttribute{
				Computed:    true,
				Description: "Identifies the policy that manages this issuer, if any.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time the issuer was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time the issuer was last updated.",
			},
			"signing_keys": kmsSigningKeysDataSourceAttribute(),
			"policies":     kmsPoliciesDataSourceAttribute(),
		},
	}
}

func (d *kmsIssuerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config kmsIssuerDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetKMSIssuer(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading KMS Issuer",
			fmt.Sprintf("Could not read KMS Issuer %s, unexpected error: %s", config.ID.ValueString(), err),
		)
		return
	}

	signingKeys, diags := kmsSigningKeysToList(ctx, out.SigningKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, diags := kmsIssuerPoliciesToList(ctx, out.Policies)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "read kms issuer", map[string]any{
		"team_id":   out.TeamID,
		"issuer_id": out.ID,
	})

	diags = resp.State.Set(ctx, kmsIssuerDataSourceModel{
		ID:          types.StringValue(out.ID),
		TeamID:      toTeamID(out.TeamID),
		Name:        types.StringValue(out.Name),
		Algorithm:   types.StringValue(out.Algorithm),
		OwnerID:     types.StringValue(out.OwnerID),
		Origin:      types.StringValue(out.Origin),
		ManagedBy:   emptyToNull(out.ManagedBy),
		CreatedAt:   types.StringValue(out.CreatedAt),
		UpdatedAt:   types.StringValue(out.UpdatedAt),
		SigningKeys: signingKeys,
		Policies:    policies,
	})
	resp.Diagnostics.Append(diags...)
}
