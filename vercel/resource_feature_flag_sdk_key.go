package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &featureFlagSDKKeyResource{}
	_ resource.ResourceWithConfigure   = &featureFlagSDKKeyResource{}
	_ resource.ResourceWithImportState = &featureFlagSDKKeyResource{}
)

func newFeatureFlagSDKKeyResource() resource.Resource {
	return &featureFlagSDKKeyResource{}
}

type featureFlagSDKKeyResource struct {
	client *client.Client
}

type featureFlagSDKKeyModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	TeamID           types.String `tfsdk:"team_id"`
	Environment      types.String `tfsdk:"environment"`
	Type             types.String `tfsdk:"type"`
	Label            types.String `tfsdk:"label"`
	SDKKey           types.String `tfsdk:"sdk_key"`
	ConnectionString types.String `tfsdk:"connection_string"`
}

func (r *featureFlagSDKKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_sdk_key"
}

func (r *featureFlagSDKKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *featureFlagSDKKeyResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Feature Flag SDK Key resource.

SDK keys are project-scoped and environment-specific. Vercel only returns the cleartext SDK key and connection string at creation time, so this resource preserves the existing sensitive values in state when later reads omit them.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The hash key identifier for the SDK key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project that owns the SDK key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The environment this SDK key should authenticate against.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				Description:   "The SDK key type. Must be one of `server`, `mobile`, or `client`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("server", "mobile", "client"),
				},
			},
			"label": schema.StringAttribute{
				Optional:      true,
				Description:   "An optional label to help identify this key in the Vercel dashboard.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"sdk_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The cleartext SDK key. Vercel only returns this at creation time.",
			},
			"connection_string": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The connection string for this SDK key. Vercel only returns this at creation time.",
			},
		},
	}
}

func (r *featureFlagSDKKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureFlagSDKKeyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	out, err := r.client.CreateFeatureFlagSDKKey(ctx, client.CreateFeatureFlagSDKKeyRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		TeamID:      plan.TeamID.ValueString(),
		Environment: plan.Environment.ValueString(),
		Type:        plan.Type.ValueString(),
		Label:       plan.Label.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Feature Flag SDK Key",
			"Could not create Feature Flag SDK Key, unexpected error: "+err.Error(),
		)
		return
	}

	result := featureFlagSDKKeyFromClient(out, plan)
	tflog.Info(ctx, "created feature flag sdk key", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"hash_key":   result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagSDKKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureFlagSDKKeyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	keys, err := r.client.ListFeatureFlagSDKKeys(ctx, client.ListFeatureFlagSDKKeysRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag SDK Key",
			fmt.Sprintf(
				"Could not list Feature Flag SDK Keys for %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	for _, key := range keys {
		if key.HashKey != state.ID.ValueString() {
			continue
		}

		result := mergeFeatureFlagSDKKeyState(featureFlagSDKKeyFromClient(key, state), state)

		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *featureFlagSDKKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a Feature Flag SDK Key is not supported",
		"Updating a Feature Flag SDK Key is not supported. Change any configurable field to force recreation.",
	)
}

func (r *featureFlagSDKKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureFlagSDKKeyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	err := r.client.DeleteFeatureFlagSDKKey(ctx, client.DeleteFeatureFlagSDKKeyRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		HashKey:   state.ID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Feature Flag SDK Key",
			fmt.Sprintf(
				"Could not delete Feature Flag SDK Key %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted feature flag sdk key", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
		"hash_key":   state.ID.ValueString(),
	})
}

func (r *featureFlagSDKKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, hashKey, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag SDK Key",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/hash_key\" or \"project_id/hash_key\"", req.ID),
		)
		return
	}

	keys, err := r.client.ListFeatureFlagSDKKeys(ctx, client.ListFeatureFlagSDKKeysRequest{
		ProjectID: projectID,
		TeamID:    teamID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag SDK Key",
			fmt.Sprintf("Could not list Feature Flag SDK Keys for %s %s, unexpected error: %s", teamID, projectID, err),
		)
		return
	}

	for _, key := range keys {
		if key.HashKey != hashKey {
			continue
		}

		result := featureFlagSDKKeyFromClient(key, featureFlagSDKKeyModel{
			ProjectID: types.StringValue(projectID),
			TeamID:    types.StringValue(r.client.TeamID(teamID)),
		})
		diags := resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.State.RemoveResource(ctx)
}

func featureFlagSDKKeyFromClient(out client.FeatureFlagSDKKey, ref featureFlagSDKKeyModel) featureFlagSDKKeyModel {
	model := featureFlagSDKKeyModel{
		ID:               types.StringValue(out.HashKey),
		ProjectID:        types.StringValue(out.ProjectID),
		TeamID:           ref.TeamID,
		Environment:      types.StringValue(out.Environment),
		Type:             types.StringValue(out.Type),
		Label:            types.StringValue(out.Label),
		SDKKey:           types.StringValue(out.KeyValue),
		ConnectionString: types.StringValue(out.ConnectionString),
	}
	if model.TeamID.IsNull() {
		model.TeamID = ref.TeamID
	}
	if model.ProjectID.IsNull() {
		model.ProjectID = ref.ProjectID
	}
	return model
}

func mergeFeatureFlagSDKKeyState(next, prior featureFlagSDKKeyModel) featureFlagSDKKeyModel {
	if next.SDKKey.IsNull() || next.SDKKey.ValueString() == "" {
		next.SDKKey = prior.SDKKey
	}
	if next.ConnectionString.IsNull() || next.ConnectionString.ValueString() == "" {
		next.ConnectionString = prior.ConnectionString
	}
	return next
}
