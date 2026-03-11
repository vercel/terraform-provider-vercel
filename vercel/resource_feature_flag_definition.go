package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &featureFlagDefinitionResource{}
	_ resource.ResourceWithConfigure   = &featureFlagDefinitionResource{}
	_ resource.ResourceWithImportState = &featureFlagDefinitionResource{}
)

func newFeatureFlagDefinitionResource() resource.Resource {
	return &featureFlagDefinitionResource{}
}

type featureFlagDefinitionResource struct {
	client *client.Client
}

type featureFlagDefinitionModel struct {
	ID          types.String              `tfsdk:"id"`
	ProjectID   types.String              `tfsdk:"project_id"`
	TeamID      types.String              `tfsdk:"team_id"`
	Key         types.String              `tfsdk:"key"`
	Description types.String              `tfsdk:"description"`
	Kind        types.String              `tfsdk:"kind"`
	Archived    types.Bool                `tfsdk:"archived"`
	Variant     []featureFlagVariantModel `tfsdk:"variant"`
}

func (r *featureFlagDefinitionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_definition"
}

func (r *featureFlagDefinitionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func featureFlagDefinitionVariantSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Required:    true,
		Description: "The variants available for this flag.",
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Required:    true,
					Description: "The stable variant identifier referenced by flag configuration.",
				},
				"label": schema.StringAttribute{
					Optional:    true,
					Description: "A human-readable label for the variant.",
				},
				"description": schema.StringAttribute{
					Optional:    true,
					Description: "A human-readable description for the variant.",
				},
				"value_string": schema.StringAttribute{
					Optional:    true,
					Description: "The string value for this variant. Use this when `kind = \"string\"`.",
				},
				"value_number": schema.NumberAttribute{
					Optional:    true,
					Description: "The numeric value for this variant. Use this when `kind = \"number\"`.",
				},
				"value_bool": schema.BoolAttribute{
					Optional:    true,
					Description: "The boolean value for this variant. Use this when `kind = \"boolean\"`.",
				},
			},
		},
	}
}

func (r *featureFlagDefinitionResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Feature Flag Definition resource.

This resource owns the stable flag contract: the flag key, kind, description, archival state, and variants.

Use this resource by itself when you want Terraform to register the flag but leave ongoing rollout and targeting to the Vercel dashboard.

Vercel requires environments when a flag is created, so this resource bootstraps all environments in a paused state using the neutral control/off variant until rollout is managed elsewhere.

If Terraform should also manage the simplified per-environment rollout, pair this resource with ` + "`vercel_feature_flag_config`" + `.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the feature flag.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project that owns the flag.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Required:      true,
				Description:   "The stable flag key used in your application code.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						featureFlagKeyRegex,
						"Flag keys may only contain letters, numbers, dashes, and underscores.",
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A human-readable description of the flag.",
			},
			"kind": schema.StringAttribute{
				Required:      true,
				Description:   "The type of value this flag returns. Must be one of `boolean`, `string`, or `number`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("boolean", "string", "number"),
				},
			},
			"archived": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the flag should be archived instead of active.",
			},
			"variant": featureFlagDefinitionVariantSchema(),
		},
	}
}

