package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The unique identifier of the API key.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:   "The human-readable name of the API key.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	TeamID       types.String `tfsdk:"team_id"`
	APIKeyString types.String `tfsdk:"api_key_string"`
	PartialKey   types.String `tfsdk:"partial_key"`
}

func responseToAIGatewayAPIKey(out client.AIGatewayAPIKey, apiKeyString types.String) AIGatewayAPIKey {
	if out.APIKeyString != nil {
		apiKeyString = types.StringPointerValue(out.APIKeyString)
	}

	return AIGatewayAPIKey{
		ID:           types.StringValue(out.ID),
		Name:         types.StringValue(out.Name),
		TeamID:       types.StringValue(out.TeamID),
		APIKeyString: apiKeyString,
		PartialKey:   types.StringValue(out.PartialKey),
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

	out, err := r.client.CreateAIGatewayAPIKey(ctx, client.CreateAIGatewayAPIKeyRequest{
		Name:    name,
		Purpose: "ai-gateway",
		TeamID:  plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AI Gateway API Key",
			"Could not create AI Gateway API Key, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToAIGatewayAPIKey(out, types.StringNull())
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

	result := responseToAIGatewayAPIKey(out, state.APIKeyString)
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *aiGatewayAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating an AI Gateway API Key is not supported",
		"Updating an AI Gateway API Key is not supported",
	)
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

	diags := resp.State.Set(ctx, responseToAIGatewayAPIKey(out, types.StringNull()))
	resp.Diagnostics.Append(diags...)
}
