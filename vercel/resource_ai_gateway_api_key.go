package vercel

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &aiGatewayAPIKeyResource{}
	_ resource.ResourceWithConfigure   = &aiGatewayAPIKeyResource{}
	_ resource.ResourceWithImportState = &aiGatewayAPIKeyResource{}
)

func newAIGatewayAPIKeyResource() resource.Resource {
	return &aiGatewayAPIKeyResource{}
}

type aiGatewayAPIKeyResource struct {
	client *client.Client
}

func (r *aiGatewayAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_gateway_api_key"
}

func (r *aiGatewayAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aiGatewayAPIKeyResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an AI Gateway API Key resource.

An AI Gateway API Key can be used to authenticate with the [Vercel AI Gateway](https://vercel.com/docs/ai-gateway).

The ` + "`api_key_string`" + ` value is only returned during creation and is stored (marked sensitive) in Terraform state. It cannot be retrieved again later, so imported resources will not populate ` + "`api_key_string`" + `.

-> Managing ` + "`ai_gateway_quota`" + ` requires the AI Gateway API key quotas feature to be enabled for the team.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The unique identifier of the API key.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The human-readable name of the API key.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
				},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the Vercel team scope for this API key. Required if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of a project to restrict the API key to. When unset, the API key grants access to all projects in the team. Cannot be changed after creation.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"expires_at": schema.Int64Attribute{
				Description:   "The Unix timestamp in milliseconds when the API key should expire. Must not be in the past or more than two years in the future. Cannot be changed after creation.",
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"ai_gateway_quota": schema.SingleNestedAttribute{
				Description: "A spend quota (budget) for the API key. Removing this attribute archives the quota; the API key itself is unaffected.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"limit_amount": schema.Float64Attribute{
						Description: "The quota limit amount in US dollars.",
						Required:    true,
						Validators: []validator.Float64{
							float64validator.AtLeast(1),
						},
					},
					"refresh_period": schema.StringAttribute{
						Description: "How often the quota refreshes. Must be one of `daily`, `weekly`, `monthly` or `none`. Defaults to `none`.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("none"),
						Validators: []validator.String{
							stringvalidator.OneOf("daily", "weekly", "monthly", "none"),
						},
					},
					"alert_thresholds": schema.SetAttribute{
						Description: "Spend percentages (a subset of `[50, 75, 100]`) at which to send a spend alert.",
						Optional:    true,
						ElementType: types.Int64Type,
						Validators: []validator.Set{
							setvalidator.ValueInt64sAre(int64validator.OneOf(50, 75, 100)),
						},
					},
				},
			},
			"api_key_string": schema.StringAttribute{
				Description:   "The API key value. This is only returned during creation and then preserved in Terraform state.",
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"partial_key": schema.StringAttribute{
				Description:   "The final characters of the API key, used for identification.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

type AIGatewayAPIKey struct {
	ID           types.String          `tfsdk:"id"`
	Name         types.String          `tfsdk:"name"`
	TeamID       types.String          `tfsdk:"team_id"`
	ProjectID    types.String          `tfsdk:"project_id"`
	ExpiresAt    types.Int64           `tfsdk:"expires_at"`
	Quota        *AIGatewayAPIKeyQuota `tfsdk:"ai_gateway_quota"`
	APIKeyString types.String          `tfsdk:"api_key_string"`
	PartialKey   types.String          `tfsdk:"partial_key"`
}

type AIGatewayAPIKeyQuota struct {
	LimitAmount     types.Float64 `tfsdk:"limit_amount"`
	RefreshPeriod   types.String  `tfsdk:"refresh_period"`
	AlertThresholds types.Set     `tfsdk:"alert_thresholds"`
}

func (q *AIGatewayAPIKeyQuota) alertThresholds(ctx context.Context) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	if q.AlertThresholds.IsNull() || q.AlertThresholds.IsUnknown() {
		return nil, diags
	}
	var thresholds []int64
	diags = q.AlertThresholds.ElementsAs(ctx, &thresholds, false)
	return thresholds, diags
}

func responseToAIGatewayAPIKey(out client.AIGatewayAPIKey, apiKeyString types.String) AIGatewayAPIKey {
	if out.APIKeyString != nil {
		apiKeyString = types.StringPointerValue(out.APIKeyString)
	}

	return AIGatewayAPIKey{
		ID:           types.StringValue(out.ID),
		Name:         types.StringValue(out.Name),
		TeamID:       types.StringValue(out.TeamID),
		ProjectID:    types.StringPointerValue(out.ProjectID),
		ExpiresAt:    types.Int64PointerValue(out.ExpiresAt),
		APIKeyString: apiKeyString,
		PartialKey:   types.StringValue(out.PartialKey),
	}
}

// quotaFromResponse converts an API quota into state, normalizing empty or
// absent alert thresholds against the prior state so a null configuration
// does not show perpetual drift against an empty server-side list.
func quotaFromResponse(quota *client.AIGatewayAPIKeyQuota, prior *AIGatewayAPIKeyQuota) *AIGatewayAPIKeyQuota {
	if quota == nil {
		return nil
	}

	thresholds := types.SetNull(types.Int64Type)
	if len(quota.AlertThresholds) > 0 {
		values := make([]attr.Value, 0, len(quota.AlertThresholds))
		for _, threshold := range quota.AlertThresholds {
			values = append(values, types.Int64Value(threshold))
		}
		thresholds = types.SetValueMust(types.Int64Type, values)
	} else if prior != nil && !prior.AlertThresholds.IsNull() && !prior.AlertThresholds.IsUnknown() && len(prior.AlertThresholds.Elements()) == 0 {
		thresholds = prior.AlertThresholds
	}

	return &AIGatewayAPIKeyQuota{
		LimitAmount:     types.Float64Value(quota.LimitAmount),
		RefreshPeriod:   types.StringValue(quota.RefreshPeriod),
		AlertThresholds: thresholds,
	}
}

func quotasEqual(a, b *AIGatewayAPIKeyQuota) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.LimitAmount.Equal(b.LimitAmount) &&
		a.RefreshPeriod.Equal(b.RefreshPeriod) &&
		a.AlertThresholds.Equal(b.AlertThresholds)
}

