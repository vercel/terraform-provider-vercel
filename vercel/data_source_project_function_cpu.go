package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &projectFunctionCPUDataSource{}
	_ datasource.DataSourceWithConfigure = &projectFunctionCPUDataSource{}
)

func newProjectFunctionCPUDataSource() datasource.DataSource {
	return &projectFunctionCPUDataSource{}
}

type projectFunctionCPUDataSource struct {
	client *client.Client
}

func (d *projectFunctionCPUDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_function_cpu"
}

func (d *projectFunctionCPUDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *projectFunctionCPUDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		DeprecationMessage: "This data source is deprecated and no longer works. Please use the `vercel_project` data source and its `resource_config` attribute instead.",
		Description: `Provides information about a Project's Function CPU setting.

This controls the maximum amount of CPU utilization your Serverless Functions can use while executing. Standard is optimal for most frontend workloads. You can override this per function using the vercel.json file.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the resource.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project to read the Function CPU setting for.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"cpu": schema.StringAttribute{
				Description: "The amount of CPU available to your Serverless Functions. Should be one of 'basic' (0.6vCPU), 'standard' (1vCPU) or 'performance' (1.7vCPUs).",
				Computed:    true,
			},
		},
	}
}

func (d *projectFunctionCPUDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("`vercel_project_function_cpu` data source deprecated", "use `vercel_project` data source and its `resource_config` attribute instead"),
	)
	var config ProjectFunctionCPU
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetProjectFunctionCPU(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project Function CPU",
			fmt.Sprintf("Could not get Project Function CPU %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectFunctionCPU(out)
	tflog.Info(ctx, "read project function cpu", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
