package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &projectProtectionInventoryDataSource{}
	_ datasource.DataSourceWithConfigure = &projectProtectionInventoryDataSource{}
)

func newProjectProtectionInventoryDataSource() datasource.DataSource {
	return &projectProtectionInventoryDataSource{}
}

type projectProtectionInventoryDataSource struct{ client *client.Client }

func (d *projectProtectionInventoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protection_inventory"
}

func (d *projectProtectionInventoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
	}
}

func (d *projectProtectionInventoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Reads protection bypasses and access exceptions visible through the public project and alias APIs, without creating, adopting, or revoking them. Includes automation and integration bypasses, URL exceptions, shareable links, individual access, OPTIONS paths, and Trusted Sources, including implicit self-access and Vercel CI access. Bearer secrets and secret-bearing URLs are not stored in state.

This is a snapshot of visible configuration, not a guarantee that a deployment is protected. The public APIs do not establish coverage of deployment-owned access without alias records or guarantee complete visibility for the API token, so complete is currently always false. Missing access maps are reported in incomplete_reasons. Failed requests, invalid JSON, and unsupported Trusted Sources or OPTIONS fields fail the read instead of returning an empty inventory. Expired links, pending requests, invitations, and explicit empty Trusted Sources rules remain in the inventory.

For trusted-project entries, null custom_allow means default matching-environment access (plus development to preview for self-access); an empty set means no access. Vercel CI same-repository access permits same-team CI to preview and production deployments built from the CI repository. This defaults to enabled when not stored. These defaults are access routes even when no explicit exception has been configured.

Approval keys identify the observed access configuration, including credential rotation, target identity, and rule changes. They exclude notes and audit timestamps. They are versioned SHA-256 fingerprints, not API IDs or credentials. Terraform postconditions can reject unapproved entries, but cannot prevent later dashboard changes.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "The project ID."},
			"project_id": schema.StringAttribute{
				Required: true, Description: "The project ID. Names are not accepted so inventory identity survives project renames.",
				Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^prj_[a-zA-Z0-9]+$`), "must be a project ID starting with prj_")},
			},
			"team_id":            schema.StringAttribute{Optional: true, Computed: true, Description: "The team ID. Defaults to the provider's team."},
			"entries":            protectionInventoryEntriesSchema(),
			"complete":           schema.BoolAttribute{Computed: true, Description: "Whether all project access routes are covered. Currently always false because public API coverage and token visibility are not guaranteed."},
			"incomplete_reasons": schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Known limitations and missing access metadata. An empty entries set must not be interpreted as no exceptions when this set is nonempty."},
		},
	}
}

func protectionInventoryEntriesSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		Computed: true, Description: "Observed access entries, including implicit defaults and inactive entries. Entries are not filtered by their status.",
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"approval_key":       schema.StringAttribute{Computed: true, Description: "Versioned fingerprint of the team, project, target, credential or subject, and access configuration. Changes when access changes."},
			"scope":              schema.StringAttribute{Computed: true, Description: "API bypass scope, or options-allowlist, trusted-project, trusted-oidc, or vercel-ci-same-repository. Unrecognized API scopes are retained with unknown status."},
			"target_type":        schema.StringAttribute{Computed: true, Description: "The owner of the entry: project or alias."},
			"target_id":          schema.StringAttribute{Computed: true, Description: "The project or alias ID owning this entry."},
			"alias":              schema.StringAttribute{Computed: true, Description: "Alias hostname, without a query string, or null for project access."},
			"subject":            schema.StringAttribute{Computed: true, Description: "The user or email map key for individual access, or null for other scopes. Bearer map keys are never returned."},
			"status":             schema.StringAttribute{Computed: true, Description: "active, expired, granted, requested, invited, disabled, or unknown. Active describes configuration, not whether protection currently applies to a URL. Invitations are not treated as confirmed grants."},
			"created_at":         schema.Int64Attribute{Computed: true, Description: "Creation time in Unix milliseconds, when returned by the API."},
			"created_by":         schema.StringAttribute{Computed: true, Description: "Creator ID, when returned by the API."},
			"last_updated_at":    schema.Int64Attribute{Computed: true, Description: "Last update time in Unix milliseconds, when returned by the API."},
			"last_updated_by":    schema.StringAttribute{Computed: true, Description: "Last updater ID, when returned by the API."},
			"expires_at":         schema.Int64Attribute{Computed: true, Description: "Expiry time in Unix seconds, or null when no expiry was returned."},
			"is_env_var":         schema.BoolAttribute{Computed: true, Description: "Whether an automation bypass is the deployment environment variable default. Omitted legacy values are interpreted as true. Null for other scopes."},
			"note":               schema.StringAttribute{Computed: true, Description: "Display note or Trusted Source label, when returned by the API. Not used for approval identity."},
			"integration_id":     schema.StringAttribute{Computed: true, Description: "Integration ID for integration-created automation access."},
			"configuration_id":   schema.StringAttribute{Computed: true, Description: "Integration configuration ID for integration-created automation access."},
			"path_prefix":        schema.StringAttribute{Computed: true, Description: "Path prefix bypassed for OPTIONS requests, or null for other scopes."},
			"trusted_project_id": schema.StringAttribute{Computed: true, Description: "Source project ID for trusted-project access."},
			"custom_allow": schema.SetNestedAttribute{
				Computed: true, Description: "Custom environment rules for trusted-project access. Null uses defaults; an empty set explicitly denies access.",
				NestedObject: schema.NestedAttributeObject{Attributes: trustedSourcesAccessRuleDataSourceSchema()},
			},
			"issuer":   schema.StringAttribute{Computed: true, Description: "OIDC issuer for trusted-oidc access."},
			"claims":   schema.MapAttribute{Computed: true, ElementType: trustedSourcesClaimValuesType, Description: "Required OIDC claims and accepted values for trusted-oidc access. Values are ORed within each claim; claims are ANDed."},
			"to":       schema.SingleNestedAttribute{Computed: true, Description: "Allowed target environments for trusted-oidc access.", Attributes: trustedSourcesEnvMatcherDataSourceSchema()},
			"implicit": schema.BoolAttribute{Computed: true, Description: "Whether this entry represents implicit self-access or a Vercel CI default rather than an explicit rule."},
		}},
	}
}

type projectProtectionInventoryModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	TeamID            types.String `tfsdk:"team_id"`
	Entries           types.Set    `tfsdk:"entries"`
	Complete          types.Bool   `tfsdk:"complete"`
	IncompleteReasons types.Set    `tfsdk:"incomplete_reasons"`
}

func (d *projectProtectionInventoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config projectProtectionInventoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inventory, err := d.client.GetProjectProtectionInventory(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading project protection inventory", err.Error())
		return
	}
	entryType := protectionInventoryEntriesSchema().GetType().(types.SetType).ElemType.(types.ObjectType)
	entries := make([]attr.Value, 0, len(inventory.Entries))
	for _, entry := range inventory.Entries {
		customAllow := types.SetNull(trustedSourcesAccessRuleAttrType)
		if entry.CustomAllow != nil {
			rules := make([]attr.Value, 0, len(entry.CustomAllow))
			for _, rule := range entry.CustomAllow {
				rules = append(rules, types.ObjectValueMust(trustedSourcesAccessRuleAttrType.AttrTypes, map[string]attr.Value{
					"from": trustedSourcesEnvMatcherObjectFromResponse(rule.From),
					"to":   trustedSourcesEnvMatcherObjectFromResponse(rule.To),
				}))
			}
			customAllow = types.SetValueMust(trustedSourcesAccessRuleAttrType, rules)
		}
		to := types.ObjectNull(trustedSourcesEnvMatcherAttrType.AttrTypes)
		if entry.To != nil {
			to = trustedSourcesEnvMatcherObjectFromResponse(*entry.To)
		}
		entries = append(entries, types.ObjectValueMust(entryType.AttrTypes, map[string]attr.Value{
			"approval_key": types.StringValue(entry.ApprovalKey), "scope": types.StringValue(entry.Scope),
			"target_type": types.StringValue(entry.TargetType), "target_id": types.StringValue(entry.TargetID),
			"alias": types.StringPointerValue(entry.Alias), "subject": types.StringPointerValue(entry.Subject),
			"status": types.StringValue(entry.Status), "created_at": types.Int64PointerValue(entry.CreatedAt),
			"created_by": types.StringPointerValue(entry.CreatedBy), "last_updated_at": types.Int64PointerValue(entry.LastUpdatedAt),
			"last_updated_by": types.StringPointerValue(entry.LastUpdatedBy), "expires_at": types.Int64PointerValue(entry.ExpiresAt),
			"is_env_var": types.BoolPointerValue(entry.IsEnvVar), "note": types.StringPointerValue(entry.Note),
			"integration_id": types.StringPointerValue(entry.IntegrationID), "configuration_id": types.StringPointerValue(entry.ConfigurationID),
			"path_prefix": types.StringPointerValue(entry.PathPrefix), "trusted_project_id": types.StringPointerValue(entry.TrustedProjectID),
			"custom_allow": customAllow, "issuer": types.StringPointerValue(entry.Issuer),
			"claims": trustedSourcesClaimsMapFromResponse(entry.Claims), "to": to, "implicit": types.BoolValue(entry.Implicit),
		}))
	}
	config.ID = types.StringValue(inventory.ProjectID)
	config.TeamID = toTeamID(inventory.TeamID)
	config.Entries = types.SetValueMust(entryType, entries)
	config.Complete = types.BoolValue(false)
	reasons, diags := types.SetValueFrom(ctx, types.StringType, inventory.IncompleteReasons)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.IncompleteReasons = reasons
	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