func clientQuotasEqual(a, b *client.AIGatewayAPIKeyQuota) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.LimitAmount != b.LimitAmount || a.RefreshPeriod != b.RefreshPeriod {
		return false
	}
	aThresholds := append([]int64(nil), a.AlertThresholds...)
	bThresholds := append([]int64(nil), b.AlertThresholds...)
	slices.Sort(aThresholds)
	slices.Sort(bThresholds)
	return slices.Equal(aThresholds, bThresholds)
}

// waitForQuota polls the list endpoint until it reflects a quota write. Quota
// reads are eventually consistent with writes, so without this a refresh
// immediately after an apply can observe stale quota values. A timeout is
// reported as a warning rather than an error: the write itself succeeded, and
// any phantom drift resolves once the change propagates.
func (r *aiGatewayAPIKeyResource) waitForQuota(ctx context.Context, keyID, teamID string, want *client.AIGatewayAPIKeyQuota, diags *diag.Diagnostics) {
	deadline := time.Now().Add(90 * time.Second)
	for {
		got, err := r.client.GetAIGatewayAPIKeyQuota(ctx, keyID, teamID)
		if err == nil && clientQuotasEqual(got, want) {
			return
		}
		if time.Now().After(deadline) {
			diags.AddWarning(
				"AI Gateway API Key quota change is still propagating",
				fmt.Sprintf("The quota change for AI Gateway API Key %s was applied, but is not yet reflected by the Vercel API. A subsequent plan may show a difference until the change propagates.", keyID),
			)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (r *aiGatewayAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AIGatewayAPIKey
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(plan.Name.ValueString())
	if name == "" || name != plan.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Invalid AI Gateway API key name",
			"API key name cannot be empty, whitespace, or have leading or trailing whitespace.",
		)
		return
	}
	if r.client.TeamID(plan.TeamID.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("team_id"),
			"AI Gateway API key requires team scope",
			"Set `team_id` or configure a default team on the provider.",
		)
		return
	}

	var quota *client.AIGatewayAPIKeyQuota
	if plan.Quota != nil {
		thresholds, diags := plan.Quota.alertThresholds(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		quota = &client.AIGatewayAPIKeyQuota{
			LimitAmount:     plan.Quota.LimitAmount.ValueFloat64(),
			RefreshPeriod:   plan.Quota.RefreshPeriod.ValueString(),
			AlertThresholds: thresholds,
		}
	}

	out, err := r.client.CreateAIGatewayAPIKey(ctx, client.CreateAIGatewayAPIKeyRequest{
		Name:           name,
		Purpose:        "ai-gateway",
		ExpiresAt:      plan.ExpiresAt.ValueInt64Pointer(),
		ProjectID:      plan.ProjectID.ValueStringPointer(),
		AIGatewayQuota: quota,
		TeamID:         plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AI Gateway API Key",
			"Could not create AI Gateway API Key, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToAIGatewayAPIKey(out, types.StringNull())
	// The create response does not include quota information; it reflects
	// what was just requested.
	result.Quota = plan.Quota
	if quota != nil {
		r.waitForQuota(ctx, result.ID.ValueString(), result.TeamID.ValueString(), quota, &resp.Diagnostics)
	}
	tflog.Info(ctx, "created AI Gateway API key", map[string]any{
		"key_id":  result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *aiGatewayAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AIGatewayAPIKey
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAIGatewayAPIKey(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading AI Gateway API Key",
			fmt.Sprintf("Could not get AI Gateway API Key %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	quota, err := r.client.GetAIGatewayAPIKeyQuota(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading AI Gateway API Key",
			fmt.Sprintf("Could not get quota for AI Gateway API Key %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	result := responseToAIGatewayAPIKey(out, state.APIKeyString)
	result.Quota = quotaFromResponse(quota, state.Quota)
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *aiGatewayAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AIGatewayAPIKey
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state AIGatewayAPIKey
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(plan.Name.ValueString())
	if name == "" || name != plan.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Invalid AI Gateway API key name",
			"API key name cannot be empty, whitespace, or have leading or trailing whitespace.",
		)
		return
	}

	result := AIGatewayAPIKey{
		ID:           state.ID,
		Name:         plan.Name,
		TeamID:       state.TeamID,
		ProjectID:    plan.ProjectID,
		ExpiresAt:    plan.ExpiresAt,
		Quota:        plan.Quota,
		APIKeyString: state.APIKeyString,
		PartialKey:   state.PartialKey,
	}

	if !plan.Name.Equal(state.Name) {
		out, err := r.client.UpdateAIGatewayAPIKey(ctx, client.UpdateAIGatewayAPIKeyRequest{
			KeyID:  state.ID.ValueString(),
			TeamID: state.TeamID.ValueString(),
			Name:   name,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating AI Gateway API Key",
				fmt.Sprintf("Could not update AI Gateway API Key %s, unexpected error: %s", state.ID.ValueString(), err),
			)
			return
		}
		result.Name = types.StringValue(out.Name)
	}

	if !quotasEqual(plan.Quota, state.Quota) {
		var request client.UpdateAIGatewayAPIKeyQuotaRequest
		var want *client.AIGatewayAPIKeyQuota
		if plan.Quota != nil {
			thresholds, diags := plan.Quota.alertThresholds(ctx)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if thresholds == nil {
				// Explicitly clear alerts when the configuration omits them.
				thresholds = []int64{}
			}
			request = client.UpdateAIGatewayAPIKeyQuotaRequest{
				KeyID:           state.ID.ValueString(),
				TeamID:          state.TeamID.ValueString(),
				LimitAmount:     plan.Quota.LimitAmount.ValueFloat64Pointer(),
				RefreshPeriod:   plan.Quota.RefreshPeriod.ValueString(),
				AlertThresholds: &thresholds,
			}
			want = &client.AIGatewayAPIKeyQuota{
				LimitAmount:     plan.Quota.LimitAmount.ValueFloat64(),
				RefreshPeriod:   plan.Quota.RefreshPeriod.ValueString(),
				AlertThresholds: thresholds,
			}
		} else {
			archived := true
			request = client.UpdateAIGatewayAPIKeyQuotaRequest{
				KeyID:    state.ID.ValueString(),
				TeamID:   state.TeamID.ValueString(),
				Archived: &archived,
			}
		}

		quota, err := r.client.UpdateAIGatewayAPIKeyQuota(ctx, request)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating AI Gateway API Key quota",
				fmt.Sprintf("Could not update quota for AI Gateway API Key %s, unexpected error: %s", state.ID.ValueString(), err),
			)
			return
		}
		if plan.Quota != nil {
			result.Quota = quotaFromResponse(quota, plan.Quota)
		}
		r.waitForQuota(ctx, state.ID.ValueString(), state.TeamID.ValueString(), want, &resp.Diagnostics)
	}

	tflog.Info(ctx, "updated AI Gateway API key", map[string]any{
		"key_id":  result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *aiGatewayAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AIGatewayAPIKey
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAIGatewayAPIKey(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting AI Gateway API Key",
			fmt.Sprintf("Could not delete AI Gateway API Key %s, unexpected error: %s", state.ID.ValueString(), err),
		)
	}
}

func (r *aiGatewayAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, keyID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing AI Gateway API Key",
			fmt.Sprintf("Invalid id '%s' specified. Should be in format \"team_id/key_id\" or \"key_id\".", req.ID),
		)
		return
	}

	out, err := r.client.GetAIGatewayAPIKey(ctx, keyID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing AI Gateway API Key",
			fmt.Sprintf("Could not get AI Gateway API Key %s, unexpected error: %s", keyID, err),
		)
		return
	}

	quota, err := r.client.GetAIGatewayAPIKeyQuota(ctx, keyID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing AI Gateway API Key",
			fmt.Sprintf("Could not get quota for AI Gateway API Key %s, unexpected error: %s", keyID, err),
		)
		return
	}

	result := responseToAIGatewayAPIKey(out, types.StringNull())
	result.Quota = quotaFromResponse(quota, nil)
	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
