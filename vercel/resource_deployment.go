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
		Attributes: map[string]tfsdk.Attribute{
			"domains": {
				Computed: true,
				Type: types.ListType{
					ElemType: types.StringType,
				},
			},
			"environment": {
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
			"team_id": {
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"project_id": {
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"url": {
				Computed: true,
				Type:     types.StringType,
			},
			"production": {
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.BoolType,
			},
			"files": {
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
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"build_command": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The build command for this project. If omitted, this value will be automatically detected",
					},
					"dev_command": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The dev command for this project. If omitted, this value will be automatically detected",
					},
					"framework": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The framework that is being used for this project. If omitted, no framework is selected",
					},
					"install_command": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The install command for this project. If omitted, this value will be automatically detected",
					},
					"output_directory": {
						Optional:      true,
						PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
						Type:          types.StringType,
						Description:   "The output directory of the project. When null is used this value will be automatically detected",
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

	target := "preview"
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
