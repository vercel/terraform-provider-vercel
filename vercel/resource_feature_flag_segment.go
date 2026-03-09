package vercel

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &featureFlagSegmentResource{}
	_ resource.ResourceWithConfigure   = &featureFlagSegmentResource{}
	_ resource.ResourceWithImportState = &featureFlagSegmentResource{}
)

var featureFlagSegmentSlugRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,512}$`)

func newFeatureFlagSegmentResource() resource.Resource {
	return &featureFlagSegmentResource{}
}

type featureFlagSegmentResource struct {
	client *client.Client
}

type featureFlagSegmentMatchModel struct {
	Entity    types.String `tfsdk:"entity"`
	Attribute types.String `tfsdk:"attribute"`
	Values    types.Set    `tfsdk:"values"`
}

type featureFlagSegmentModel struct {
	ID          types.String                   `tfsdk:"id"`
	ProjectID   types.String                   `tfsdk:"project_id"`
	TeamID      types.String                   `tfsdk:"team_id"`
	Slug        types.String                   `tfsdk:"slug"`
	Name        types.String                   `tfsdk:"name"`
	Description types.String                   `tfsdk:"description"`
	Hint        types.String                   `tfsdk:"hint"`
	Include     []featureFlagSegmentMatchModel `tfsdk:"include"`
	Exclude     []featureFlagSegmentMatchModel `tfsdk:"exclude"`
}

func (r *featureFlagSegmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_segment"
}

func (r *featureFlagSegmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func featureFlagSegmentMatchSchema(description string) schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		Optional:    true,
		Description: description,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"entity": schema.StringAttribute{
					Required:    true,
					Description: "The entity type to match, for example `user`.",
				},
				"attribute": schema.StringAttribute{
					Required:    true,
					Description: "The entity attribute to match, for example `email`.",
				},
				"values": schema.SetAttribute{
					Required:    true,
					ElementType: types.StringType,
					Description: "The exact values to include or exclude for this entity attribute.",
					Validators: []validator.Set{
						setvalidator.SizeAtLeast(1),
					},
				},
			},
		},
	}
}

func (r *featureFlagSegmentResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Feature Flag Segment resource.

This first draft focuses on exact-match membership lists through ` + "`include`" + ` and ` + "`exclude`" + ` entries. Dashboard-defined rule logic is intentionally not flattened into Terraform here.

Vercel's API requires a ` + "`hint`" + ` field even for simple segments, so this resource defaults it to an empty string unless you want to populate it for dashboard users.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the segment.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project that owns the segment.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				Required:      true,
				Description:   "The stable segment slug used by the Vercel Flags API.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						featureFlagSegmentSlugRegex,
						"Segment slugs may only contain letters, numbers, dashes, and underscores.",
					),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The human-readable segment name shown in the Vercel dashboard.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A human-readable description of the segment.",
			},
			"hint": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "An optional dashboard hint for the segment.",
			},
			"include": featureFlagSegmentMatchSchema("Exact entity attribute values that should always be part of this segment."),
			"exclude": featureFlagSegmentMatchSchema("Exact entity attribute values that should always be excluded from this segment."),
		},
	}
}

func (r *featureFlagSegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureFlagSegmentModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	createReq, diags := featureFlagSegmentCreateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateFeatureFlagSegment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Feature Flag Segment",
			"Could not create Feature Flag Segment, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := featureFlagSegmentFromClient(ctx, out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created feature flag segment", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"segment_id": result.ID.ValueString(),
		"slug":       result.Slug.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagSegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureFlagSegmentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	out, err := r.client.GetFeatureFlagSegment(ctx, client.GetFeatureFlagSegmentRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		SegmentID: state.ID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag Segment",
			fmt.Sprintf(
				"Could not get Feature Flag Segment %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagSegmentFromClient(ctx, out, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagSegmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan featureFlagSegmentModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	updateReq, diags := featureFlagSegmentUpdateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateFeatureFlagSegment(ctx, updateReq)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Feature Flag Segment",
			fmt.Sprintf(
				"Could not update Feature Flag Segment %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ProjectID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagSegmentFromClient(ctx, out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagSegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureFlagSegmentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	err := r.client.DeleteFeatureFlagSegment(ctx, client.DeleteFeatureFlagSegmentRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		SegmentID: state.ID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Feature Flag Segment",
			fmt.Sprintf(
				"Could not delete Feature Flag Segment %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted feature flag segment", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
		"segment_id": state.ID.ValueString(),
	})
}

func (r *featureFlagSegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, segmentID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Segment",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/segment_id\" or \"project_id/segment_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetFeatureFlagSegment(ctx, client.GetFeatureFlagSegmentRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		SegmentID: segmentID,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Segment",
			fmt.Sprintf("Could not get Feature Flag Segment %s %s %s, unexpected error: %s", teamID, projectID, segmentID, err),
		)
		return
	}

	result, diags := featureFlagSegmentFromClient(ctx, out, featureFlagSegmentModel{
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

func featureFlagSegmentCreateRequest(ctx context.Context, plan featureFlagSegmentModel) (client.CreateFeatureFlagSegmentRequest, diag.Diagnostics) {
	req, diags := featureFlagSegmentUpsertRequest(ctx, plan)
	req.ProjectID = plan.ProjectID.ValueString()
	req.TeamID = plan.TeamID.ValueString()
	req.Slug = plan.Slug.ValueString()
	return req, diags
}

func featureFlagSegmentUpdateRequest(ctx context.Context, plan featureFlagSegmentModel) (client.UpdateFeatureFlagSegmentRequest, diag.Diagnostics) {
	req, diags := featureFlagSegmentUpsertRequest(ctx, plan)
	return client.UpdateFeatureFlagSegmentRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		TeamID:      plan.TeamID.ValueString(),
		SegmentID:   plan.ID.ValueString(),
		Label:       req.Label,
		Description: req.Description,
		Hint:        req.Hint,
		Data:        req.Data,
	}, diags
}

func featureFlagSegmentUpsertRequest(ctx context.Context, plan featureFlagSegmentModel) (client.CreateFeatureFlagSegmentRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	include, d := featureFlagSegmentMatchListToClient(ctx, plan.Include)
	diags.Append(d...)
	exclude, d := featureFlagSegmentMatchListToClient(ctx, plan.Exclude)
	diags.Append(d...)
	if diags.HasError() {
		return client.CreateFeatureFlagSegmentRequest{}, diags
	}
	if len(include) == 0 && len(exclude) == 0 {
		diags.AddError(
			"Invalid Feature Flag Segment",
			"At least one of include or exclude must contain a value.",
		)
		return client.CreateFeatureFlagSegmentRequest{}, diags
	}

	return client.CreateFeatureFlagSegmentRequest{
		Label:       plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Hint:        plan.Hint.ValueString(),
		Data: client.FeatureFlagSegmentData{
			Rules:   []client.FeatureFlagSegmentRule{},
			Include: include,
			Exclude: exclude,
		},
	}, diags
}

func featureFlagSegmentMatchListToClient(ctx context.Context, list []featureFlagSegmentMatchModel) (map[string]map[string][]client.FeatureFlagSegmentValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := map[string]map[string][]client.FeatureFlagSegmentValue{}

	for _, entry := range list {
		var values []string
		d := entry.Values.ElementsAs(ctx, &values, false)
		diags.Append(d...)
		if diags.HasError() {
			return out, diags
		}

		if _, ok := out[entry.Entity.ValueString()]; !ok {
			out[entry.Entity.ValueString()] = map[string][]client.FeatureFlagSegmentValue{}
		}
		if _, ok := out[entry.Entity.ValueString()][entry.Attribute.ValueString()]; ok {
			diags.AddError(
				"Duplicate Feature Flag Segment match target",
				fmt.Sprintf(
					"%s.%s is declared more than once. Combine the values into a single include or exclude block instead.",
					entry.Entity.ValueString(),
					entry.Attribute.ValueString(),
				),
			)
			return out, diags
		}

		clientValues := make([]client.FeatureFlagSegmentValue, 0, len(values))
		for _, value := range values {
			clientValues = append(clientValues, client.FeatureFlagSegmentValue{Value: value})
		}
		out[entry.Entity.ValueString()][entry.Attribute.ValueString()] = clientValues
	}

	return out, diags
}

func featureFlagSegmentFromClient(ctx context.Context, out client.FeatureFlagSegment, ref featureFlagSegmentModel) (featureFlagSegmentModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(out.Data.Rules) > 0 {
		diags.AddError(
			"Unsupported Feature Flag Segment configuration",
			fmt.Sprintf("Feature flag segment %q contains rule logic, which this first draft resource does not model yet.", out.Slug),
		)
		return featureFlagSegmentModel{}, diags
	}

	model := featureFlagSegmentModel{
		ID:          types.StringValue(out.ID),
		ProjectID:   types.StringValue(out.ProjectID),
		TeamID:      ref.TeamID,
		Slug:        types.StringValue(out.Slug),
		Name:        types.StringValue(out.Label),
		Description: featureFlagSegmentOptionalStringValue(out.Description, ref.Description),
		Hint:        featureFlagSegmentOptionalStringValue(out.Hint, ref.Hint),
	}

	include := ref.Include
	if len(out.Data.Include) > 0 || len(ref.Include) > 0 {
		var d diag.Diagnostics
		include, d = featureFlagSegmentMatchListFromClient(ctx, out.Data.Include)
		diags.Append(d...)
	}
	exclude := ref.Exclude
	if len(out.Data.Exclude) > 0 || len(ref.Exclude) > 0 {
		var d diag.Diagnostics
		exclude, d = featureFlagSegmentMatchListFromClient(ctx, out.Data.Exclude)
		diags.Append(d...)
	}
	if diags.HasError() {
		return model, diags
	}
	model.Include = include
	model.Exclude = exclude
	if model.TeamID.IsNull() {
		model.TeamID = ref.TeamID
	}

	return model, diags
}

func featureFlagSegmentMatchListFromClient(ctx context.Context, source map[string]map[string][]client.FeatureFlagSegmentValue) ([]featureFlagSegmentMatchModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	type matchRow struct {
		entity    string
		attribute string
		values    []string
	}

	rows := make([]matchRow, 0)
	for entity, attributes := range source {
		for attribute, values := range attributes {
			row := matchRow{entity: entity, attribute: attribute, values: make([]string, 0, len(values))}
			for _, value := range values {
				if value.Note != "" {
					diags.AddError(
						"Unsupported Feature Flag Segment value metadata",
						fmt.Sprintf("The value %q for %s.%s includes a note, which this first draft resource does not model yet.", value.Value, entity, attribute),
					)
					return nil, diags
				}
				row.values = append(row.values, value.Value)
			}
			sort.Strings(row.values)
			rows = append(rows, row)
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		left := rows[i].entity + "." + rows[i].attribute
		right := rows[j].entity + "." + rows[j].attribute
		return left < right
	})

	result := make([]featureFlagSegmentMatchModel, 0, len(rows))
	for _, row := range rows {
		values, d := types.SetValueFrom(ctx, types.StringType, row.values)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		result = append(result, featureFlagSegmentMatchModel{
			Entity:    types.StringValue(row.entity),
			Attribute: types.StringValue(row.attribute),
			Values:    values,
		})
	}

	return result, diags
}

func featureFlagSegmentOptionalStringValue(value string, prior types.String) types.String {
	if value == "" {
		if !prior.IsNull() {
			return prior
		}
		return types.StringNull()
	}
	return types.StringValue(value)
}
