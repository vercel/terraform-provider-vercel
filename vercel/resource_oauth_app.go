package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &oauthAppResource{}
	_ resource.ResourceWithConfigure   = &oauthAppResource{}
	_ resource.ResourceWithImportState = &oauthAppResource{}
)

func newOAuthAppResource() resource.Resource {
	return &oauthAppResource{}
}

type oauthAppResource struct {
	client *client.Client
}

func (r *oauthAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_app"
}

func (r *oauthAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *oauthAppResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an OAuth App resource for [Sign in with Vercel](https://vercel.com/docs/sign-in-with-vercel).

An OAuth App lets people use their Vercel account to log in to your application via OAuth 2.0 / OpenID Connect. Use the ` + "`vercel_oauth_app_client_secret`" + ` resource to generate the client secret your application authenticates with.

~> Managing OAuth Apps requires the Owner role on the team.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The client ID of the OAuth App (`cl_...`). Use this as the OAuth `client_id`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the OAuth App should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "A human-readable name for the OAuth App, shown on the consent page.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 200),
				},
			},
			"slug": schema.StringAttribute{
				Description: "A URL-friendly slug for the OAuth App. Must be unique.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 200),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9-]+$`), "must be lowercase alphanumerics and hyphens"),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the OAuth App, shown on the consent page.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(500),
				},
			},
			"home_page_uri": schema.StringAttribute{
				Description: "The URL of the application's home page.",
				Optional:    true,
			},
			"redirect_uris": schema.SetAttribute{
				Description: "The authorization callback URLs of the OAuth App. Must be absolute `https` URLs (`http` is allowed for loopback addresses only).",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtMost(20),
				},
			},
			"scopes": schema.SetAttribute{
				Description: "The scopes the OAuth App may request: `openid` (always required), `email`, `profile`, and `offline_access` (issues refresh tokens). Defaults to `[\"openid\"]`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(
						"openid",
						"email",
						"profile",
						"offline_access",
					)),
					oauthAppScopesIncludeOpenID(),
				},
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.SetAttribute{
				Description: "Vercel REST API permissions granted to the app's tokens (e.g. `read:team`, `read:project`, `read-write:project`, `read:deployment`, `read-write:deployment`). Shown on the consent page for users to approve. Defaults to none.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"privacy_policy_url": schema.StringAttribute{
				Description: "The URL of the application's privacy policy, shown on the consent page.",
				Optional:    true,
			},
			"terms_of_service_url": schema.StringAttribute{
				Description: "The URL of the application's terms of service, shown on the consent page.",
				Optional:    true,
			},
			"code_of_conduct_url": schema.StringAttribute{
				Description: "The URL of the application's code of conduct.",
				Optional:    true,
			},
		},
	}
}

type OAuthApp struct {
	ID                types.String `tfsdk:"id"`
	TeamID            types.String `tfsdk:"team_id"`
	Name              types.String `tfsdk:"name"`
	Slug              types.String `tfsdk:"slug"`
	Description       types.String `tfsdk:"description"`
	HomePageURI       types.String `tfsdk:"home_page_uri"`
	RedirectURIs      types.Set    `tfsdk:"redirect_uris"`
	Scopes            types.Set    `tfsdk:"scopes"`
	Permissions       types.Set    `tfsdk:"permissions"`
	PrivacyPolicyURL  types.String `tfsdk:"privacy_policy_url"`
	TermsOfServiceURL types.String `tfsdk:"terms_of_service_url"`
	CodeOfConductURL  types.String `tfsdk:"code_of_conduct_url"`
}

// oauthAppStringOrNull keeps unset optional strings null in state: the API
// reports cleared/never-set fields as absent (empty), which should not show as
// a diff against a null config value.
func oauthAppStringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func responseToOAuthApp(ctx context.Context, out client.OAuthApp, prior OAuthApp) (OAuthApp, diag.Diagnostics) {
	redirectURIs := types.SetNull(types.StringType)
	if len(out.RedirectURIs) > 0 || !prior.RedirectURIs.IsNull() {
		var diags diag.Diagnostics
		redirectURIs, diags = types.SetValueFrom(ctx, types.StringType, out.RedirectURIs)
		if diags.HasError() {
			return OAuthApp{}, diags
		}
	}
	permissions := types.SetNull(types.StringType)
	if len(out.Permissions) > 0 || !prior.Permissions.IsNull() {
		var diags diag.Diagnostics
		permissions, diags = types.SetValueFrom(ctx, types.StringType, out.Permissions)
		if diags.HasError() {
			return OAuthApp{}, diags
		}
	}
	scopes, diags := types.SetValueFrom(ctx, types.StringType, out.Scopes)
	if diags.HasError() {
		return OAuthApp{}, diags
	}

	return OAuthApp{
		ID:                types.StringValue(out.ClientID),
		TeamID:            types.StringValue(out.TeamID),
		Name:              types.StringValue(out.Name),
		Slug:              types.StringValue(out.Slug),
		Description:       oauthAppStringOrNull(out.Description),
		HomePageURI:       oauthAppStringOrNull(out.HomePageURI),
		RedirectURIs:      redirectURIs,
		Scopes:            scopes,
		Permissions:       permissions,
		PrivacyPolicyURL:  oauthAppStringOrNull(out.PrivacyPolicyURL),
		TermsOfServiceURL: oauthAppStringOrNull(out.TermsOfServiceURL),
		CodeOfConductURL:  oauthAppStringOrNull(out.CodeOfConductURL),
	}, diags
}

