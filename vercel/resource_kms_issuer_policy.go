package vercel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

const kmsProjectGrantKind = "project-grant"

var (
	_ resource.Resource                = &kmsIssuerPolicyResource{}
	_ resource.ResourceWithConfigure   = &kmsIssuerPolicyResource{}
	_ resource.ResourceWithImportState = &kmsIssuerPolicyResource{}
)

func newKMSIssuerPolicyResource() resource.Resource {
	return &kmsIssuerPolicyResource{}
}

type kmsIssuerPolicyResource struct {
	client *client.Client
}

type kmsIssuerPolicyResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TeamID       types.String `tfsdk:"team_id"`
	IssuerID     types.String `tfsdk:"issuer_id"`
	ProjectID    types.String `tfsdk:"project_id"`
	Environments types.List   `tfsdk:"environments"`
	TokenClaims  types.String `tfsdk:"token_claims"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *kmsIssuerPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_issuer_policy"
}

func (r *kmsIssuerPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *kmsIssuerPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel KMS Issuer project-grant policy.

A project-grant policy authorizes a Vercel project to request signed JWTs from a
KMS issuer in the listed environments, optionally with additional token claims.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier for this resource. Format: issuer_id/project_id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the issuer exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"issuer_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the issuer this policy is attached to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project being granted access.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environments": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "The environments in which the project may request signed tokens (for example `production`, `preview`).",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"token_claims": schema.StringAttribute{
				Optional:    true,
				Description: "Additional claims KMS should include in signed JWTs for this policy, as a JSON-encoded object.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "The time the policy was created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time the policy was last updated.",
			},
		},
	}
}

func kmsFindProjectGrantPolicy(policies []client.KMSIssuerPolicy, projectID string) (client.KMSIssuerPolicy, bool) {
	for _, policy := range policies {
		if policy.Kind == kmsProjectGrantKind && policy.ProjectID == projectID {
			return policy, true
		}
	}
	return client.KMSIssuerPolicy{}, false
}

func (r *kmsIssuerPolicyResource) modelFromResponse(ctx context.Context, issuerID, projectID string, teamID types.String, policy client.KMSIssuerPolicy, prior kmsIssuerPolicyResourceModel) (kmsIssuerPolicyResourceModel, diag.Diagnostics) {
	environments, diags := kmsEnvironmentsValue(ctx, policy.Environments, prior.Environments)
	model := kmsIssuerPolicyResourceModel{
		ID:           types.StringValue(fmt.Sprintf("%s/%s", issuerID, projectID)),
		TeamID:       teamID,
		IssuerID:     types.StringValue(issuerID),
		ProjectID:    types.StringValue(projectID),
		Environments: environments,
		TokenClaims:  kmsTokenClaimsValue(policy.TokenClaims, prior.TokenClaims),
		CreatedAt:    types.StringValue(policy.CreatedAt),
		UpdatedAt:    types.StringValue(policy.UpdatedAt),
	}
	return model, diags
}

func kmsTokenClaimsRaw(value types.String) json.RawMessage {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	return json.RawMessage(value.ValueString())
}

func (r *kmsIssuerPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan kmsIssuerPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environments []string
	diags = plan.Environments.ElementsAs(ctx, &environments, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateKMSIssuerPolicy(ctx, client.CreateKMSIssuerPolicyRequest{
		IssuerID:     plan.IssuerID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
		ProjectID:    plan.ProjectID.ValueString(),
		Environments: environments,
		TokenClaims:  kmsTokenClaimsRaw(plan.TokenClaims),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating KMS Issuer Policy",
			"Could not create KMS Issuer Policy, unexpected error: "+err.Error(),
		)
		return
	}

	teamID := toTeamID(r.client.TeamID(plan.TeamID.ValueString()))
	result, diags := r.modelFromResponse(ctx, plan.IssuerID.ValueString(), plan.ProjectID.ValueString(), teamID, out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created kms issuer policy", map[string]any{
		"issuer_id":  plan.IssuerID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsIssuerPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state kmsIssuerPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, state.IssuerID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading KMS Issuer Policy",
			fmt.Sprintf("Could not read KMS Issuer %s, unexpected error: %s", state.IssuerID.ValueString(), err),
		)
		return
	}

	policy, ok := kmsFindProjectGrantPolicy(out.Policies, state.ProjectID.ValueString())
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	result, diags := r.modelFromResponse(ctx, state.IssuerID.ValueString(), state.ProjectID.ValueString(), toTeamID(r.client.TeamID(state.TeamID.ValueString())), policy, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsIssuerPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan kmsIssuerPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environments []string
	diags = plan.Environments.ElementsAs(ctx, &environments, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateKMSIssuerPolicy(ctx, client.UpdateKMSIssuerPolicyRequest{
		IssuerID:     plan.IssuerID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
		ProjectID:    plan.ProjectID.ValueString(),
		Environments: environments,
		TokenClaims:  kmsTokenClaimsRaw(plan.TokenClaims),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating KMS Issuer Policy",
			fmt.Sprintf("Could not update KMS Issuer Policy %s/%s, unexpected error: %s", plan.IssuerID.ValueString(), plan.ProjectID.ValueString(), err),
		)
		return
	}

	teamID := toTeamID(r.client.TeamID(plan.TeamID.ValueString()))
	result, diags := r.modelFromResponse(ctx, plan.IssuerID.ValueString(), plan.ProjectID.ValueString(), teamID, out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *kmsIssuerPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state kmsIssuerPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteKMSIssuerPolicy(ctx, state.IssuerID.ValueString(), state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting KMS Issuer Policy",
			fmt.Sprintf("Could not delete KMS Issuer Policy %s/%s, unexpected error: %s", state.IssuerID.ValueString(), state.ProjectID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted kms issuer policy", map[string]any{
		"issuer_id":  state.IssuerID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

func (r *kmsIssuerPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, issuerID, projectID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing KMS Issuer Policy",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/issuer_id/project_id\" or \"issuer_id/project_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetKMSIssuer(ctx, issuerID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing KMS Issuer Policy",
			fmt.Sprintf("Could not import KMS Issuer Policy %s %s/%s, unexpected error: %s", teamID, issuerID, projectID, err),
		)
		return
	}

	policy, ok := kmsFindProjectGrantPolicy(out.Policies, projectID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing KMS Issuer Policy",
			fmt.Sprintf("No project-grant policy for project %s found on issuer %s", projectID, issuerID),
		)
		return
	}

	result, diags := r.modelFromResponse(ctx, issuerID, projectID, toTeamID(r.client.TeamID(teamID)), policy, kmsIssuerPolicyResourceModel{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
