package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

var (
	_ resource.Resource                = &customEnvironmentResource{}
	_ resource.ResourceWithConfigure   = &customEnvironmentResource{}
	_ resource.ResourceWithImportState = &customEnvironmentResource{}
)

func newCustomEnvironmentResource() resource.Resource {
	return &customEnvironmentResource{}
}

type customEnvironmentResource struct {
	client *client.Client
}

func (r *customEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_environment"
}

func (r *customEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Environments help manage the deployment lifecycle on the Vercel platform.

By default, all teams use three environments when developing their project: Production, Preview, and Development. However, teams can also create custom environments to suit their needs. To learn more about the limits for each plan, see limits.

Custom environments allow you to configure customized, pre-production environments for your project, such as staging or QA, with branch rules that will automatically deploy your branch when the branch name matches the rule. With custom environments you can also attach a domain to your environment, set environment variables, or import environment variables from another environment.

Custom environments are designed as pre-production environments intended for long-running use. This contrasts with regular preview environments, which are designed for creating ephemeral, short-lived deployments.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the environment.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
				Description:   "The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the existing Vercel Project.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9\-]{0,32}$`),
						"The name of a Custom Environment can only contain up to 32 alphanumeric lowercase characters and hyphens",
					),
				},
			},
			"description": schema.StringAttribute{
				Description:   "A description of what the environment is.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"branch_tracking": schema.SingleNestedAttribute{
				Description:   "The branch tracking configuration for the environment. When enabled, each qualifying merge will generate a deployment.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"pattern": schema.StringAttribute{
						Description: "The pattern of the branch name to track.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 100),
						},
					},
					"type": schema.StringAttribute{
						Description: "How a branch name should be matched against the pattern. Must be one of 'startsWith', 'endsWith' or 'equals'.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("startsWith", "endsWith", "equals"),
						},
					},
				},
			},
		},
	}
}

type BranchTracking struct {
	Pattern types.String `tfsdk:"pattern"`
	Type    types.String `tfsdk:"type"`
}

type CustomEnvironment struct {
	ID             types.String `tfsdk:"id"`
	TeamID         types.String `tfsdk:"team_id"`
	ProjectID      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	BranchTracking types.Object `tfsdk:"branch_tracking"`
}

func (c CustomEnvironment) toCreateRequest(ctx context.Context) (client.CreateCustomEnvironmentRequest, diag.Diagnostics) {
	var bm *client.BranchMatcher
	if !c.BranchTracking.IsNull() && !c.BranchTracking.IsUnknown() {
		bt, diags := c.branchTracking(ctx)
		if diags.HasError() {
			return client.CreateCustomEnvironmentRequest{}, diags
		}
		bm = &client.BranchMatcher{
			Pattern: bt.Pattern.ValueString(),
			Type:    bt.Type.ValueString(),
		}
	}
	return client.CreateCustomEnvironmentRequest{
		TeamID:        c.TeamID.ValueString(),
		ProjectID:     c.ProjectID.ValueString(),
		Slug:          c.Name.ValueString(),
		Description:   c.Description.ValueString(),
		BranchMatcher: bm,
	}, nil
}

func (c CustomEnvironment) toUpdateRequest(ctx context.Context) (client.UpdateCustomEnvironmentRequest, diag.Diagnostics) {
	var bm *client.BranchMatcher
	if !c.BranchTracking.IsNull() && !c.BranchTracking.IsUnknown() {
		bt, diags := c.branchTracking(ctx)
		if diags.HasError() {
			return client.UpdateCustomEnvironmentRequest{}, diags
		}
		bm = &client.BranchMatcher{
			Pattern: bt.Pattern.ValueString(),
			Type:    bt.Type.ValueString(),
		}
	}
	return client.UpdateCustomEnvironmentRequest{
		TeamID:        c.TeamID.ValueString(),
		ProjectID:     c.ProjectID.ValueString(),
		Slug:          c.Name.ValueString(),
		Description:   c.Description.ValueString(),
		BranchMatcher: bm,
	}, nil
}

