package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithValidateConfig = &sharedEnvironmentVariableDataSource{}
)

func newSharedEnvironmentVariableDataSource() datasource.DataSource {
	return &sharedEnvironmentVariableDataSource{}
}

type sharedEnvironmentVariableDataSource struct {
	client *client.Client
}

func (d *sharedEnvironmentVariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_environment_variable"
}

func (d *sharedEnvironmentVariableDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config SharedEnvironmentVariable
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ID.IsNull() && config.Key.IsNull() {
		resp.Diagnostics.AddError(
			"Shared Environment Variable invalid",
			"Shared Environment Variable must have either a key and target, or an ID",
		)
		return
	}

	if !config.ID.IsNull() && (!config.Key.IsNull() || len(config.Target.Elements()) > 0) {
		resp.Diagnostics.AddError(
			"Shared Environment Variable invalid",
			"Shared Environment Variable can only specify either an ID or a key and target, not both",
		)
		return
	}

	if !config.Key.IsNull() && !config.Target.IsUnknown() && len(config.Target.Elements()) == 0 {
		resp.Diagnostics.AddError(
			"Shared Environment Variable invalid",
			"Shared Environment Variable must specify at least one `target` when specifying a key",
		)
		return
	}
}

func (d *sharedEnvironmentVariableDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for a shared environment variable data source
func (d *sharedEnvironmentVariableDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Shared Environment Variable within Vercel.

A Shared Environment Variable resource defines an Environment Variable that can be shared between multiple Vercel Projects.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables/shared-environment-variables).
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team. Shared environment variables require a team.",
			},
			"id": schema.StringAttribute{
				Description: "The ID of the Environment Variable.",
				Optional:    true,
				Computed:    true,
			},
			"target": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf("production", "preview", "development"),
					),
					setvalidator.SizeAtLeast(1),
				},
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the Environment Variable.",
			},
			"value": schema.StringAttribute{
				Computed:    true,
				Description: "The value of the Environment Variable.",
				Sensitive:   true,
			},
			"project_ids": schema.SetAttribute{
				Computed:    true,
				Description: "The ID of the Vercel project.",
				ElementType: types.StringType,
			},
			"sensitive": schema.BoolAttribute{
				Description: "Whether the Environment Variable is sensitive or not.",
				Computed:    true,
			},
			"comment": schema.StringAttribute{
				Description: "A comment explaining what the environment variable is for.",
				Computed:    true,
			},
		},
	}
}

func isSameTarget(a []string, b []types.String) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range b {
		if !contains(a, v.ValueString()) {
			return false
		}
	}
	return true
}

// Read will read project information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *sharedEnvironmentVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SharedEnvironmentVariable
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.ID.ValueString()
	if id == "" {
		// Then we need to look up the shared env var by key + target. Bleugh.
		envs, err := d.client.ListSharedEnvironmentVariables(ctx, config.TeamID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error finding shared environment variable",
				fmt.Sprintf("Could not list shared environment variables for team %s, unexpected error: %s",
					config.TeamID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Info(ctx, "list shared environment variable", map[string]interface{}{
			"team_id": config.TeamID.ValueString(),
		})
		var configTarget []types.String
		diags := config.Target.ElementsAs(ctx, &configTarget, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, e := range envs {
			if e.Key == config.Key.ValueString() && isSameTarget(e.Target, configTarget) {
				// We have found the right env var by Key + Target(s).
				id = e.ID
				break
			}
		}

		if id == "" {
			// the env var was not found - output an error
			targetStrs := []string{}
			for _, t := range configTarget {
				targetStrs = append(targetStrs, t.ValueString())
			}
			resp.Diagnostics.AddError(
				"Error reading shared environment variable",
				fmt.Sprintf("Could not read shared environment variable %s, unexpected error: %s",
					config.Key.ValueString(),
					fmt.Errorf(
						"the shared environment variable with key %s and target %s was not found. Please ensure the full `targets` are specified and that it exists",
						config.Key.ValueString(),
						strings.Join(targetStrs, ","),
					),
				),
			)
			return
		}
	}

	// else we can get by ID.
	out, err := d.client.GetSharedEnvironmentVariable(ctx, config.TeamID.ValueString(), id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not read shared environment variable %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, types.StringNull())
	tflog.Info(ctx, "read shared environment variable", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
