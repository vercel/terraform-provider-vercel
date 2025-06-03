package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &projectRollingReleaseDataSource{}
	_ datasource.DataSourceWithConfigure = &projectRollingReleaseDataSource{}
)

func newProjectRollingReleaseDataSource() datasource.DataSource {
	return &projectRollingReleaseDataSource{}
}

type projectRollingReleaseDataSource struct {
	client *client.Client
}

func (r *projectRollingReleaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_rolling_release"
}

func (r *projectRollingReleaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Schema returns the schema information for a project Rolling Release datasource.
func (r *projectRollingReleaseDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Rolling Release datasource.

A Project Rolling Release datasource details information about a Rolling Release on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/rolling-releases).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project for the rolling release",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team.",
			},
		},
	}
}

type ProjectRollingReleaseWithID struct {
	Enabled              types.Bool   `tfsdk:"enabled"`
	AdvancementType      types.String `tfsdk:"advancement_type"`
	CanaryResponseHeader types.Bool   `tfsdk:"canary_response_header"`
	Stages               types.List   `tfsdk:"stages"`

	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	ID        types.String `tfsdk:"id"`
}

// Read will read the rolling release configuration of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectRollingReleaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectRollingReleaseWithID
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetRollingRelease(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("Could not get project rolling release %s %s, unexpected error: %s",
				config.ProjectID.ValueString(),
				config.TeamID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectRollingRelease(out, config.ProjectID)
	tflog.Info(ctx, "read project rolling release", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	var stages []attr.Value
	for _, s := range result.Stages {
		stages = append(stages, types.Float64Value(s.TargetPercentage))
	}

	diags = resp.State.Set(ctx, ProjectRollingReleaseWithID{
		Enabled:              types.BoolValue(result.Enabled),
		AdvancementType:      types.StringValue(result.AdvancementType),
		CanaryResponseHeader: types.BoolValue(result.CanaryResponseHeader),
		Stages:               types.ListValueMust(types.StringType, stages),
		ProjectID:            result.ProjectID,
		TeamID:               result.TeamID,
		ID:                   result.ProjectID,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
