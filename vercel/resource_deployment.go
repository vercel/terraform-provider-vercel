package vercel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
	"github.com/vercel/terraform-provider-vercel/file"
)

var (
	_ resource.Resource              = &deploymentResource{}
	_ resource.ResourceWithConfigure = &deploymentResource{}
)

func newDeploymentResource() resource.Resource {
	return &deploymentResource{}
}

type deploymentResource struct {
	client *client.Client
}

func (r *deploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *deploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Schema returns the schema information for a deployment resource.
func (r *deploymentResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
		Attributes: map[string]schema.Attribute{
			"domains": schema.ListAttribute{
				Description:   "A list of all the domains (default domains, staging domains and production domains) that were assigned upon deployment creation.",
				Computed:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				ElementType:   types.StringType,
			},
			"environment": schema.MapAttribute{
				Description:   "A map of environment variable names to values. These are specific to a Deployment, and can also be configured on the `vercel_project` resource.",
				Optional:      true,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
				ElementType:   types.StringType,
			},
			"team_id": schema.StringAttribute{
				Description:   "The team ID to add the deployment to. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The project ID to add the deployment to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"path_prefix": schema.StringAttribute{
				Description:   "If specified then the `path_prefix` will be stripped from the start of file paths as they are uploaded to Vercel. If this is omitted, then any leading `../`s will be stripped.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Description:   "A unique URL that is automatically generated for a deployment.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"production": schema.BoolAttribute{
				Description:   "true if the deployment is a production deployment, meaning production aliases will be assigned.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"files": schema.MapAttribute{
				Description:   "A map of files to be uploaded for the deployment. This should be provided by a `vercel_project_directory` or `vercel_file` data source. Required if `git_source` is not set.",
				Optional:      true,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
				ElementType:   types.StringType,
				Validators: []validator.Map{
					mapItemsMinCount(1),
				},
			},
			"ref": schema.StringAttribute{
				Description:   "The branch or commit hash that should be deployed. Note this will only work if the project is configured to use a Git repository. Required if `ref` is not set.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_settings": schema.SingleNestedAttribute{
				Description:   "Project settings that will be applied to the deployment.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"build_command": schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Description:   "The build command for this deployment. If omitted, this value will be taken from the project or automatically detected.",
					},
					"framework": schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Description:   "The framework that is being used for this deployment. If omitted, no framework is selected.",
						Validators: []validator.String{
							validateFramework(),
						},
					},
					"install_command": schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Description:   "The install command for this deployment. If omitted, this value will be taken from the project or automatically detected.",
					},
					"output_directory": schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Description:   "The output directory of the deployment. If omitted, this value will be taken from the project or automatically detected.",
					},
					"root_directory": schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Description:   "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root.",
					},
				},
			},
			"delete_on_destroy": schema.BoolAttribute{
				Description: "Set to true to hard delete the Vercel deployment when destroying the Terraform resource. If unspecified, deployments are retained indefinitely. Note that deleted deployments are not recoverable.",
				Optional:    true,
			},
		},
	}
}

// ValidateConfig allows additional validation (specifically cross-field validation) to be added.
func (r *deploymentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config Deployment
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Ref.IsNull() && !config.Files.IsNull() {
		resp.Diagnostics.AddError(
			"Deployment Invalid",
			"A Deployment cannot have both `ref` and `files` specified",
		)
		return
	}
	if config.Ref.IsNull() && config.Files.IsNull() {
		resp.Diagnostics.AddError(
			"Deployment Invalid",
			"A Deployment must have either `ref` or `files` specified",
		)
		return
	}
}

func validatePrebuiltBuilds(diags AddErrorer, config Deployment, files []client.DeploymentFile) {
	buildsFilePath, ok := getPrebuiltBuildsFile(files)
	if !ok {
		// It's okay to not have a builds.json file. So allow this.
		return
	}

	builds, err := file.ReadBuildsJSON(buildsFilePath)
	if err != nil {
		diags.AddError(
			"Error reading prebuilt output",
			fmt.Sprintf(
				"An unexpected error occurred reading the prebuilt output builds.json: %s",
				err,
			),
		)
		return
	}

	target := "preview"
	if config.Production.ValueBool() {
		target = "production"
	}

	// Verify that the target matches what we hope the target is for the deployment.
	if (builds.Target != "production" && target == "production") ||
		(builds.Target == "production" && target != "production") {
		diags.AddError(
			"Prebuilt deployment cannot be used",
			fmt.Sprintf(
				"The prebuilt deployment at `%s` was built with the target environment %s, but the deployment targets environment %s",
				buildsFilePath,
				builds.Target,
				target,
			),
		)
		return
	}
}

func getPrebuiltBuildsFile(files []client.DeploymentFile) (string, bool) {
	for _, f := range files {
		if strings.HasSuffix(f.File, filepath.Join(".vercel", "output", "builds.json")) {
			return f.File, true
		}
	}
	return "", false
}

func filterNullFromMap(m map[string]types.String) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if !v.IsNull() {
			out[k] = v.ValueString()
		}
	}
	return out
}

// Create will create a deployment within Vercel. This is done by first attempting to trigger a deployment, seeing what
// files are required, uploading those files, and then attempting to create a deployment again.
// This is called automatically by the provider when a new resource should be created.
func (r *deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	validatePrebuiltBuilds(&resp.Diagnostics, plan, files)
	if resp.Diagnostics.HasError() {
		return
	}

	var environment map[string]types.String
	diags = plan.Environment.ElementsAs(ctx, &environment, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	target := ""
	if plan.Production.ValueBool() {
		target = "production"
	}
	cdr := client.CreateDeploymentRequest{
		Files:           files,
		Environment:     filterNullFromMap(environment),
		ProjectID:       plan.ProjectID.ValueString(),
		ProjectSettings: plan.ProjectSettings.toRequest(),
		Target:          target,
		Ref:             plan.Ref.ValueString(),
	}

	_, err = r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), false)
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating deployment",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	out, err := r.client.CreateDeployment(ctx, cdr, plan.TeamID.ValueString())
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

			err = r.client.CreateFile(ctx, client.CreateFileRequest{
				Filename: normaliseFilename(f.File, plan.PathPrefix),
				SHA:      f.Sha,
				Content:  string(content),
				TeamID:   plan.TeamID.ValueString(),
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

		out, err = r.client.CreateDeployment(ctx, cdr, plan.TeamID.ValueString())
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
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a file from the filesytem and provide terraform with information about it.
// It is called by the provider whenever data source values should be read to update state.
func (r *deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Deployment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetDeployment(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment",
			fmt.Sprintf("Could not get deployment %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToDeployment(out, state)
	tflog.Trace(ctx, "read deployment", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
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
func (r *deploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
func (r *deploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Deployment
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeleteOnDestroy.ValueBool() {
		dResp, err := r.client.DeleteDeployment(ctx, state.ID.ValueString(), state.TeamID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting deployment",
				fmt.Sprintf(
					"Could not delete deployment %s, unexpected error: %s",
					state.URL.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Trace(ctx, fmt.Sprintf("deleted deployment %s", dResp.UID))
	}
}