func (c CustomEnvironment) branchTracking(ctx context.Context) (BranchTracking, diag.Diagnostics) {
	var bt BranchTracking
	diags := c.BranchTracking.As(ctx, &bt, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    false,
		UnhandledUnknownAsEmpty: false,
	})
	return bt, diags
}

var branchTrackingAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"pattern": types.StringType,
		"type":    types.StringType,
	},
}

func convertResponseToModel(res client.CustomEnvironmentResponse) CustomEnvironment {
	bt := types.ObjectNull(branchTrackingAttrType.AttrTypes)
	if res.BranchMatcher != nil {
		bt = types.ObjectValueMust(
			branchTrackingAttrType.AttrTypes, map[string]attr.Value{
				"pattern": types.StringValue(res.BranchMatcher.Pattern),
				"type":    types.StringValue(res.BranchMatcher.Type),
			})
	}
	return CustomEnvironment{
		ID:             types.StringValue(res.ID),
		TeamID:         types.StringValue(res.TeamID),
		ProjectID:      types.StringValue(res.ProjectID),
		Name:           types.StringValue(res.Slug),
		Description:    types.StringValue(res.Description),
		BranchTracking: bt,
	}
}

func (r *customEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomEnvironment
	diags := req.Plan.Get(ctx, &plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	createRequest, diags := plan.toCreateRequest(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	res, err := r.client.CreateCustomEnvironment(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom environment",
			fmt.Sprintf("Could not create custom environment, unexpected error: %s", err),
		)
		return
	}

	tflog.Info(ctx, "created custom environment", map[string]interface{}{
		"team_id":               plan.TeamID.ValueString(),
		"project_id":            plan.ProjectID.ValueString(),
		"custom_environment_id": res.ID,
		"total_res":             res,
	})

	diags = resp.State.Set(ctx, convertResponseToModel(res))
	resp.Diagnostics.Append(diags...)
}

func (r *customEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomEnvironment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetCustomEnvironment(ctx, client.GetCustomEnvironmentRequest{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		Slug:      state.Name.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom environment",
			fmt.Sprintf("Could not read custom environment, unexpected error: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "read custom environment", map[string]interface{}{
		"team_id":               state.TeamID.ValueString(),
		"project_id":            state.ProjectID.ValueString(),
		"custom_environment_id": res.ID,
	})

	diags = resp.State.Set(ctx, convertResponseToModel(res))
	resp.Diagnostics.Append(diags...)
}

func (r *customEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomEnvironment
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest, diags := plan.toUpdateRequest(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	res, err := r.client.UpdateCustomEnvironment(ctx, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom environment",
			fmt.Sprintf("Could not update custom environment, unexpected error: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created custom environment", map[string]interface{}{
		"team_id":               plan.TeamID.ValueString(),
		"project_id":            plan.ProjectID.ValueString(),
		"custom_environment_id": res.ID,
	})

	diags = resp.State.Set(ctx, convertResponseToModel(res))
	resp.Diagnostics.Append(diags...)
}

func (r *customEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomEnvironment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCustomEnvironment(ctx, client.DeleteCustomEnvironmentRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		Slug:      state.Name.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing custom environment",
			fmt.Sprintf("Could not remove custom environment: %s", err),
		)
		return
	}
}

// ImportState implements resource.ResourceWithImportState.
func (r *customEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, name, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Custom Environment",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/custom_environment_name\" or \"project_id/custom_environment_name\"", req.ID),
		)
	}
	res, err := r.client.GetCustomEnvironment(ctx, client.GetCustomEnvironmentRequest{
		TeamID:    teamID,
		ProjectID: projectID,
		Slug:      name,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom environment",
			fmt.Sprintf("Could not read custom environment, unexpected error: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "import custom environment", map[string]interface{}{
		"team_id":               teamID,
		"project_id":            projectID,
		"custom_environment_id": res.ID,
	})

	diags := resp.State.Set(ctx, convertResponseToModel(res))
	resp.Diagnostics.Append(diags...)
}
