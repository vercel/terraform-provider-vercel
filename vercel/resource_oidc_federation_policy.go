package vercel

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                     = &oidcFederationPolicyResource{}
	_ resource.ResourceWithConfigure        = &oidcFederationPolicyResource{}
	_ resource.ResourceWithConfigValidators = &oidcFederationPolicyResource{}
	_ resource.ResourceWithImportState      = &oidcFederationPolicyResource{}
)

func newOIDCFederationPolicyResource() resource.Resource {
	return &oidcFederationPolicyResource{}
}

type oidcFederationPolicyResource struct {
	client *client.Client
}

const (
	oidcFederationClientTurborepo = "turborepo"
	oidcFederationClientVercel    = "vercel"
	turborepoCLIClientID          = "cl_kyUx2zVvA4MGptBohkmtYHJly2XltXzD"
	vercelCLIClientID             = "cl_HYyOPBNtFMfHhaUn9L4QPfTZz6TP47bp"
)

type oidcFederationPolicyModel struct {
	ID          types.String                  `tfsdk:"id"`
	TeamID      types.String                  `tfsdk:"team_id"`
	Client      types.String                  `tfsdk:"client"`
	IssuerURL   types.String                  `tfsdk:"issuer_url"`
	Name        types.String                  `tfsdk:"name"`
	Claims      []oidcFederationClaimModel    `tfsdk:"claims"`
	Permissions types.Set                     `tfsdk:"permissions"`
	Commands    types.Set                     `tfsdk:"commands"`
	Resources   *oidcFederationResourcesModel `tfsdk:"resources"`
}

type oidcFederationClaimModel struct {
	Name   types.String                    `tfsdk:"name"`
	Values []oidcFederationClaimValueModel `tfsdk:"values"`
}

type oidcFederationClaimValueModel struct {
	Value     types.String `tfsdk:"value"`
	Wildcards types.Bool   `tfsdk:"wildcards"`
}

type oidcFederationResourcesModel struct {
	ProjectIDs types.Set `tfsdk:"project_ids"`
}

func (r *oidcFederationPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_federation_policy"
}

func (r *oidcFederationPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = configuredClient
}

func (r *oidcFederationPolicyResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("permissions"),
			path.MatchRoot("commands"),
		),
	}
}

