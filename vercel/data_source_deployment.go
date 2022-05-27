package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dataSourceDeploymentType struct{}

// GetSchema returns the schema information for a deployment data source
func (r dataSourceDeploymentType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides information about an existing deployment within Vercel.

A Deployment is the result of building your Project and making it available through a live URL.
        `,
		Attributes: map[string]tfsdk.Attribute{
			"domains": {
				Description: "A list of all the domains (default domains, staging domains and production domains) that were assigned upon deployment creation.",
				Computed:    true,
				Type: types.ListType{
					ElemType: types.StringType,
				},
			},
			"environment": {
				Description:   "A map of environment variable names to values. These are specific to a Deployment, and can also be configured on the `vercel_project` resource.",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
			"team_id": {
				Description:   "The team ID to add the deployment to.",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"project_id": {
				Description:   "The project ID to add the deployment to.",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"path_prefix": {
				Description:   "If specified then the `path_prefix` will be stripped from the start of file paths as they are uploaded to Vercel. If this is omitted, then any leading `../`s will be stripped.",
				Optional:      true,
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
			},
			"url": {
				Description: "A unique URL that is automatically generated for a deployment.",
				Computed:    true,
				Type:        types.StringType,
			},
			"production": {
				Description:   "true if the deployment is a production deployment, meaning production aliases will be assigned.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.BoolType,
			},
			"files": {
				Description:   "A map of files to be uploaded for the deployment. This should be provided by a `vercel_project_directory` or `vercel_file` data source. Required if `git_source` is not set.",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Validators: []tfsdk.AttributeValidator{
					mapItemsMinCount(1),
				},
			},
			"ref": {
				Description:   "The branch or commit hash that should be deployed. Note this will only work if the project is configured to use a Git repository. Required if `ref` is not set.",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"project_settings": {
				Description:   "Project settings that will be applied to the deployment.",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"build_command": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The build command for this deployment. If omitted, this value will be taken from the project or automatically detected.",
					},
					"framework": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The framework that is being used for this deployment. If omitted, no framework is selected.",
						Validators: []tfsdk.AttributeValidator{
							validateFramework(),
						},
					},
					"install_command": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The install command for this deployment. If omitted, this value will be taken from the project or automatically detected.",
					},
					"output_directory": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The output directory of the deployment. If omitted, this value will be taken from the project or automatically detected.",
					},
					"root_directory": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root.",
					},
				}),
			},
			"delete_on_destroy": {
				Description: "Set to true to hard delete the Vercel deployment when destroying the Terraform resource. If unspecified, deployments are retained indefinitely. Note that deleted deployments are not recoverable.",
				Optional:    true,
				Type:        types.BoolType,
			},
		},
	}, nil
}

// NewDataSource instantiates a new DataSource of this DataSourceType.
func (r dataSourceDeploymentType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceDeployment{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceDeployment struct {
	p provider
}

// Read will read the deployment information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (r dataSourceDeployment) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var config Deployment
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetDeployment(ctx, config.ID.Value, config.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment",
			fmt.Sprintf("Could not read deployment %s %s, unexpected error: %s",
				config.TeamID.Value,
				config.ID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToDeployment(out, config)
	tflog.Trace(ctx, "read project", map[string]interface{}{
		"team_id":      result.TeamID.Value,
		"deplument_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
