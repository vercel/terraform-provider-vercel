package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &projectDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

func newProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

type projectDataSource struct {
	client *client.Client
}

func (d *projectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for a project data source
func (d *projectDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing project within Vercel.

A Project groups deployments and custom domains. To deploy on Vercel, you need a Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/overview).
        `,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team ID the project exists beneath. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringLengthBetween(1, 52),
					stringRegex(
						regexp.MustCompile(`^[a-z0-9\-]{0,100}$`),
						"The name of a Project can only contain up to 100 alphanumeric lowercase characters and hyphens",
					),
				},
				Description: "The name of the project.",
			},
			"build_command": schema.StringAttribute{
				Computed:    true,
				Description: "The build command for this project. If omitted, this value will be automatically detected.",
			},
			"dev_command": schema.StringAttribute{
				Computed:    true,
				Description: "The dev command for this project. If omitted, this value will be automatically detected.",
			},
			"ignore_command": schema.StringAttribute{
				Computed:    true,
				Description: "When a commit is pushed to the Git repository that is connected with your Project, its SHA will determine if a new Build has to be issued. If the SHA was deployed before, no new Build will be issued. You can customize this behavior with a command that exits with code 1 (new Build needed) or code 0.",
			},
			"serverless_function_region": schema.StringAttribute{
				Computed:    true,
				Description: "The region on Vercel's network to which your Serverless Functions are deployed. It should be close to any data source your Serverless Function might depend on. A new Deployment is required for your changes to take effect. Please see [Vercel's documentation](https://vercel.com/docs/concepts/edge-network/regions) for a full list of regions.",
			},
			"environment": schema.SetNestedAttribute{
				Description: "A list of environment variables that should be configured for the project.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.SetAttribute{
							Description: "The environments that the environment variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"key": schema.StringAttribute{
							Description: "The name of the environment variable.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the environment variable.",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the environment variable",
							Computed:    true,
						},
						"git_branch": schema.StringAttribute{
							Description: "The git branch of the environment variable.",
							Computed:    true,
						},
						"sensitive": schema.BoolAttribute{
							Description: "Whether the Environment Variable is sensitive or not. Note that the value will be `null` for sensitive environment variables.",
							Computed:    true,
						},
						"comment": schema.StringAttribute{
							Description: "A comment explaining what the environment variable is for.",
							Computed:    true,
						},
					},
				},
			},
			"framework": schema.StringAttribute{
				Computed:    true,
				Description: "The framework that is being used for this project. If omitted, no framework is selected.",
			},
			"git_repository": schema.SingleNestedAttribute{
				Description: "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed. This requires the corresponding Vercel for [Github](https://vercel.com/docs/concepts/git/vercel-for-github), [Gitlab](https://vercel.com/docs/concepts/git/vercel-for-gitlab) or [Bitbucket](https://vercel.com/docs/concepts/git/vercel-for-bitbucket) plugins to be installed.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
						Computed:    true,
						Validators: []validator.String{
							stringOneOf("github", "gitlab", "bitbucket"),
						},
					},
					"repo": schema.StringAttribute{
						Description: "The name of the git repository. For example: `vercel/next.js`.",
						Computed:    true,
					},
					"production_branch": schema.StringAttribute{
						Description: "By default, every commit pushed to the main branch will trigger a Production Deployment instead of the usual Preview Deployment. You can switch to a different branch here.",
						Computed:    true,
					},
					"deploy_hooks": schema.SetNestedAttribute{
						Description: "Deploy hooks are unique URLs that allow you to trigger a deployment of a given branch. See https://vercel.com/docs/deployments/deploy-hooks for full information.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "The ID of the deploy hook.",
									Computed:    true,
								},
								"name": schema.StringAttribute{
									Description: "The name of the deploy hook.",
									Computed:    true,
								},
								"ref": schema.StringAttribute{
									Description: "The branch or commit hash that should be deployed.",
									Computed:    true,
								},
								"url": schema.StringAttribute{
									Description: "A URL that, when a POST request is made to, will trigger a new deployment.",
									Computed:    true,
									Sensitive:   true,
								},
							},
						},
					},
				},
			},
			"vercel_authentication": schema.SingleNestedAttribute{
				Description: "Ensures visitors to your Preview Deployments are logged into Vercel and have a minimum of Viewer access on your team.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Description: "The deployment environment that will be protected.",
						Computed:    true,
					},
				},
			},
			"password_protection": schema.SingleNestedAttribute{
				Description: "Ensures visitors of your Preview Deployments must enter a password in order to gain access.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Description: "The deployment environment that will be protected.",
						Computed:    true,
					},
				},
			},
			"trusted_ips": schema.SingleNestedAttribute{
				Description: "Ensures only visitors from an allowed IP address can access your deployment.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Description: "The deployment environment that will be protected.",
						Computed:    true,
					},
					"addresses": schema.ListAttribute{
						Description: "The allowed IP addressses and CIDR ranges with optional descriptions.",
						Computed:    true,
						ElementType: types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"value": types.StringType,
								"note":  types.StringType,
							},
						},
					},
					"protection_mode": schema.StringAttribute{
						Description: "Whether or not Trusted IPs is required or optional to access a deployment.",
						Computed:    true,
					},
				},
			},
			"oidc_token_config": schema.SingleNestedAttribute{
				Description: "Configuration for OpenID Connect (OIDC) tokens.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "When true, Vercel issued OpenID Connect (OIDC) tokens will be available on the compute environments. See https://vercel.com/docs/security/secure-backend-access/oidc for more information.",
						Computed:    true,
					},
					"issuer_mode": schema.StringAttribute{
						Description: "Configures the URL of the `iss` claim. `team` = `https://oidc.vercel.com/[team_slug]` `global` = `https://oidc.vercel.com`",
						Computed:    true,
						Optional:    true,
						Validators: []validator.String{
							stringOneOf("team", "global"),
						},
					},
				},
			},
			"options_allowlist": schema.SingleNestedAttribute{
				Description: "Disable Deployment Protection for CORS preflight `OPTIONS` requests for a list of paths.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"paths": schema.ListAttribute{
						Description: "The allowed paths for the OPTIONS Allowlist. Incoming requests will bypass Deployment Protection if they have the method `OPTIONS` and **start with** one of the path values.",
						Computed:    true,
						ElementType: types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"value": types.StringType,
							},
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"install_command": schema.StringAttribute{
				Computed:    true,
				Description: "The install command for this project. If omitted, this value will be automatically detected.",
			},
			"output_directory": schema.StringAttribute{
				Computed:    true,
				Description: "The output directory of the project. When null is used this value will be automatically detected.",
			},
			"public_source": schema.BoolAttribute{
				Computed:    true,
				Description: "Specifies whether the source code and logs of the deployments for this project should be public or not.",
			},
			"root_directory": schema.StringAttribute{
				Computed:    true,
				Description: "The name of a directory or relative path to the source code of your project. When null is used it will default to the project root.",
			},
			"protection_bypass_for_automation": schema.BoolAttribute{
				Computed:    true,
				Description: "Allows automation services to bypass Vercel Authentication and Password Protection for both Preview and Production Deployments on this project when using an HTTP header named `x-vercel-protection-bypass`.",
			},
			"automatically_expose_system_environment_variables": schema.BoolAttribute{
				Computed:    true,
				Description: "Vercel provides a set of Environment Variables that are automatically populated by the System, such as the URL of the Deployment or the name of the Git branch deployed. To expose them to your Deployments, enable this field",
			},
			"git_comments": schema.SingleNestedAttribute{
				Description: "Configuration for Git Comments.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"on_pull_request": schema.BoolAttribute{
						Description: "Whether Pull Request comments are enabled",
						Required:    true,
					},
					"on_commit": schema.BoolAttribute{
						Description: "Whether Commit comments are enabled",
						Required:    true,
					},
				},
			},
			"preview_comments": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether comments are enabled on your Preview Deployments.",
			},
			"auto_assign_custom_domains": schema.BoolAttribute{
				Computed:    true,
				Description: "Automatically assign custom production domains after each Production deployment via merge to the production branch or Vercel CLI deploy with --prod. Defaults to `true`",
			},
			"git_lfs": schema.BoolAttribute{
				Computed:    true,
				Description: "Enables Git LFS support. Git LFS replaces large files such as audio samples, videos, datasets, and graphics with text pointers inside Git, while storing the file contents on a remote server like GitHub.com or GitHub Enterprise.",
			},
			"function_failover": schema.BoolAttribute{
				Computed:    true,
				Description: "Automatically failover Serverless Functions to the nearest region. You can customize regions through vercel.json. A new Deployment is required for your changes to take effect.",
			},
			"customer_success_code_visibility": schema.BoolAttribute{
				Computed:    true,
				Description: "Allows Vercel Customer Support to inspect all Deployments' source code in this project to assist with debugging.",
			},
			"git_fork_protection": schema.BoolAttribute{
				Computed:    true,
				Description: "Ensures that pull requests targeting your Git repository must be authorized by a member of your Team before deploying if your Project has Environment Variables or if the pull request includes a change to vercel.json.",
			},
			"prioritise_production_builds": schema.BoolAttribute{
				Computed:    true,
				Description: "If enabled, builds for the Production environment will be prioritized over Preview environments.",
			},
			"directory_listing": schema.BoolAttribute{
				Computed:    true,
				Description: "If no index file is present within a directory, the directory contents will be displayed.",
			},
			"skew_protection": schema.StringAttribute{
				Computed:    true,
				Description: "Ensures that outdated clients always fetch the correct version for a given deployment. This value defines how long Vercel keeps Skew Protection active.",
			},
			"resource_config": schema.SingleNestedAttribute{
				Description: "Resource Configuration for the project.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					// This is actually "function_default_memory_type" in the API schema, but for better convention, we use "cpu" and do translation in the provider.
					"function_default_cpu_type": schema.StringAttribute{
						Description: "The amount of CPU available to your Serverless Functions. Should be one of 'standard_legacy' (0.6vCPU), 'standard' (1vCPU) or 'performance' (1.7vCPUs).",
						Computed:    true,
					},
					"function_default_timeout": schema.Int64Attribute{
						Description: "The default timeout for Serverless Functions.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Project reflects the state terraform stores internally for a project.
type ProjectDataSource struct {
	BuildCommand                  types.String          `tfsdk:"build_command"`
	DevCommand                    types.String          `tfsdk:"dev_command"`
	Environment                   types.Set             `tfsdk:"environment"`
	Framework                     types.String          `tfsdk:"framework"`
	GitRepository                 *GitRepository        `tfsdk:"git_repository"`
	ID                            types.String          `tfsdk:"id"`
	IgnoreCommand                 types.String          `tfsdk:"ignore_command"`
	InstallCommand                types.String          `tfsdk:"install_command"`
	Name                          types.String          `tfsdk:"name"`
	OutputDirectory               types.String          `tfsdk:"output_directory"`
	PublicSource                  types.Bool            `tfsdk:"public_source"`
	RootDirectory                 types.String          `tfsdk:"root_directory"`
	ServerlessFunctionRegion      types.String          `tfsdk:"serverless_function_region"`
	TeamID                        types.String          `tfsdk:"team_id"`
	VercelAuthentication          *VercelAuthentication `tfsdk:"vercel_authentication"`
	PasswordProtection            *PasswordProtection   `tfsdk:"password_protection"`
	TrustedIps                    *TrustedIps           `tfsdk:"trusted_ips"`
	OIDCTokenConfig               *OIDCTokenConfig      `tfsdk:"oidc_token_config"`
	OptionsAllowlist              *OptionsAllowlist     `tfsdk:"options_allowlist"`
	ProtectionBypassForAutomation types.Bool            `tfsdk:"protection_bypass_for_automation"`
	AutoExposeSystemEnvVars       types.Bool            `tfsdk:"automatically_expose_system_environment_variables"`
	GitComments                   types.Object          `tfsdk:"git_comments"`
	PreviewComments               types.Bool            `tfsdk:"preview_comments"`
	AutoAssignCustomDomains       types.Bool            `tfsdk:"auto_assign_custom_domains"`
	GitLFS                        types.Bool            `tfsdk:"git_lfs"`
	FunctionFailover              types.Bool            `tfsdk:"function_failover"`
	CustomerSuccessCodeVisibility types.Bool            `tfsdk:"customer_success_code_visibility"`
	GitForkProtection             types.Bool            `tfsdk:"git_fork_protection"`
	PrioritiseProductionBuilds    types.Bool            `tfsdk:"prioritise_production_builds"`
	DirectoryListing              types.Bool            `tfsdk:"directory_listing"`
	SkewProtection                types.String          `tfsdk:"skew_protection"`
	ResourceConfig                *ResourceConfig       `tfsdk:"resource_config"`
}

func convertResponseToProjectDataSource(ctx context.Context, response client.ProjectResponse, plan Project, environmentVariables []client.EnvironmentVariable) (ProjectDataSource, error) {
	/* Force reading of environment and git comments. These are ignored usually if the planned value is null,
	   otherwise it causes issues with terraform thinking there are changes when there aren't. However,
	   for the data source we always want to read the value */
	plan.Environment = types.SetValueMust(envVariableElemType, []attr.Value{})
	plan.GitComments = types.ObjectNull(gitCommentsAttrTypes)
	if response.GitComments != nil {
		plan.GitComments = types.ObjectValueMust(gitCommentsAttrTypes, map[string]attr.Value{
			"on_pull_request": types.BoolValue(response.GitComments.OnPullRequest),
			"on_commit":       types.BoolValue(response.GitComments.OnCommit),
		})
	}

	if response.ResourceConfig != nil {
		plan.ResourceConfig = &ResourceConfig{
			FunctionDefaultMemoryType: types.StringValue(response.ResourceConfig.FunctionDefaultMemoryType),
			FunctionDefaultTimeout:    types.Int64Value(response.ResourceConfig.FunctionDefaultTimeout),
		}
	}

	project, err := convertResponseToProject(ctx, response, plan, environmentVariables)
	if err != nil {
		return ProjectDataSource{}, err
	}

	var pp *PasswordProtection
	if project.PasswordProtection != nil {
		pp = &PasswordProtection{
			DeploymentType: project.PasswordProtection.DeploymentType,
		}
	}
	return ProjectDataSource{
		BuildCommand:                  project.BuildCommand,
		DevCommand:                    project.DevCommand,
		Environment:                   project.Environment,
		Framework:                     project.Framework,
		GitRepository:                 project.GitRepository,
		ID:                            project.ID,
		IgnoreCommand:                 project.IgnoreCommand,
		InstallCommand:                project.InstallCommand,
		Name:                          project.Name,
		OutputDirectory:               project.OutputDirectory,
		PublicSource:                  project.PublicSource,
		RootDirectory:                 project.RootDirectory,
		ServerlessFunctionRegion:      project.ServerlessFunctionRegion,
		TeamID:                        project.TeamID,
		VercelAuthentication:          project.VercelAuthentication,
		PasswordProtection:            pp,
		TrustedIps:                    project.TrustedIps,
		OIDCTokenConfig:               project.OIDCTokenConfig,
		OptionsAllowlist:              project.OptionsAllowlist,
		AutoExposeSystemEnvVars:       types.BoolPointerValue(response.AutoExposeSystemEnvVars),
		ProtectionBypassForAutomation: project.ProtectionBypassForAutomation,
		GitComments:                   project.GitComments,
		PreviewComments:               project.PreviewComments,
		AutoAssignCustomDomains:       project.AutoAssignCustomDomains,
		GitLFS:                        project.GitLFS,
		FunctionFailover:              project.FunctionFailover,
		CustomerSuccessCodeVisibility: project.CustomerSuccessCodeVisibility,
		GitForkProtection:             project.GitForkProtection,
		PrioritiseProductionBuilds:    project.PrioritiseProductionBuilds,
		DirectoryListing:              project.DirectoryListing,
		SkewProtection:                project.SkewProtection,
		ResourceConfig:                project.ResourceConfig,
	}, nil
}

// Read will read project information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetProject(ctx, config.Name.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.Name.ValueString(),
				err,
			),
		)
		return
	}

	environmentVariables, err := d.client.GetEnvironmentVariables(ctx, out.ID, out.TeamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not read project, unexpected error: "+err.Error(),
		)
		return
	}
	result, err := convertResponseToProjectDataSource(ctx, out, nullProject, environmentVariables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting project response to model",
			"Could not read project, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Info(ctx, "read project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
