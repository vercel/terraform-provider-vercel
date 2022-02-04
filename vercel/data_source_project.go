package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dataSourceProjectType struct{}

func (r dataSourceProjectType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides information about an existing project within Vercel.

A Project groups deployments and custom domains. To deploy on Vercel, you need a Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/overview).
        `,
		Attributes: map[string]tfsdk.Attribute{
			"team_id": {
				Optional:      true,
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Description:   "The team ID the project exists beneath.",
			},
			"name": {
				Required: true,
				Type:     types.StringType,
				Validators: []tfsdk.AttributeValidator{
					stringLengthBetween(1, 52),
				},
				Description: "The name of the project.",
			},
			"build_command": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The build command for this project. If omitted, this value will be automatically detected.",
			},
			"dev_command": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The dev command for this project. If omitted, this value will be automatically detected.",
			},
			"environment": {
				Description: "A list of environment variables that should be configured for the project.",
				Computed:    true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"target": {
						Description: "The environments that the environment variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
						Type: types.SetType{
							ElemType: types.StringType,
						},
						Computed: true,
					},
					"key": {
						Description: "The name of the environment variable.",
						Type:        types.StringType,
						Computed:    true,
					},
					"value": {
						Description: "The value of the environment variable.",
						Type:        types.StringType,
						Computed:    true,
					},
					"id": {
						Description: "The ID of the environment variable",
						Type:        types.StringType,
						Computed:    true,
					},
				}, tfsdk.ListNestedAttributesOptions{}),
			},
			"framework": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The framework that is being used for this project. If omitted, no framework is selected.",
				Validators: []tfsdk.AttributeValidator{
					validateFramework(),
				},
			},
			"git_repository": {
				Description:   "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed. This requires the corresponding Vercel for [Github](https://vercel.com/docs/concepts/git/vercel-for-github), [Gitlab](https://vercel.com/docs/concepts/git/vercel-for-gitlab) or [Bitbucket](https://vercel.com/docs/concepts/git/vercel-for-bitbucket) plugins to be installed.",
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type": {
						Description: "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
						Type:        types.StringType,
						Required:    true,
						Validators: []tfsdk.AttributeValidator{
							stringOneOf("github", "gitlab", "bitbucket"),
						},
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
					},
					"repo": {
						Description:   "The name of the git repository. For example: `vercel/next.js`.",
						Type:          types.StringType,
						Required:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
					},
				}),
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"install_command": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The install command for this project. If omitted, this value will be automatically detected.",
			},
			"output_directory": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The output directory of the project. When null is used this value will be automatically detected.",
			},
			"public_source": {
				Computed:    true,
				Type:        types.BoolType,
				Description: "Specifies whether the source code and logs of the deployments for this project should be public or not.",
			},
			"root_directory": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root.",
			},
		},
	}, nil
}

func (r dataSourceProjectType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceProject{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceProject struct {
	p provider
}

func (r dataSourceProject) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var config Project
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetProject(ctx, config.Name.Value, config.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project %s %s, unexpected error: %s",
				config.TeamID.Value,
				config.Name.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToProject(out, config.TeamID, types.String{})
	tflog.Trace(ctx, "read project", "team_id", result.TeamID.Value, "project_id", result.ID.Value)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
