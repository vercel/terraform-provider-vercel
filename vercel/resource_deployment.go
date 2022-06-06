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

// GetSchema returns the schema information for a deployment resource.
func (r resourceDeploymentType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a Deployment resource.

A Deployment is the result of building your Project and making it available through a live URL.

When making deployments, the Project will be uploaded and transformed into a production-ready output through the use of a Build Step.

Once the build step has completed successfully, a new, immutable deployment will be made available at the preview URL. Deployments are retained indefinitely unless deleted manually.

-> In order to provide files to a deployment, you'll need to use the ` + "`vercel_file` or `vercel_project_directory` data sources." + `

~> If you are creating Deployments through terraform and intend to use both preview and production
deployments, you may wish to 'layer' your terraform, creating the Project with a different set of
terraform to your Deployment.
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

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceDeploymentType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceDeployment{
		p: *(p.(*provider)),
	}, nil
}

type resourceDeployment struct {
	p provider
}

// ValidateConfig allows additional validation (specifically cross-field validation) to be added.
func (r resourceDeployment) ValidateConfig(ctx context.Context, req tfsdk.ValidateResourceConfigRequest, resp *tfsdk.ValidateResourceConfigResponse) {
	var config Deployment
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Ref.Null && !config.Files.Null {
		resp.Diagnostics.AddError(
			"Deployment Invalid",
			"A Deployment cannot have both `ref` and `files` specified",
		)
	}
	if config.Ref.Null && config.Files.Null {
		resp.Diagnostics.AddError(
			"Deployment Invalid",
			"A Deployment must have either `ref` or `files` specified",
		)
	}
}

// Create will create a deployment within Vercel. This is done by first attempting to trigger a deployment, seeing what
// files are required, uploading those files, and then attempting to create a deployment again.
// This is called automatically by the provider when a new resource should be created.
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

	var unparsedFiles map[string]string
	diags = plan.Files.ElementsAs(ctx, &unparsedFiles, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	files, filesBySha, err := getFiles(unparsedFiles, plan.PathPrefix)
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
		Ref:             plan.Ref.Value,
	}

	_, err = r.p.client.GetProject(ctx, plan.ProjectID.Value, plan.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		resp.Diagnostics.AddError(
			"Error creating deployment",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
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

			err = r.p.client.CreateFile(ctx, client.CreateFileRequest{
				Filename: f.File,
				SHA:      f.Sha,
				Content:  string(content),
				TeamID:   plan.TeamID.Value,
			})
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
	tflog.Trace(ctx, "created deployment", map[string]interface{}{
		"team_id":    result.TeamID.Value,
		"project_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a file from the filesytem and provide terraform with information about it.
// It is called by the provider whenever data source values should be read to update state.
func (r resourceDeployment) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state Deployment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetDeployment(ctx, state.ID.Value, state.TeamID.Value)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment",
			fmt.Sprintf("Could not get deployment %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.ID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToDeployment(out, state)
	tflog.Trace(ctx, "read deployment", map[string]interface{}{
		"team_id":    result.TeamID.Value,
		"project_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the deployment state.
// Note that only the `delete_on_destroy` field is updatable, and this does not affect Vercel. So it is just a case
// of setting terraform state.
func (r resourceDeployment) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
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

	var state Deployment
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Copy over the planned field only
	state.DeleteOnDestroy = plan.DeleteOnDestroy
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete conditionally deletes a Deployment.
// Typically, Vercel users do not delete old Deployments so deployments will be deleted only if delete_on_destroy
// parameter is set to true.
func (r resourceDeployment) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state Deployment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeleteOnDestroy.Value {
		dResp, err := r.p.client.DeleteDeployment(ctx, state.ID.Value, state.TeamID.Value)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting deployment",
				fmt.Sprintf(
					"Could not delete deployment %s, unexpected error: %s",
					state.URL.Value,
					err,
				),
			)
			return
		}
		tflog.Trace(ctx, fmt.Sprintf("deleted deployment %s", dResp.UID))
	}
}