func (r *oauthAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuthApp
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var redirectURIs []string
	if !plan.RedirectURIs.IsNull() && !plan.RedirectURIs.IsUnknown() {
		diags = plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var scopes []string
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var permissions []string
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		diags = plan.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	out, err := r.client.CreateOAuthApp(ctx, client.CreateOAuthAppRequest{
		TeamID:            plan.TeamID.ValueString(),
		Name:              plan.Name.ValueString(),
		Slug:              plan.Slug.ValueString(),
		Description:       plan.Description.ValueString(),
		HomePageURI:       plan.HomePageURI.ValueString(),
		RedirectURIs:      redirectURIs,
		Scopes:            scopes,
		PrivacyPolicyURL:  plan.PrivacyPolicyURL.ValueString(),
		TermsOfServiceURL: plan.TermsOfServiceURL.ValueString(),
		CodeOfConductURL:  plan.CodeOfConductURL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating OAuth App",
			"Could not create OAuth App, unexpected error: "+err.Error(),
		)
		return
	}

	// The create endpoint does not accept permissions — grant them with an
	// immediate follow-up update. On failure the half-configured app is deleted
	// so a failed create never leaves an unmanaged app behind.
	if len(permissions) > 0 {
		out, err = r.client.UpdateOAuthApp(ctx, client.UpdateOAuthAppRequest{
			TeamID:            out.TeamID,
			ClientID:          out.ClientID,
			Name:              out.Name,
			Slug:              out.Slug,
			Description:       out.Description,
			HomePageURI:       oauthAppNullableString(plan.HomePageURI),
			RedirectURIs:      redirectURIs,
			Scopes:            out.Scopes,
			Permissions:       permissions,
			PrivacyPolicyURL:  oauthAppNullableString(plan.PrivacyPolicyURL),
			TermsOfServiceURL: oauthAppNullableString(plan.TermsOfServiceURL),
			CodeOfConductURL:  oauthAppNullableString(plan.CodeOfConductURL),
		})
		if err != nil {
			_ = r.client.DeleteOAuthApp(ctx, out.ClientID, out.TeamID)
			resp.Diagnostics.AddError(
				"Error creating OAuth App",
				"Could not grant permissions to the created OAuth App (the app has been deleted), unexpected error: "+err.Error(),
			)
			return
		}
	}

	result, diags := responseToOAuthApp(ctx, out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created oauth app", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *oauthAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OAuthApp
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetOAuthApp(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.OAuthAppNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading OAuth App",
			fmt.Sprintf("Could not get OAuth App %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToOAuthApp(ctx, out, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read oauth app", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// oauthAppNullableString serializes an optional attribute for the update
// endpoint, where an explicit JSON null clears a previously set value.
func oauthAppNullableString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueString()
	return &value
}

func (r *oauthAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OAuthApp
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state OAuthApp
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The update endpoint treats these as authoritative: an empty list clears
	// redirect URIs, and scopes always includes at least "openid".
	redirectURIs := []string{}
	if !plan.RedirectURIs.IsNull() && !plan.RedirectURIs.IsUnknown() {
		diags = plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var scopes []string
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}
	permissions := []string{}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		diags = plan.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	out, err := r.client.UpdateOAuthApp(ctx, client.UpdateOAuthAppRequest{
		TeamID:            state.TeamID.ValueString(),
		ClientID:          state.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		Slug:              plan.Slug.ValueString(),
		Description:       plan.Description.ValueString(),
		HomePageURI:       oauthAppNullableString(plan.HomePageURI),
		RedirectURIs:      redirectURIs,
		Scopes:            scopes,
		Permissions:       permissions,
		PrivacyPolicyURL:  oauthAppNullableString(plan.PrivacyPolicyURL),
		TermsOfServiceURL: oauthAppNullableString(plan.TermsOfServiceURL),
		CodeOfConductURL:  oauthAppNullableString(plan.CodeOfConductURL),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating OAuth App",
			fmt.Sprintf("Could not update OAuth App %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToOAuthApp(ctx, out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "updated oauth app", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *oauthAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OAuthApp
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOAuthApp(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.OAuthAppNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting OAuth App",
			fmt.Sprintf(
				"Could not delete OAuth App %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted oauth app", map[string]any{
		"team_id":   state.TeamID.ValueString(),
		"client_id": state.ID.ValueString(),
	})
}

func (r *oauthAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, clientID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing OAuth App",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/client_id\" or \"client_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetOAuthApp(ctx, clientID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing OAuth App",
			fmt.Sprintf("Could not get OAuth App %s %s, unexpected error: %s",
				teamID,
				clientID,
				err,
			),
		)
		return
	}

	result, diags := responseToOAuthApp(ctx, out, OAuthApp{
		RedirectURIs: types.SetNull(types.StringType),
		Permissions:  types.SetNull(types.StringType),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "imported oauth app", map[string]any{
		"team_id":   result.TeamID.ValueString(),
		"client_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
