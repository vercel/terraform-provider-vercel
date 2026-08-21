package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithConfigure   = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

func newAPIKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type apiKeyResource struct {
	client *client.Client
}

func (r *apiKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiKeyResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an API Key resource.

An API Key can be used to authenticate with the Vercel AI Gateway.

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
			"purpose": schema.StringAttribute{
				Description:   "The purpose of the API key. Currently, only `ai-gateway` is supported.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("ai-gateway"),
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
			"created_at": schema.Int64Attribute{
				Description:   "The Unix timestamp in milliseconds when the API key was created.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

type APIKey struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Purpose      types.String `tfsdk:"purpose"`
	TeamID       types.String `tfsdk:"team_id"`
	APIKeyString types.String `tfsdk:"api_key_string"`
	PartialKey   types.String `tfsdk:"partial_key"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
}

func responseToAPIKey(out client.APIKey, apiKeyString types.String) APIKey {
	if out.APIKeyString != nil {
		apiKeyString = types.StringPointerValue(out.APIKeyString)
	}

	return APIKey{
		ID:           types.StringValue(out.ID),
		Name:         types.StringValue(out.Name),
		Purpose:      types.StringValue(out.Purpose),
		TeamID:       types.StringValue(out.TeamID),
		APIKeyString: apiKeyString,
		PartialKey:   types.StringValue(out.PartialKey),
		CreatedAt:    types.Int64Value(out.CreatedAt),
	}
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APIKey
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(plan.Name.ValueString())
	if name == "" || name != plan.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Invalid API key name",
			"API key name cannot be empty, whitespace, or have leading or trailing whitespace.",
		)
		return
	}
	if r.client.TeamID(plan.TeamID.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("team_id"),
			"API key requires team scope",
			"Set `team_id` or configure a default team on the provider.",
		)
		return
	}

	out, err := r.client.CreateAPIKey(ctx, client.CreateAPIKeyRequest{
		Name:    name,
		Purpose: plan.Purpose.ValueString(),
		TeamID:  plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating API Key",
			"Could not create API Key, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToAPIKey(out, types.StringNull())
	tflog.Info(ctx, "created API key", map[string]any{
		"key_id":  result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APIKey
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAPIKey(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading API Key",
			fmt.Sprintf("Could not get API Key %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	result := responseToAPIKey(out, state.APIKeyString)
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating an API Key is not supported",
		"Updating an API Key is not supported",
	)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APIKey
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAPIKey(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting API Key",
			fmt.Sprintf("Could not delete API Key %s, unexpected error: %s", state.ID.ValueString(), err),
		)
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, keyID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing API Key",
			fmt.Sprintf("Invalid id '%s' specified. Should be in format \"team_id/key_id\" or \"key_id\".", req.ID),
		)
		return
	}

	out, err := r.client.GetAPIKey(ctx, keyID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing API Key",
			fmt.Sprintf("Could not get API Key %s, unexpected error: %s", keyID, err),
		)
		return
	}

	diags := resp.State.Set(ctx, responseToAPIKey(out, types.StringNull()))
	resp.Diagnostics.Append(diags...)
}
