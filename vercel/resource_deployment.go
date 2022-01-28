package vercel

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

type resourceDeploymentType struct{}

func (r resourceDeploymentType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a Deployment resource.

A Deployment is the result of building your Project and making it available through a live URL.

When making deployments, the Project will be uploaded and transformed into a production-ready output through the use of a Build Step.

Once the build step has completed successfully, a new, immutable deployment will be made available at the preview URL. Deployments are retained indefinitely unless deleted manually.

-> In order to provide files for a deployment, you'll need to use the ` + "`vercel_file` or `vercel_project_directory` data sources.",
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
				Description:   "A map of files to be uploaded for the deployment. This should be provided by a `vercel_project_directory` or `vercel_file` data source.",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Validators: []tfsdk.AttributeValidator{
					mapItemsMinCount(1),
				},
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
						Description:   "The build command for this deployment. If omitted, this value will be taken from the project or automatically detected",
					},
					"framework": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The framework that is being used for this deployment. If omitted, no framework is selected",
					},
					"install_command": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The install command for this deployment. If omitted, this value will be taken from the project or automatically detected",
					},
					"output_directory": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The output directory of the deployment. If omitted, this value will be taken from the project or automatically detected",
					},
					"root_directory": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root",
					},
				}),
			},
		},
	}, nil
}

func (r resourceDeploymentType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceDeployment{
		p: *(p.(*provider)),
	}, nil
}

type resourceDeployment struct {
	p provider
}

func (r resourceDeployment) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var plan Deployment
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting deployment plan",
			"Error getting deployment plan",
		)
		return
	}

	files, filesBySha, err := plan.getFiles()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment",
			"Could not parse files, unexpected error: "+err.Error(),
		)
		return
	}

	target := ""
	if plan.Production.Value {
		target = "production"
	}
	var environment map[string]string
	diags = plan.Environment.ElementsAs(ctx, &environment, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cdr := client.CreateDeploymentRequest{
		Files:           files,
		Environment:     environment,
		ProjectID:       plan.ProjectID.Value,
		ProjectSettings: plan.ProjectSettings.toRequest(),
		Target:          target,
	}

	out, err := r.p.client.CreateDeployment(ctx, cdr, plan.TeamID.Value)
	var mfErr client.MissingFilesError
	if errors.As(err, &mfErr) {
		// Then we need to upload the files, and create the deployment again.
		for _, sha := range mfErr.Missing {
			f := filesBySha[sha]
			content, err := os.ReadFile(f.File)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading file",
					fmt.Sprintf(
						"Could not read file %s, unexpected error: %s",
						f.File,
						err,
					),
				)
				return
			}

			err = r.p.client.CreateFile(ctx, f.File, f.Sha, string(content))
			if err != nil {
				resp.Diagnostics.AddError(
					"Error uploading deployment file",
					fmt.Sprintf(
						"Could not upload deployment file %s, unexpected error: %s",
						f.File,
						err,
					),
				)
				return
			}
		}

		out, err = r.p.client.CreateDeployment(ctx, cdr, plan.TeamID.Value)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating deployment",
				"Could not create deployment, unexpected error: "+err.Error(),
			)
			return
		}
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment",
			"Could not create deployment, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToDeployment(out, plan)
	tflog.Trace(ctx, "created deployment", "team_id", result.TeamID.Value, "project_id", result.ID.Value)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceDeployment) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state Deployment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetDeployment(ctx, state.ID.Value, state.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment",
			fmt.Sprintf("Could not get project %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.URL.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToDeployment(out, state)
	tflog.Trace(ctx, "read deployment", "team_id", result.TeamID.Value, "project_id", result.ID.Value)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceDeployment) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Nothing to do here - we can't update deployments
}

func (r resourceDeployment) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	tflog.Trace(ctx, "deleted deployment")
	resp.State.RemoveResource(ctx)
}

func (r resourceDeployment) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStateNotImplemented(ctx, "", resp)
}
