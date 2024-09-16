package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &projectFunctionMaxDurationDataSource{}
	_ datasource.DataSourceWithConfigure = &projectFunctionMaxDurationDataSource{}
)

func newProjectFunctionMaxDurationDataSource() datasource.DataSource {
	return &projectFunctionMaxDurationDataSource{}
}

type projectFunctionMaxDurationDataSource struct {
	client *client.Client
}

func (d *projectFunctionMaxDurationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_function_max_duration"
}

func (d *projectFunctionMaxDurationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (r *projectFunctionMaxDurationDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides information about a Project's Function max duration setting.

This controls the default maximum duration of your Serverless Functions can use while executing. 10s is recommended for most workloads. Can be configured from 1 to 900 seconds (plan limits apply). You can override this per function using the vercel.json file.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the resource.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project to read the Function max duration setting for.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"max_duration": schema.Int64Attribute{
				Description: "The default max duration for your Serverless Functions. Must be between 1 and 900 (plan limits apply)",
				Computed:    true,
				Validators: []validator.Int64{
					int64GreaterThan(0),
					int64LessThan(900),
				},
			},
		},
	}
}

func (d *projectFunctionMaxDurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectFunctionMaxDuration
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetProjectFunctionMaxDuration(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project Function max duration",
			fmt.Sprintf("Could not get Project Function max duration %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectFunctionMaxDuration(out)
	tflog.Info(ctx, "read project function max duration", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
