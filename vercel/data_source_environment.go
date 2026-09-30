package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &environmentDataSource{}
	_ datasource.DataSourceWithConfigure = &environmentDataSource{}
)

func newEnvironmentDataSource() datasource.DataSource {
	return &environmentDataSource{}
}

type environmentDataSource struct {
	client *client.Client
}

func (d *environmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *environmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *environmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about a Vercel environment.

This data source reads both built-in environments (` + "`production`" + `, ` + "`preview`" + `, ` + "`development`" + `) and user-managed Custom Environments (` + "`env_*`" + `) through one schema. Resolve an environment by project plus ID or slug.

Built-in environments are managed by Vercel and cannot be imported into or managed by ` + "`vercel_custom_environment`" + `. Use ` + "`vercel_custom_environment`" + ` only for user-managed ` + "`env_*`" + ` environments.
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team ID to use when reading the environment. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the existing Vercel Project.",
			},
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The canonical environment ID. Built-in environments use `production`, `preview`, or `development`. Custom Environments use an `env_*` ID. Exactly one of `id` or `slug` must be set.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("id"),
						path.MatchRoot("slug"),
					),
				},
			},
			"slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The environment slug. Built-in environments use `production`, `preview`, or `development`. Custom Environments use their configured name. Exactly one of `id` or `slug` must be set.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("id"),
						path.MatchRoot("slug"),
					),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name of the environment.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The environment type. One of `production`, `preview`, or `development`. Custom Environments are `preview`.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A description of the environment.",
			},
			"kind": schema.StringAttribute{
				Computed:    true,
				Description: "Whether the environment is a built-in `system` environment or a user-managed `custom` environment.",
			},
			"managed_by": schema.StringAttribute{
				Computed:    true,
				Description: "Who manages the environment. `vercel` for built-in environments and `user` for Custom Environments.",
			},
			"branch_tracking": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The branch routing configuration for the environment. When set, qualifying git branches deploy into this environment.",
				Attributes: map[string]schema.Attribute{
					"pattern": schema.StringAttribute{
						Computed:    true,
						Description: "The pattern of the branch name to track.",
					},
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "How a branch name should be matched against the pattern. One of `startsWith`, `endsWith`, or `equals`.",
					},
				},
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "The timestamp the environment was created, in milliseconds since the Unix epoch. Null for built-in environments.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "The timestamp the environment was last updated, in milliseconds since the Unix epoch. Null for built-in environments.",
			},
			"capabilities": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "What this environment can do. Built-in Preview and Production support domains and deployments but cannot be managed as Custom Environments. Development does not support domains or deployments.",
				Attributes: map[string]schema.Attribute{
					"manage": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether the environment can be created, updated, or deleted through `vercel_custom_environment`. Always false for built-in environments.",
					},
					"domains": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether domains can be assigned to this environment.",
					},
					"deployments": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether deployments can target this environment.",
					},
				},
			},
		},
	}
}

type environmentDataSourceModel struct {
	TeamID         types.String `tfsdk:"team_id"`
	ProjectID      types.String `tfsdk:"project_id"`
	ID             types.String `tfsdk:"id"`
	Slug           types.String `tfsdk:"slug"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Description    types.String `tfsdk:"description"`
	Kind           types.String `tfsdk:"kind"`
	ManagedBy      types.String `tfsdk:"managed_by"`
	BranchTracking types.Object `tfsdk:"branch_tracking"`
	CreatedAt      types.Int64  `tfsdk:"created_at"`
	UpdatedAt      types.Int64  `tfsdk:"updated_at"`
	Capabilities   types.Object `tfsdk:"capabilities"`
}

var environmentCapabilitiesAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"manage":      types.BoolType,
		"domains":     types.BoolType,
		"deployments": types.BoolType,
	},
}

func (d *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config environmentDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idOrSlug := config.ID.ValueString()
	if idOrSlug == "" {
		idOrSlug = config.Slug.ValueString()
	}

	res, err := d.client.GetEnvironment(ctx, client.GetEnvironmentRequest{
		TeamID:    config.TeamID.ValueString(),
		ProjectID: config.ProjectID.ValueString(),
		IDOrSlug:  idOrSlug,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading environment",
			fmt.Sprintf("Could not read environment %s in project %s, unexpected error: %s",
				idOrSlug,
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertEnvironmentToDataSourceModel(res)
	tflog.Trace(ctx, "read environment", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"project_id":     result.ProjectID.ValueString(),
		"environment_id": result.ID.ValueString(),
		"kind":           result.Kind.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func convertEnvironmentToDataSourceModel(res client.Environment) environmentDataSourceModel {
	bt := types.ObjectNull(branchTrackingAttrType.AttrTypes)
	if res.BranchMatcher != nil {
		bt = types.ObjectValueMust(
			branchTrackingAttrType.AttrTypes, map[string]attr.Value{
				"pattern": types.StringValue(res.BranchMatcher.Pattern),
				"type":    types.StringValue(res.BranchMatcher.Type),
			},
		)
	}

	createdAt := types.Int64Null()
	if res.CreatedAt != nil {
		createdAt = types.Int64Value(*res.CreatedAt)
	}
	updatedAt := types.Int64Null()
	if res.UpdatedAt != nil {
		updatedAt = types.Int64Value(*res.UpdatedAt)
	}

	return environmentDataSourceModel{
		TeamID:         types.StringValue(res.TeamID),
		ProjectID:      types.StringValue(res.ProjectID),
		ID:             types.StringValue(res.ID),
		Slug:           types.StringValue(res.Slug),
		Name:           types.StringValue(res.Name),
		Type:           types.StringValue(res.Type),
		Description:    types.StringValue(res.Description),
		Kind:           types.StringValue(res.Lifecycle),
		ManagedBy:      types.StringValue(res.ManagedBy),
		BranchTracking: bt,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		Capabilities: types.ObjectValueMust(
			environmentCapabilitiesAttrType.AttrTypes, map[string]attr.Value{
				"manage":      types.BoolValue(res.Capabilities.Manage),
				"domains":     types.BoolValue(res.Capabilities.Domains),
				"deployments": types.BoolValue(res.Capabilities.Deployments),
			},
		),
	}
}