func (r *featureFlagDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureFlagDefinitionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	createReq, diags := featureFlagDefinitionCreateRequest(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createPayload := createReq
	createPayload.State = ""

	out, err := r.client.CreateFeatureFlag(ctx, createPayload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Feature Flag Definition",
			"Could not create Feature Flag Definition, unexpected error: "+err.Error(),
		)
		return
	}

	if createReq.State == "archived" {
		out, err = r.client.UpdateFeatureFlag(ctx, client.UpdateFeatureFlagRequest{
			ProjectID: plan.ProjectID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			FlagID:    out.ID,
			State:     "archived",
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error archiving Feature Flag Definition after creation",
				"Could not archive Feature Flag Definition after creation, unexpected error: "+err.Error(),
			)
			return
		}
	}

	result, diags := featureFlagDefinitionFromClient(out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created feature flag definition", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"flag_id":    result.ID.ValueString(),
		"key":        result.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureFlagDefinitionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	out, err := r.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		FlagID:    state.ID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag Definition",
			fmt.Sprintf(
				"Could not get Feature Flag Definition %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagDefinitionFromClient(out, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan featureFlagDefinitionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	updateReq, diags := featureFlagDefinitionUpdateRequest(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateFeatureFlag(ctx, updateReq)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Feature Flag Definition",
			fmt.Sprintf(
				"Could not update Feature Flag Definition %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ProjectID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagDefinitionFromClient(out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureFlagDefinitionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	if !state.Archived.IsNull() && !state.Archived.ValueBool() {
		_, err := r.client.UpdateFeatureFlag(ctx, client.UpdateFeatureFlagRequest{
			ProjectID: state.ProjectID.ValueString(),
			TeamID:    state.TeamID.ValueString(),
			FlagID:    state.ID.ValueString(),
			State:     "archived",
		})
		if client.NotFound(err) {
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error archiving Feature Flag Definition before delete",
				fmt.Sprintf(
					"Could not archive Feature Flag Definition %s %s %s before deleting it, unexpected error: %s",
					state.TeamID.ValueString(),
					state.ProjectID.ValueString(),
					state.ID.ValueString(),
					err,
				),
			)
			return
		}
	}

	err := r.client.DeleteFeatureFlag(ctx, client.DeleteFeatureFlagRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		FlagID:    state.ID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Feature Flag Definition",
			fmt.Sprintf(
				"Could not delete Feature Flag Definition %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted feature flag definition", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
		"flag_id":    state.ID.ValueString(),
	})
}

func (r *featureFlagDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, flagID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Definition",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/flag_id\" or \"project_id/flag_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		FlagID:    flagID,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Definition",
			fmt.Sprintf("Could not get Feature Flag Definition %s %s %s, unexpected error: %s", teamID, projectID, flagID, err),
		)
		return
	}

	result, diags := featureFlagDefinitionFromClient(out, featureFlagDefinitionModel{
		ProjectID: types.StringValue(projectID),
		TeamID:    types.StringValue(r.client.TeamID(teamID)),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func featureFlagDefinitionCreateRequest(plan featureFlagDefinitionModel) (client.CreateFeatureFlagRequest, diag.Diagnostics) {
	variants, _, diags := featureFlagVariantsToClient(plan.Kind.ValueString(), plan.Variant)
	if diags.HasError() {
		return client.CreateFeatureFlagRequest{}, diags
	}

	bootstrapVariantID, err := featureFlagBootstrapVariantID(plan.Kind.ValueString(), variants)
	if err != nil {
		diags.AddError("Invalid Feature Flag bootstrap variant", err.Error())
		return client.CreateFeatureFlagRequest{}, diags
	}

	req := client.CreateFeatureFlagRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
		Key:          plan.Key.ValueString(),
		Kind:         plan.Kind.ValueString(),
		Description:  plan.Description.ValueString(),
		State:        "active",
		Variants:     variants,
		Environments: featureFlagBootstrapEnvironments(bootstrapVariantID),
	}
	if !plan.Archived.IsNull() && plan.Archived.ValueBool() {
		req.State = "archived"
	}

	return req, diags
}

func featureFlagDefinitionUpdateRequest(plan featureFlagDefinitionModel) (client.UpdateFeatureFlagRequest, diag.Diagnostics) {
	variants, _, diags := featureFlagVariantsToClient(plan.Kind.ValueString(), plan.Variant)
	return client.UpdateFeatureFlagRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		TeamID:      plan.TeamID.ValueString(),
		FlagID:      plan.ID.ValueString(),
		Description: plan.Description.ValueString(),
		State:       featureFlagDefinitionState(plan.Archived),
		Variants:    variants,
	}, diags
}

func featureFlagDefinitionFromClient(out client.FeatureFlag, ref featureFlagDefinitionModel) (featureFlagDefinitionModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := featureFlagDefinitionModel{
		ID:          types.StringValue(out.ID),
		ProjectID:   types.StringValue(out.ProjectID),
		TeamID:      ref.TeamID,
		Key:         types.StringValue(out.Slug),
		Description: featureFlagOptionalStringValue(out.Description, ref.Description),
		Kind:        types.StringValue(out.Kind),
		Archived:    types.BoolValue(out.State == "archived"),
	}

	priorVariants := map[string]featureFlagVariantModel{}
	for _, variant := range ref.Variant {
		priorVariants[variant.ID.ValueString()] = variant
	}

	model.Variant = make([]featureFlagVariantModel, 0, len(out.Variants))
	for _, variant := range out.Variants {
		mapped, err := featureFlagVariantFromClient(out.Kind, variant, priorVariants[variant.ID])
		if err != nil {
			diags.AddError(
				"Unsupported Feature Flag variant",
				fmt.Sprintf("Feature flag %q has a variant that cannot be represented by this resource: %s", out.Slug, err),
			)
			return model, diags
		}
		model.Variant = append(model.Variant, mapped)
	}

	if model.TeamID.IsNull() {
		model.TeamID = ref.TeamID
	}
	if model.ProjectID.IsNull() {
		model.ProjectID = ref.ProjectID
	}

	return model, diags
}

func featureFlagDefinitionState(archived types.Bool) string {
	if !archived.IsNull() && archived.ValueBool() {
		return "archived"
	}
	return "active"
}