func (r *oidcFederationPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an OIDC Federation Policy resource.

OIDC federation policies allow trusted external workloads to exchange an OIDC token for a short-lived Vercel access token. The client selects the Vercel CLI receiving access, while claims constrain the external workload identity.

~> This API currently supports first-party CLI clients enabled for the team, including the Vercel CLI and Turborepo CLI. The API token used by the provider must have permission to manage OIDC federation policies for the team.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the OIDC federation policy.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team the policy should exist under. Required when a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"client": schema.StringAttribute{
				Description:   "The CLI whose access tokens this policy permits the workload to obtain. Valid values are `turborepo` and `vercel`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf(oidcFederationClientTurborepo, oidcFederationClientVercel),
				},
			},
			"issuer_url": schema.StringAttribute{
				Description:   "The HTTPS issuer URL in OIDC tokens recognized by this policy.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^https://[^/?#]+(?:/[^/?#]+)*$`), "The issuer URL must use HTTPS, must not contain a query or fragment, and must not end with a slash."),
					stringvalidator.LengthBetween(1, 2048),
				},
			},
			"name": schema.StringAttribute{
				Description: "A human-readable name describing the policy.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
				},
			},
			"claims": schema.ListNestedAttribute{
				Description: "OIDC claims that must all match for a token to be exchanged.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The OIDC claim name.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 256),
							},
						},
						"values": schema.ListNestedAttribute{
							Description: "Values accepted for this claim. A claim matches when any configured value matches.",
							Required:    true,
							Validators: []validator.List{
								listvalidator.SizeBetween(1, 50),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"value": schema.StringAttribute{
										Description: "The accepted claim value or wildcard pattern.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthBetween(1, 1024),
										},
									},
									"wildcards": schema.BoolAttribute{
										Description: "Whether `*` characters in the value should be interpreted as wildcards.",
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
									},
								},
							},
						},
					},
				},
			},
			"permissions": schema.SetAttribute{
				Description: "Permissions granted to exchanged tokens. Use `[\"*\"]` for every permission authorized for the selected client. Exactly one of `permissions` or `commands` must be configured.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, 64),
				},
			},
			"commands": schema.SetAttribute{
				Description: "Vercel CLI command IDs allowed for exchanged tokens. Only supported for the Vercel CLI client. Exactly one of `commands` or `permissions` must be configured.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, 256),
				},
			},
			"resources": schema.SingleNestedAttribute{
				Description:   "Optional project boundary for exchanged tokens. Omitting this block leaves access unbounded within the permissions authorized for the client.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"project_ids": schema.SetAttribute{
						Description: "Project IDs in the resource boundary. Use `[\"*\"]` for all current and future team projects, or an empty set for no projects.",
						Required:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func oidcFederationClientID(clientName string) string {
	switch clientName {
	case oidcFederationClientTurborepo:
		return turborepoCLIClientID
	case oidcFederationClientVercel:
		return vercelCLIClientID
	default:
		return ""
	}
}

func oidcFederationClientName(clientID string) (string, bool) {
	switch clientID {
	case turborepoCLIClientID:
		return oidcFederationClientTurborepo, true
	case vercelCLIClientID:
		return oidcFederationClientVercel, true
	default:
		return "", false
	}
}

func oidcFederationClaimsToClient(claims []oidcFederationClaimModel) []client.OIDCClaim {
	result := make([]client.OIDCClaim, 0, len(claims))
	for _, claim := range claims {
		values := make([]client.OIDCClaimValue, 0, len(claim.Values))
		for _, value := range claim.Values {
			values = append(values, client.OIDCClaimValue{
				Value:     value.Value.ValueString(),
				Wildcards: value.Wildcards.ValueBool(),
			})
		}
		result = append(result, client.OIDCClaim{
			Name:   claim.Name.ValueString(),
			Values: values,
		})
	}
	return result
}

func oidcFederationStringSet(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result []string
	diags := value.ElementsAs(ctx, &result, false)
	return result, diags
}

func oidcFederationPolicyFromClient(ctx context.Context, out client.OIDCFederationPolicy) (oidcFederationPolicyModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	clientName, ok := oidcFederationClientName(out.ClientID)
	if !ok {
		diags.AddError("Unsupported OIDC federation policy client", fmt.Sprintf("The policy uses unsupported client ID %q.", out.ClientID))
	}

	permissions, permissionDiags := types.SetValueFrom(ctx, types.StringType, out.Permissions)
	diags.Append(permissionDiags...)

	commands := types.SetNull(types.StringType)
	if out.Commands != nil {
		var commandDiags diag.Diagnostics
		commands, commandDiags = types.SetValueFrom(ctx, types.StringType, out.Commands)
		diags.Append(commandDiags...)
	}

	claims := make([]oidcFederationClaimModel, 0, len(out.Claims))
	for _, claim := range out.Claims {
		values := make([]oidcFederationClaimValueModel, 0, len(claim.Values))
		for _, value := range claim.Values {
			values = append(values, oidcFederationClaimValueModel{
				Value:     types.StringValue(value.Value),
				Wildcards: types.BoolValue(value.Wildcards),
			})
		}
		claims = append(claims, oidcFederationClaimModel{
			Name:   types.StringValue(claim.Name),
			Values: values,
		})
	}

	var resources *oidcFederationResourcesModel
	if out.Resources != nil {
		projectIDs, projectDiags := types.SetValueFrom(ctx, types.StringType, out.Resources.ProjectIDs)
		diags.Append(projectDiags...)
		resources = &oidcFederationResourcesModel{ProjectIDs: projectIDs}
	}

	name := types.StringNull()
	if out.Name != nil {
		name = types.StringValue(*out.Name)
	}

	return oidcFederationPolicyModel{
		ID:          types.StringValue(out.PolicyID),
		TeamID:      toTeamID(out.TeamID),
		Client:      types.StringValue(clientName),
		IssuerURL:   types.StringValue(out.IssuerURL),
		Name:        name,
		Claims:      claims,
		Permissions: permissions,
		Commands:    commands,
		Resources:   resources,
	}, diags
}

func (r *oidcFederationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oidcFederationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions, diags := oidcFederationStringSet(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	commands, diags := oidcFederationStringSet(ctx, plan.Commands)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := client.CreateOIDCFederationPolicyRequest{
		TeamID:      plan.TeamID.ValueString(),
		ClientID:    oidcFederationClientID(plan.Client.ValueString()),
		IssuerURL:   plan.IssuerURL.ValueString(),
		Claims:      oidcFederationClaimsToClient(plan.Claims),
		Permissions: permissions,
		Commands:    commands,
	}
	if !plan.Name.IsNull() {
		name := plan.Name.ValueString()
		request.Name = &name
	}
	if plan.Resources != nil {
		projectIDs, projectDiags := oidcFederationStringSet(ctx, plan.Resources.ProjectIDs)
		resp.Diagnostics.Append(projectDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Resources = &client.OIDCResources{ProjectIDs: projectIDs}
	}

	out, err := r.client.CreateOIDCFederationPolicy(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating OIDC Federation Policy", "Could not create OIDC Federation Policy, unexpected error: "+err.Error())
		return
	}

	result, diags := oidcFederationPolicyFromClient(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created OIDC federation policy", map[string]any{"team_id": result.TeamID.ValueString(), "policy_id": result.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *oidcFederationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oidcFederationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetOIDCFederationPolicy(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading OIDC Federation Policy", fmt.Sprintf("Could not get OIDC Federation Policy %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ID.ValueString(), err))
		return
	}

	result, diags := oidcFederationPolicyFromClient(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *oidcFederationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oidcFederationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	claims := oidcFederationClaimsToClient(plan.Claims)
	name := plan.Name.ValueString()
	request := client.UpdateOIDCFederationPolicyRequest{
		TeamID:   plan.TeamID.ValueString(),
		PolicyID: plan.ID.ValueString(),
		Name:     &name,
		Claims:   &claims,
	}
	if !plan.Commands.IsNull() {
		commands, diags := oidcFederationStringSet(ctx, plan.Commands)
		resp.Diagnostics.Append(diags...)
		request.Commands = &commands
	} else {
		permissions, diags := oidcFederationStringSet(ctx, plan.Permissions)
		resp.Diagnostics.Append(diags...)
		request.Permissions = &permissions
	}
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateOIDCFederationPolicy(ctx, request)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating OIDC Federation Policy", fmt.Sprintf("Could not update OIDC Federation Policy %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ID.ValueString(), err))
		return
	}

	result, diags := oidcFederationPolicyFromClient(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *oidcFederationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oidcFederationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOIDCFederationPolicy(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting OIDC Federation Policy", fmt.Sprintf("Could not delete OIDC Federation Policy %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "deleted OIDC federation policy", map[string]any{"team_id": state.TeamID.ValueString(), "policy_id": state.ID.ValueString()})
}

func (r *oidcFederationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, policyID, ok := splitInto1Or2(req.ID)
	if !ok || policyID == "" || (strings.Contains(req.ID, "/") && teamID == "") {
		resp.Diagnostics.AddError("Error importing OIDC Federation Policy", fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/policy_id\" or \"policy_id\"", req.ID))
		return
	}

	out, err := r.client.GetOIDCFederationPolicy(ctx, policyID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error importing OIDC Federation Policy", fmt.Sprintf("Could not get OIDC Federation Policy %s %s, unexpected error: %s", teamID, policyID, err))
		return
	}

	result, diags := oidcFederationPolicyFromClient(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
