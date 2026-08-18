package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &oauthAppClientSecretResource{}
	_ resource.ResourceWithConfigure = &oauthAppClientSecretResource{}
)

func newOAuthAppClientSecretResource() resource.Resource {
	return &oauthAppClientSecretResource{}
}

type oauthAppClientSecretResource struct {
	client *client.Client
}

func (r *oauthAppClientSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_app_client_secret"
}

func (r *oauthAppClientSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *oauthAppClientSecretResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a client secret for a ` + "`vercel_oauth_app`" + ` ([Sign in with Vercel](https://vercel.com/docs/sign-in-with-vercel)).

The secret value is only ever returned by the API at creation time and is stored (marked sensitive) in the Terraform state. An OAuth App can have at most two client secrets at a time, so zero-downtime rotation is possible by creating a new secret before destroying the old one (e.g. with ` + "`terraform apply -replace`" + ` and ` + "`create_before_destroy`" + `).

~> Managing client secrets requires the Owner role on the team.

-> This resource cannot be imported, as the API never re-exposes the secret value.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The unique identifier of the client secret.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"oauth_app_id": schema.StringAttribute{
				Description:   "The client ID of the OAuth App (`cl_...`) to generate a secret for.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the OAuth App exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"client_secret": schema.StringAttribute{
				Description:   "The generated client secret. Only available at creation time; stored in the Terraform state.",
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"last_four_chars": schema.StringAttribute{
				Description:   "The last four characters of the client secret — the identifier the Vercel API and dashboard use to reference this secret.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

type OAuthAppClientSecret struct {
	ID            types.String `tfsdk:"id"`
	OAuthAppID    types.String `tfsdk:"oauth_app_id"`
	TeamID        types.String `tfsdk:"team_id"`
	ClientSecret  types.String `tfsdk:"client_secret"`
	LastFourChars types.String `tfsdk:"last_four_chars"`
}

func (r *oauthAppClientSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuthAppClientSecret
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateOAuthAppSecret(ctx, plan.OAuthAppID.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating OAuth App Client Secret",
			"Could not create OAuth App Client Secret, unexpected error: "+err.Error(),
		)
		return
	}
	if len(out.ClientSecret) < 4 {
		resp.Diagnostics.AddError(
			"Error creating OAuth App Client Secret",
			"The API returned an unexpectedly short client secret.",
		)
		return
	}

	// The app always belongs to a team; read it back so team_id resolves even
	// when relying on the provider's default team.
	app, err := r.client.GetOAuthApp(ctx, plan.OAuthAppID.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating OAuth App Client Secret",
			"Could not read OAuth App after creating Client Secret, unexpected error: "+err.Error(),
		)
		return
	}

	lastFourChars := out.ClientSecret[len(out.ClientSecret)-4:]
	// The API's secret metadata carries a stable id; resolve it from the app's
	// secret list (matched by the last four characters, which is also how the
	// delete endpoint addresses secrets). Fall back to a synthetic composite if
	// the metadata omits it.
	secretID := fmt.Sprintf("%s/%s", plan.OAuthAppID.ValueString(), lastFourChars)
	for _, secret := range app.ClientSecrets {
		if secret.LastFourChars == lastFourChars && secret.ID != "" {
			secretID = secret.ID
		}
	}

	result := OAuthAppClientSecret{
		ID:            types.StringValue(secretID),
		OAuthAppID:    plan.OAuthAppID,
		TeamID:        types.StringValue(app.TeamID),
		ClientSecret:  types.StringValue(out.ClientSecret),
		LastFourChars: types.StringValue(lastFourChars),
	}
	tflog.Info(ctx, "created oauth app client secret", map[string]any{
		"team_id":         result.TeamID.ValueString(),
		"client_id":       result.OAuthAppID.ValueString(),
		"last_four_chars": result.LastFourChars.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *oauthAppClientSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OAuthAppClientSecret
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetOAuthApp(ctx, state.OAuthAppID.ValueString(), state.TeamID.ValueString())
	if client.OAuthAppNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading OAuth App Client Secret",
			fmt.Sprintf("Could not get OAuth App %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.OAuthAppID.ValueString(),
				err,
			),
		)
		return
	}

	exists := false
	for _, secret := range app.ClientSecrets {
		if secret.LastFourChars == state.LastFourChars.ValueString() {
			exists = true
			break
		}
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "read oauth app client secret", map[string]any{
		"team_id":         state.TeamID.ValueString(),
		"client_id":       state.OAuthAppID.ValueString(),
		"last_four_chars": state.LastFourChars.ValueString(),
	})

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update does nothing: every attribute change requires replacement.
func (r *oauthAppClientSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating an OAuth App Client Secret is not supported",
		"Updating an OAuth App Client Secret is not supported",
	)
}

func (r *oauthAppClientSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OAuthAppClientSecret
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOAuthAppSecret(
		ctx,
		state.OAuthAppID.ValueString(),
		state.LastFourChars.ValueString(),
		state.TeamID.ValueString(),
	)
	if client.OAuthAppNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting OAuth App Client Secret",
			fmt.Sprintf(
				"Could not delete OAuth App Client Secret %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.OAuthAppID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted oauth app client secret", map[string]any{
		"team_id":         state.TeamID.ValueString(),
		"client_id":       state.OAuthAppID.ValueString(),
		"last_four_chars": state.LastFourChars.ValueString(),
	})
}
