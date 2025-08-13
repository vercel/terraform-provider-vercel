package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
	_ resource.ResourceWithModifyPlan  = &projectResource{}
	// _ resource.ResourceWithConfigValidators = &projectResource{}
)

func newProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *client.Client
}

func (r *projectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project resource.

A Project groups deployments and custom domains. To deploy on Vercel, you need to create a Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/overview).

~> Terraform currently provides a standalone Project Environment Variable resource (a single Environment Variable), a Project Environment Variables resource (multiple Environment Variables), and this Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variables` or `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
        `,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
				Description:   "The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 52),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9\-]{0,100}$`),
						"The name of a Project can only contain up to 100 alphanumeric lowercase characters and hyphens",
					),
				},
				Description: "The desired name for the project.",
			},
			"build_command": schema.StringAttribute{
				Optional:    true,
				Description: "The build command for this project. If omitted, this value will be automatically detected.",
			},
			"dev_command": schema.StringAttribute{
				Optional:    true,
				Description: "The dev command for this project. If omitted, this value will be automatically detected.",
			},
			"ignore_command": schema.StringAttribute{
				Optional:    true,
				Description: "When a commit is pushed to the Git repository that is connected with your Project, its SHA will determine if a new Build has to be issued. If the SHA was deployed before, no new Build will be issued. You can customize this behavior with a command that exits with code 1 (new Build needed) or code 0.",
			},
			"serverless_function_region": schema.StringAttribute{
				DeprecationMessage: "This attribute is deprecated. Please use resource_config.function_default_regions instead.",
				Optional:           true,
				Computed:           true,
				Description:        "The region on Vercel's network to which your Serverless Functions are deployed. It should be close to any data source your Serverless Function might depend on. A new Deployment is required for your changes to take effect. Please see [Vercel's documentation](https://vercel.com/docs/concepts/edge-network/regions) for a full list of regions.",
				Validators: []validator.String{
					validateServerlessFunctionRegion(),
					stringvalidator.ConflictsWith(
						path.MatchRoot("serverless_function_region"),
						path.MatchRoot("resource_config").AtName("function_default_regions"),
					),
				},
			},
			"node_version": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"18.x",
						"20.x",
						"22.x",
					),
				},
				Description: "The version of Node.js that is used in the Build Step and for Serverless Functions. A new Deployment is required for your changes to take effect.",
			},
			"environment": schema.SetNestedAttribute{
				Description: "A set of Environment Variables that should be configured for the project.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.SetAttribute{
							Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`. At least one of `target` or `custom_environment_ids` must be set.",
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(stringvalidator.OneOf("production", "preview", "development")),
								setvalidator.SizeAtLeast(1),
								setvalidator.AtLeastOneOf(
									path.MatchRelative().AtParent().AtName("target"),
									path.MatchRelative().AtParent().AtName("custom_environment_ids"),
								),
							},
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Set{
								setplanmodifier.UseStateForUnknown(),
							},
						},
						"custom_environment_ids": schema.SetAttribute{
							Description: "The IDs of Custom Environments that the Environment Variable should be present on. At least one of `target` or `custom_environment_ids` must be set.",
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.AtLeastOneOf(
									path.MatchRelative().AtParent().AtName("target"),
									path.MatchRelative().AtParent().AtName("custom_environment_ids"),
								),
							},
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Set{
								setplanmodifier.UseStateForUnknown(),
							},
						},
						"git_branch": schema.StringAttribute{
							Description: "The git branch of the Environment Variable.",
							Optional:    true,
						},
						"key": schema.StringAttribute{
							Description: "The name of the Environment Variable.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the Environment Variable.",
							Required:    true,
							Sensitive:   true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the Environment Variable.",
							Computed:    true,
						},
						"sensitive": schema.BoolAttribute{
							Description: "Whether the Environment Variable is sensitive or not. (May be affected by a [team-wide environment variable policy](https://vercel.com/docs/projects/environment-variables/sensitive-environment-variables#environment-variables-policy))",
							Optional:    true,
							Computed:    true,
						},
						"comment": schema.StringAttribute{
							Description: "A comment explaining what the environment variable is for.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(0, 1000),
							},
						},
					},
				},
			},
			"framework": schema.StringAttribute{
				Optional:    true,
				Description: "The framework that is being used for this project. If omitted, no framework is selected.",
				Validators: []validator.String{
					validateFramework(),
				},
			},
			"git_repository": schema.SingleNestedAttribute{
				Description:   "The Git Repository that will be connected to the project. When this is defined, any pushes to the specified connected Git Repository will be automatically deployed. This requires the corresponding Vercel for [Github](https://vercel.com/docs/concepts/git/vercel-for-github), [Gitlab](https://vercel.com/docs/concepts/git/vercel-for-gitlab) or [Bitbucket](https://vercel.com/docs/concepts/git/vercel-for-bitbucket) plugins to be installed.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The git provider of the repository. Must be either `github`, `gitlab`, or `bitbucket`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("github", "gitlab", "bitbucket"),
						},
					},
					"repo": schema.StringAttribute{
						Description: "The name of the git repository. For example: `vercel/next.js`.",
						Required:    true,
					},
					"production_branch": schema.StringAttribute{
						Description:   "By default, every commit pushed to the main branch will trigger a Production Deployment instead of the usual Preview Deployment. You can switch to a different branch here.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deploy_hooks": schema.SetNestedAttribute{
						Description: "Deploy hooks are unique URLs that allow you to trigger a deployment of a given branch. See https://vercel.com/docs/deployments/deploy-hooks for full information.",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "The ID of the deploy hook.",
									Computed:    true,
								},
								"name": schema.StringAttribute{
									Description: "The name of the deploy hook.",
									Required:    true,
								},
								"ref": schema.StringAttribute{
									Description: "The branch or commit hash that should be deployed.",
									Required:    true,
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
				Description:   "Ensures visitors to your Preview Deployments are logged into Vercel and have a minimum of Viewer access on your team.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Required:    true,
						Description: "The deployment environment to protect. The default value is `standard_protection_new` (Standard Protection). Must be one of `standard_protection_new` (Standard Protection), `standard_protection` (Legacy Standard Protection), `all_deployments`, `only_preview_deployments`, or `none`.",
						Validators: []validator.String{
							stringvalidator.OneOf("standard_protection_new", "standard_protection", "all_deployments", "only_preview_deployments", "none"),
						},
					},
				},
			},
			"password_protection": schema.SingleNestedAttribute{
				Description: "Ensures visitors of your Preview Deployments must enter a password in order to gain access.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"password": schema.StringAttribute{
						Description: "The password that visitors must enter to gain access to your Preview Deployments. Drift detection is not possible for this field.",
						Required:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 72),
						},
					},
					"deployment_type": schema.StringAttribute{
						Required:      true,
						Description:   "The deployment environment to protect. Must be one of `standard_protection_new` (Standard Protection), `standard_protection` (Legacy Standard Protection), `all_deployments`, or `only_preview_deployments`.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("standard_protection_new", "standard_protection", "all_deployments", "only_preview_deployments"),
						},
					},
				},
			},
			"trusted_ips": schema.SingleNestedAttribute{
				Description: "Ensures only visitors from an allowed IP address can access your deployment.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"addresses": schema.SetNestedAttribute{
						Description:   "The allowed IP addressses and CIDR ranges with optional descriptions.",
						Required:      true,
						PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Description: "The address or CIDR range that can access deployments.",
									Required:    true,
									Sensitive:   true,
								},
								"note": schema.StringAttribute{
									Description: "A description for the value",
									Optional:    true,
								},
							},
						},
						Validators: []validator.Set{
							setvalidator.SizeAtLeast(1),
						},
					},
					"deployment_type": schema.StringAttribute{
						Required:      true,
						Description:   "The deployment environment to protect. Must be one of `standard_protection_new` (Standard Protection), `standard_protection` (Legacy Standard Protection), `all_deployments`, `only_production_deployments`, or `only_preview_deployments`.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("standard_protection_new", "standard_protection", "all_deployments", "only_production_deployments", "only_preview_deployments"),
						},
					},
					"protection_mode": schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						Default:       stringdefault.StaticString("trusted_ip_required"),
						Description:   "Whether or not Trusted IPs is optional to access a deployment. Must be either `trusted_ip_required` or `trusted_ip_optional`. `trusted_ip_optional` is only available with Standalone Trusted IPs.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("trusted_ip_required", "trusted_ip_optional"),
						},
					},
				},
			},
			"oidc_token_config": schema.SingleNestedAttribute{
				Description: "Configuration for OpenID Connect (OIDC) tokens.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						DeprecationMessage: "This field is deprecated and will be removed in a future version.",
						Description:        "When true, Vercel issued OpenID Connect (OIDC) tokens will be available on the compute environments. See https://vercel.com/docs/security/secure-backend-access/oidc for more information.",
						Optional:           true,
						Computed:           true,
						Validators: []validator.Bool{
							onlyTrueValidator("This field is deprecated and can no longer be specified as 'false'"),
						},
					},
					"issuer_mode": schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						Default:       stringdefault.StaticString("team"),
						Description:   "Configures the URL of the `iss` claim. `team` = `https://oidc.vercel.com/[team_slug]` `global` = `https://oidc.vercel.com`",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("team", "global"),
						},
					},
				},
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"enabled":     types.BoolType,
						"issuer_mode": types.StringType,
					},
					map[string]attr.Value{
						"enabled":     types.BoolValue(true),
						"issuer_mode": types.StringValue("team"),
					},
				)),
			},
			"options_allowlist": schema.SingleNestedAttribute{
				Description: "Disable Deployment Protection for CORS preflight `OPTIONS` requests for a list of paths.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"paths": schema.SetNestedAttribute{
						Description:   "The allowed paths for the OPTIONS Allowlist. Incoming requests will bypass Deployment Protection if they have the method `OPTIONS` and **start with** one of the path values.",
						Required:      true,
						PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Description: "The path prefix to compare with the incoming request path.",
									Required:    true,
								},
							},
						},
						Validators: []validator.Set{
							setvalidator.SizeAtLeast(1),
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"install_command": schema.StringAttribute{
				Optional:    true,
				Description: "The install command for this project. If omitted, this value will be automatically detected.",
			},
			"output_directory": schema.StringAttribute{
				Optional:    true,
				Description: "The output directory of the project. If omitted, this value will be automatically detected.",
			},
			"public_source": schema.BoolAttribute{
				Optional:    true,
				Description: "By default, visitors to the `/_logs` and `/_src` paths of your Production and Preview Deployments must log in with Vercel (requires being a member of your team) to see the Source, Logs and Deployment Status of your project. Setting `public_source` to `true` disables this behaviour, meaning the Source, Logs and Deployment Status can be publicly viewed.",
			},
			"root_directory": schema.StringAttribute{
				Optional:    true,
				Description: "The name of a directory or relative path to the source code of your project. If omitted, it will default to the project root.",
			},
			"protection_bypass_for_automation": schema.BoolAttribute{
				Optional:    true,
				Description: "Allow automation services to bypass Deployment Protection on this project when using an HTTP header named `x-vercel-protection-bypass` with a value of the `protection_bypass_for_automation_secret` field.",
			},
			"protection_bypass_for_automation_secret": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Sensitive:   true,
				Description: "If `protection_bypass_for_automation` is enabled, optionally set this value to specify a 32 character secret, otherwise a secret will be generated.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]{32}$`),
						"Specify `generate` to have the value generated automatically, or specify a 32 character secret.",
					),
					validateAutomationBypassSecret(),
				},
			},
			"automatically_expose_system_environment_variables": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Vercel provides a set of Environment Variables that are automatically populated by the System, such as the URL of the Deployment or the name of the Git branch deployed. To expose them to your Deployments, enable this field",
			},
			"enable_affected_projects_deployments": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "When enabled, Vercel will automatically deploy all projects that are affected by a change to this project.",
			},
			"git_comments": schema.SingleNestedAttribute{
				Description: "Configuration for Git Comments.",
				Optional:    true,
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
				Description:        "Enables the Vercel Toolbar on your preview deployments.",
				DeprecationMessage: "Use `enable_preview_feedback` instead. This attribute will be removed in a future version.",
				Optional:           true,
				Computed:           true,
				Validators: []validator.Bool{boolvalidator.ConflictsWith(
					path.MatchRoot("preview_comments"),
					path.MatchRoot("enable_preview_feedback"),
				)},
			},
			"enable_preview_feedback": schema.BoolAttribute{
				Description: "Enables the Vercel Toolbar on your preview deployments.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Bool{boolvalidator.ConflictsWith(
					path.MatchRoot("preview_comments"),
					path.MatchRoot("enable_preview_feedback"),
				)},
			},
			"enable_production_feedback": schema.BoolAttribute{
				Description:   "Enables the Vercel Toolbar on your production deployments: one of on, off or default.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"preview_deployments_disabled": schema.BoolAttribute{
				Description:   "Disable creation of Preview Deployments for this project.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"auto_assign_custom_domains": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Automatically assign custom production domains after each Production deployment via merge to the production branch or Vercel CLI deploy with --prod. Defaults to `true`",
				Default:     booldefault.StaticBool(true),
			},
			"git_lfs": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Enables Git LFS support. Git LFS replaces large files such as audio samples, videos, datasets, and graphics with text pointers inside Git, while storing the file contents on a remote server like GitHub.com or GitHub Enterprise.",
			},
			"function_failover": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Automatically failover Serverless Functions to the nearest region. You can customize regions through vercel.json. A new Deployment is required for your changes to take effect.",
			},
			"customer_success_code_visibility": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Allows Vercel Customer Support to inspect all Deployments' source code in this project to assist with debugging.",
			},
			"git_fork_protection": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Ensures that pull requests targeting your Git repository must be authorized by a member of your Team before deploying if your Project has Environment Variables or if the pull request includes a change to vercel.json. Defaults to `true`.",
			},
			"prioritise_production_builds": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "If enabled, builds for the Production environment will be prioritized over Preview environments.",
			},
			"directory_listing": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "If no index file is present within a directory, the directory contents will be displayed.",
			},
			"skew_protection": schema.StringAttribute{
				Optional:    true,
				Description: "Ensures that outdated clients always fetch the correct version for a given deployment. This value defines how long Vercel keeps Skew Protection active.",
				Validators: []validator.String{
					stringvalidator.OneOf("30 minutes", "12 hours", "1 day", "7 days"),
				},
			},
			"resource_config": schema.SingleNestedAttribute{
				Description: "Resource Configuration for the project.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					// This is actually "function_default_memory_type" in the API schema, but for better convention, we use "cpu" and do translation in the provider.
					"function_default_cpu_type": schema.StringAttribute{
						Description: "The amount of CPU available to your Serverless Functions. Should be one of 'standard_legacy' (0.6vCPU), 'standard' (1vCPU) or 'performance' (1.7vCPUs).",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("standard_legacy", "standard", "performance"),
						},
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"function_default_timeout": schema.Int64Attribute{
						Description: "The default timeout for Serverless Functions.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
							int64validator.AtMost(900),
						},
						PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					},
					"function_default_regions": schema.SetAttribute{
						Description: "The default regions for Serverless Functions. Must be an array of valid region identifiers.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(
								validateServerlessFunctionRegion(),
							),
							setvalidator.ConflictsWith(
								path.MatchRoot("serverless_function_region"),
								path.MatchRoot("resource_config").AtName("function_default_regions"),
							),
						},
					},
					"fluid": schema.BoolAttribute{
						Description:   "Enable fluid compute for your Vercel Functions to automatically manage concurrency and optimize performance. Vercel will handle the defaults to ensure the best experience for your workload.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"on_demand_concurrent_builds": schema.BoolAttribute{
				Description:   "Instantly scale build capacity to skip the queue, even if all build slots are in use. You can also choose a larger build machine; charges apply per minute if it exceeds your team's default.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"build_machine_type": schema.StringAttribute{
				Description: "The build machine type to use for this project. Must be one of \"enhanced\" or \"turbo\".",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("enhanced", "turbo"),
				},
			},
		},
	}
}

/*
func (r *projectResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&fluidComputeBasicCPUValidator{},
	}
}
*/

// Project reflects the state terraform stores internally for a project.
type Project struct {
	BuildCommand                        types.String                    `tfsdk:"build_command"`
	DevCommand                          types.String                    `tfsdk:"dev_command"`
	Environment                         types.Set                       `tfsdk:"environment"`
	Framework                           types.String                    `tfsdk:"framework"`
	GitRepository                       *GitRepository                  `tfsdk:"git_repository"`
	ID                                  types.String                    `tfsdk:"id"`
	IgnoreCommand                       types.String                    `tfsdk:"ignore_command"`
	InstallCommand                      types.String                    `tfsdk:"install_command"`
	Name                                types.String                    `tfsdk:"name"`
	NodeVersion                         types.String                    `tfsdk:"node_version"`
	OutputDirectory                     types.String                    `tfsdk:"output_directory"`
	PublicSource                        types.Bool                      `tfsdk:"public_source"`
	RootDirectory                       types.String                    `tfsdk:"root_directory"`
	ServerlessFunctionRegion            types.String                    `tfsdk:"serverless_function_region"`
	TeamID                              types.String                    `tfsdk:"team_id"`
	VercelAuthentication                types.Object                    `tfsdk:"vercel_authentication"`
	PasswordProtection                  *PasswordProtectionWithPassword `tfsdk:"password_protection"`
	TrustedIps                          *TrustedIps                     `tfsdk:"trusted_ips"`
	OIDCTokenConfig                     *OIDCTokenConfig                `tfsdk:"oidc_token_config"`
	OptionsAllowlist                    *OptionsAllowlist               `tfsdk:"options_allowlist"`
	ProtectionBypassForAutomation       types.Bool                      `tfsdk:"protection_bypass_for_automation"`
	ProtectionBypassForAutomationSecret types.String                    `tfsdk:"protection_bypass_for_automation_secret"`
	AutoExposeSystemEnvVars             types.Bool                      `tfsdk:"automatically_expose_system_environment_variables"`
	GitComments                         types.Object                    `tfsdk:"git_comments"`
	PreviewComments                     types.Bool                      `tfsdk:"preview_comments"`
	EnablePreviewFeedback               types.Bool                      `tfsdk:"enable_preview_feedback"`
	EnableProductionFeedback            types.Bool                      `tfsdk:"enable_production_feedback"`
	PreviewDeploymentsDisabled          types.Bool                      `tfsdk:"preview_deployments_disabled"`
	AutoAssignCustomDomains             types.Bool                      `tfsdk:"auto_assign_custom_domains"`
	GitLFS                              types.Bool                      `tfsdk:"git_lfs"`
	FunctionFailover                    types.Bool                      `tfsdk:"function_failover"`
	CustomerSuccessCodeVisibility       types.Bool                      `tfsdk:"customer_success_code_visibility"`
	GitForkProtection                   types.Bool                      `tfsdk:"git_fork_protection"`
	PrioritiseProductionBuilds          types.Bool                      `tfsdk:"prioritise_production_builds"`
	DirectoryListing                    types.Bool                      `tfsdk:"directory_listing"`
	EnableAffectedProjectsDeployments   types.Bool                      `tfsdk:"enable_affected_projects_deployments"`
	SkewProtection                      types.String                    `tfsdk:"skew_protection"`
	ResourceConfig                      types.Object                    `tfsdk:"resource_config"`
	OnDemandConcurrentBuilds            types.Bool                      `tfsdk:"on_demand_concurrent_builds"`
	BuildMachineType                    types.String                    `tfsdk:"build_machine_type"`
}

type GitComments struct {
	OnPullRequest types.Bool `tfsdk:"on_pull_request"`
	OnCommit      types.Bool `tfsdk:"on_commit"`
}

func (g *GitComments) toUpdateProjectRequest() *client.GitComments {
	if g == nil {
		return nil
	}
	return &client.GitComments{
		OnPullRequest: g.OnPullRequest.ValueBool(),
		OnCommit:      g.OnCommit.ValueBool(),
	}
}

func (p Project) RequiresUpdateAfterCreation() bool {
	return p.PasswordProtection != nil ||
		p.TrustedIps != nil ||
		p.OIDCTokenConfig != nil ||
		p.OptionsAllowlist != nil ||
		!p.AutoExposeSystemEnvVars.IsNull() ||
		p.GitComments.IsNull() ||
		(!p.AutoAssignCustomDomains.IsNull() && !p.AutoAssignCustomDomains.ValueBool()) ||
		!p.GitLFS.IsNull() ||
		!p.FunctionFailover.IsNull() ||
		!p.CustomerSuccessCodeVisibility.IsNull() ||
		(!p.GitForkProtection.IsNull() && !p.GitForkProtection.ValueBool()) ||
		!p.PrioritiseProductionBuilds.IsNull() ||
		!p.DirectoryListing.IsNull() ||
		!p.SkewProtection.IsNull() ||
		!p.NodeVersion.IsNull()

}

var nullProject = Project{
	/* As this is read only, none of these fields are specified - so treat them all as Null */
	BuildCommand:    types.StringNull(),
	DevCommand:      types.StringNull(),
	InstallCommand:  types.StringNull(),
	OutputDirectory: types.StringNull(),
	PublicSource:    types.BoolNull(),
	Environment:     types.SetNull(envVariableElemType),
}

func (p *Project) environment(ctx context.Context) ([]EnvironmentItem, error) {
	if p.Environment.IsNull() || p.Environment.IsUnknown() {
		return nil, nil
	}

	var vars []EnvironmentItem
	err := p.Environment.ElementsAs(ctx, &vars, true)
	if err != nil {
		return nil, fmt.Errorf("error reading project environment variables: %s", err)
	}
	return vars, nil
}

func parseEnvironment(ctx context.Context, vars []EnvironmentItem) (out []client.EnvironmentVariable, diags diag.Diagnostics) {
	for _, e := range vars {
		var target []string
		diags = e.Target.ElementsAs(ctx, &target, true)
		if diags.HasError() {
			return out, diags
		}
		var customEnvironmentIDs []string
		diags = e.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
		if diags.HasError() {
			return out, diags
		}

		var envVariableType string
		if e.Sensitive.ValueBool() {
			envVariableType = "sensitive"
		} else {
			envVariableType = "encrypted"
		}

		out = append(out, client.EnvironmentVariable{
			Key:                  e.Key.ValueString(),
			Value:                e.Value.ValueString(),
			Target:               target,
			CustomEnvironmentIDs: customEnvironmentIDs,
			GitBranch:            e.GitBranch.ValueStringPointer(),
			Type:                 envVariableType,
			ID:                   e.ID.ValueString(),
			Comment:              e.Comment.ValueString(),
		})
	}
	return out, nil
}

func (p *Project) toCreateProjectRequest(ctx context.Context, envs []EnvironmentItem) (req client.CreateProjectRequest, diags diag.Diagnostics) {
	clientEnvs, diags := parseEnvironment(ctx, envs)
	if diags.HasError() {
		return req, diags
	}
	resourceConfig, diags := p.resourceConfig(ctx)
	if diags.HasError() {
		return req, diags
	}
	vercelAuthentication, diags := p.vercelAuthentication(ctx)
	if diags.HasError() {
		return req, diags
	}

	return client.CreateProjectRequest{
		BuildCommand:                      p.BuildCommand.ValueStringPointer(),
		CommandForIgnoringBuildStep:       p.IgnoreCommand.ValueStringPointer(),
		DevCommand:                        p.DevCommand.ValueStringPointer(),
		EnableAffectedProjectsDeployments: p.EnableAffectedProjectsDeployments.ValueBoolPointer(),
		EnvironmentVariables:              clientEnvs,
		Framework:                         p.Framework.ValueStringPointer(),
		GitRepository:                     p.GitRepository.toCreateProjectRequest(),
		InstallCommand:                    p.InstallCommand.ValueStringPointer(),
		Name:                              p.Name.ValueString(),
		OIDCTokenConfig:                   p.OIDCTokenConfig.toCreateProjectRequest(),
		OutputDirectory:                   p.OutputDirectory.ValueStringPointer(),
		PublicSource:                      p.PublicSource.ValueBoolPointer(),
		RootDirectory:                     p.RootDirectory.ValueStringPointer(),
		ResourceConfig:                    resourceConfig.toClientResourceConfig(ctx, p.OnDemandConcurrentBuilds, p.BuildMachineType, p.ServerlessFunctionRegion),
		EnablePreviewFeedback:             oneBoolPointer(p.EnablePreviewFeedback, p.PreviewComments),
		EnableProductionFeedback:          p.EnableProductionFeedback.ValueBoolPointer(),
		VercelAuthentication:              vercelAuthentication.toVercelAuthentication(),
		PreviewDeploymentsDisabled:        p.PreviewDeploymentsDisabled.ValueBoolPointer(),
	}, diags
}

func toSkewProtectionAge(sp types.String) int {
	if sp.IsNull() || sp.IsUnknown() {
		return 0
	}
	var ages = map[string]int{
		"30 minutes": 1800,
		"12 hours":   43200,
		"1 day":      86400,
		"7 days":     604800,
	}
	v, ok := ages[sp.ValueString()]
	if !ok {
		// Should not happen due to validation
		return 0
	}
	return v
}

func oneBoolPointer(a, b types.Bool) *bool {
	if !a.IsNull() && !a.IsUnknown() {
		return a.ValueBoolPointer()
	}
	if !b.IsNull() && !b.IsUnknown() {
		return b.ValueBoolPointer()
	}
	return nil
}

func (p *Project) toUpdateProjectRequest(ctx context.Context, oldName string) (req client.UpdateProjectRequest, diags diag.Diagnostics) {
	var name *string = nil
	if oldName != p.Name.ValueString() {
		n := p.Name.ValueString()
		name = &n
	}
	var gc *GitComments
	diags = p.GitComments.As(ctx, &gc, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	if diags.HasError() {
		return req, diags
	}
	resourceConfig, diags := p.resourceConfig(ctx)
	if diags.HasError() {
		return req, diags
	}
	vercelAuthentication, diags := p.vercelAuthentication(ctx)
	if diags.HasError() {
		return req, diags
	}
	return client.UpdateProjectRequest{
		BuildCommand:                         p.BuildCommand.ValueStringPointer(),
		CommandForIgnoringBuildStep:          p.IgnoreCommand.ValueStringPointer(),
		DevCommand:                           p.DevCommand.ValueStringPointer(),
		Framework:                            p.Framework.ValueStringPointer(),
		InstallCommand:                       p.InstallCommand.ValueStringPointer(),
		Name:                                 name,
		OutputDirectory:                      p.OutputDirectory.ValueStringPointer(),
		PublicSource:                         p.PublicSource.ValueBoolPointer(),
		RootDirectory:                        p.RootDirectory.ValueStringPointer(),
		PasswordProtection:                   p.PasswordProtection.toUpdateProjectRequest(),
		VercelAuthentication:                 vercelAuthentication.toVercelAuthentication(),
		TrustedIps:                           p.TrustedIps.toUpdateProjectRequest(),
		OIDCTokenConfig:                      p.OIDCTokenConfig.toUpdateProjectRequest(),
		OptionsAllowlist:                     p.OptionsAllowlist.toUpdateProjectRequest(),
		AutoExposeSystemEnvVars:              p.AutoExposeSystemEnvVars.ValueBool(),
		EnablePreviewFeedback:                oneBoolPointer(p.EnablePreviewFeedback, p.PreviewComments),
		EnableProductionFeedback:             p.EnableProductionFeedback.ValueBoolPointer(),
		EnableAffectedProjectsDeployments:    p.EnableAffectedProjectsDeployments.ValueBoolPointer(),
		PreviewDeploymentsDisabled:           p.PreviewDeploymentsDisabled.ValueBool(),
		AutoAssignCustomDomains:              p.AutoAssignCustomDomains.ValueBool(),
		GitLFS:                               p.GitLFS.ValueBool(),
		ServerlessFunctionZeroConfigFailover: p.FunctionFailover.ValueBool(),
		CustomerSupportCodeVisibility:        p.CustomerSuccessCodeVisibility.ValueBool(),
		GitForkProtection:                    p.GitForkProtection.ValueBool(),
		ProductionDeploymentsFastLane:        p.PrioritiseProductionBuilds.ValueBool(),
		DirectoryListing:                     p.DirectoryListing.ValueBool(),
		SkewProtectionMaxAge:                 toSkewProtectionAge(p.SkewProtection),
		GitComments:                          gc.toUpdateProjectRequest(),
		ResourceConfig:                       resourceConfig.toClientResourceConfig(ctx, p.OnDemandConcurrentBuilds, p.BuildMachineType, p.ServerlessFunctionRegion),
		NodeVersion:                          p.NodeVersion.ValueString(),
	}, nil
}

// EnvironmentItem reflects the state terraform stores internally for a project's environment variable.
type EnvironmentItem struct {
	Target               types.Set    `tfsdk:"target"`
	CustomEnvironmentIDs types.Set    `tfsdk:"custom_environment_ids"`
	GitBranch            types.String `tfsdk:"git_branch"`
	Key                  types.String `tfsdk:"key"`
	Value                types.String `tfsdk:"value"`
	ID                   types.String `tfsdk:"id"`
	Sensitive            types.Bool   `tfsdk:"sensitive"`
	Comment              types.String `tfsdk:"comment"`
}

func (e *EnvironmentItem) equal(other *EnvironmentItem) bool {
	return e.Key.ValueString() == other.Key.ValueString() &&
		e.Value.ValueString() == other.Value.ValueString() &&
		e.Target.Equal(other.Target) &&
		e.CustomEnvironmentIDs.Equal(other.CustomEnvironmentIDs) &&
		e.GitBranch.ValueString() == other.GitBranch.ValueString() &&
		e.Sensitive.ValueBool() == other.Sensitive.ValueBool() &&
		e.Comment.ValueString() == other.Comment.ValueString()
}

func (e *EnvironmentItem) toAttrValue() attr.Value {
	return types.ObjectValueMust(envVariableElemType.AttrTypes, map[string]attr.Value{
		"id":                     e.ID,
		"key":                    e.Key,
		"value":                  e.Value,
		"target":                 e.Target,
		"custom_environment_ids": e.CustomEnvironmentIDs,
		"git_branch":             e.GitBranch,
		"sensitive":              e.Sensitive,
		"comment":                e.Comment,
	})
}

func (e *EnvironmentItem) toEnvironmentVariableRequest(ctx context.Context) (req client.EnvironmentVariableRequest, diags diag.Diagnostics) {
	var target []string
	diags = e.Target.ElementsAs(ctx, &target, true)
	if diags.HasError() {
		return req, diags
	}
	var customEnvironmentIDs []string
	diags = e.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
	if diags.HasError() {
		return req, diags
	}

	var envVariableType string
	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.EnvironmentVariableRequest{
		Key:                  e.Key.ValueString(),
		Value:                e.Value.ValueString(),
		Target:               target,
		CustomEnvironmentIDs: customEnvironmentIDs,
		GitBranch:            e.GitBranch.ValueStringPointer(),
		Type:                 envVariableType,
		Comment:              e.Comment.ValueString(),
	}, nil
}

type DeployHook struct {
	Name types.String `tfsdk:"name"`
	Ref  types.String `tfsdk:"ref"`
	URL  types.String `tfsdk:"url"`
	ID   types.String `tfsdk:"id"`
}

// GitRepository reflects the state terraform stores internally for a nested git_repository block on a project resource.
type GitRepository struct {
	Type             types.String `tfsdk:"type"`
	Repo             types.String `tfsdk:"repo"`
	ProductionBranch types.String `tfsdk:"production_branch"`
	DeployHooks      types.Set    `tfsdk:"deploy_hooks"`
}

func (g *GitRepository) isDifferentRepo(other *GitRepository) bool {
	if g == nil && other == nil {
		return false
	}

	if g == nil || other == nil {
		return true
	}

	return g.Repo.ValueString() != other.Repo.ValueString() || g.Type.ValueString() != other.Type.ValueString()
}

func (g *GitRepository) toCreateProjectRequest() *client.GitRepository {
	if g == nil {
		return nil
	}
	return &client.GitRepository{
		Type: g.Type.ValueString(),
		Repo: g.Repo.ValueString(),
	}
}

func toApiDeploymentProtectionType(dt types.String) string {
	switch dt {
	case types.StringValue("standard_protection"):
		return "prod_deployment_urls_and_all_previews"
	case types.StringValue("standard_protection_new"):
		return "all_except_custom_domains"
	case types.StringValue("all_deployments"):
		return "all"
	case types.StringValue("only_preview_deployments"):
		return "preview"
	case types.StringValue("only_production_deployments"):
		return "production"
	default:
		return dt.ValueString()
	}
}

func fromApiDeploymentProtectionType(dt string) types.String {
	switch dt {
	case "prod_deployment_urls_and_all_previews":
		return types.StringValue("standard_protection")
	case "all_except_custom_domains":
		return types.StringValue("standard_protection_new")
	case "all":
		return types.StringValue("all_deployments")
	case "preview":
		return types.StringValue("only_preview_deployments")
	case "production":
		return types.StringValue("only_production_deployments")
	default:
		return types.StringValue(dt)
	}
}

func (p *PasswordProtectionWithPassword) toUpdateProjectRequest() *client.PasswordProtectionWithPassword {
	if p == nil {
		return nil
	}

	return &client.PasswordProtectionWithPassword{
		DeploymentType: toApiDeploymentProtectionType(p.DeploymentType),
		Password:       p.Password.ValueString(),
	}
}

func toApiTrustedIpProtectionMode(dt types.String) string {
	switch dt {
	case types.StringValue("trusted_ip_required"):
		return "additional"
	case types.StringValue("trusted_ip_optional"):
		return "exclusive"
	default:
		return dt.ValueString()
	}
}

func fromApiTrustedIpProtectionMode(dt string) types.String {
	switch dt {
	case "additional":
		return types.StringValue("trusted_ip_required")
	case "exclusive":
		return types.StringValue("trusted_ip_optional")
	default:
		return types.StringValue(dt)
	}
}

func (t *TrustedIps) toUpdateProjectRequest() *client.TrustedIps {
	if t == nil {
		return nil
	}

	var addresses = []client.TrustedIpAddress{}
	for _, address := range t.Addresses {
		addresses = append(addresses, client.TrustedIpAddress{
			Value: address.Value.ValueString(),
			Note:  address.Note.ValueStringPointer(),
		})
	}

	return &client.TrustedIps{
		Addresses:      addresses,
		DeploymentType: toApiDeploymentProtectionType(t.DeploymentType),
		ProtectionMode: toApiTrustedIpProtectionMode(t.ProtectionMode),
	}
}

type OIDCTokenConfig struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	IssuerMode types.String `tfsdk:"issuer_mode"`
}

func (o *OIDCTokenConfig) toCreateProjectRequest() *client.OIDCTokenConfig {
	if o == nil {
		return nil
	}

	// If the block is present but `enabled` is unspecified, default to true
	var enabled *bool
	if o.Enabled.IsUnknown() || o.Enabled.IsNull() {
		v := true
		enabled = &v
	} else {
		enabled = o.Enabled.ValueBoolPointer()
	}

	return &client.OIDCTokenConfig{
		Enabled:    enabled,
		IssuerMode: o.IssuerMode.ValueString(),
	}
}

func (o *OIDCTokenConfig) toUpdateProjectRequest() *client.OIDCTokenConfig {
	if o == nil {
		// No block provided; do not update OIDC token config
		return nil
	}

	// If the block is present but `enabled` is unspecified, default to true
	var enabled *bool
	if o.Enabled.IsUnknown() || o.Enabled.IsNull() {
		v := true
		enabled = &v
	} else {
		enabled = o.Enabled.ValueBoolPointer()
	}

	return &client.OIDCTokenConfig{
		Enabled:    enabled,
		IssuerMode: o.IssuerMode.ValueString(),
	}
}

var resourceConfigAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"function_default_cpu_type": types.StringType,
		"function_default_timeout":  types.Int64Type,
		"function_default_regions":  types.SetType{ElemType: types.StringType},
		"fluid":                     types.BoolType,
	},
}

var vercelAuthenticationAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"deployment_type": types.StringType,
	},
}

type ResourceConfig struct {
	FunctionDefaultCPUType types.String `tfsdk:"function_default_cpu_type"`
	FunctionDefaultTimeout types.Int64  `tfsdk:"function_default_timeout"`
	FunctionDefaultRegions types.Set    `tfsdk:"function_default_regions"`
	Fluid                  types.Bool   `tfsdk:"fluid"`
}

func (p *Project) resourceConfig(ctx context.Context) (rc *ResourceConfig, diags diag.Diagnostics) {
	diags = p.ResourceConfig.As(ctx, &rc, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	return rc, diags
}

func (p *Project) vercelAuthentication(ctx context.Context) (va *VercelAuthentication, diags diag.Diagnostics) {
	diags = p.VercelAuthentication.As(ctx, &va, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	return va, diags
}

func (v *VercelAuthentication) toVercelAuthentication() *client.VercelAuthentication {
	if v == nil {
		return &client.VercelAuthentication{
			DeploymentType: toApiDeploymentProtectionType(types.StringValue("standard_protection_new")),
		}
	}

	return &client.VercelAuthentication{
		DeploymentType: toApiDeploymentProtectionType(v.DeploymentType),
	}
}

func (r *ResourceConfig) toClientResourceConfig(ctx context.Context, onDemandConcurrentBuilds types.Bool, buildMachineType types.String, serverlessFunctionRegion types.String) *client.ResourceConfig {
	var resourceConfig *client.ResourceConfig = nil
	if r != nil {
		resourceConfig = &client.ResourceConfig{}
	}
	if r != nil && !r.FunctionDefaultCPUType.IsUnknown() {
		resourceConfig.FunctionDefaultMemoryType = r.FunctionDefaultCPUType.ValueStringPointer()
	}
	if r != nil && !r.FunctionDefaultTimeout.IsUnknown() {
		resourceConfig.FunctionDefaultTimeout = r.FunctionDefaultTimeout.ValueInt64Pointer()
	}
	if r != nil && !r.FunctionDefaultRegions.IsUnknown() && !r.FunctionDefaultRegions.IsNull() {
		var regions []string
		r.FunctionDefaultRegions.ElementsAs(ctx, &regions, false)
		resourceConfig.FunctionDefaultRegions = regions
	} else if !serverlessFunctionRegion.IsUnknown() && !serverlessFunctionRegion.IsNull() {
		if resourceConfig == nil {
			resourceConfig = &client.ResourceConfig{}
		}
		resourceConfig.FunctionDefaultRegions = []string{serverlessFunctionRegion.ValueString()}
	}
	if r != nil && !r.Fluid.IsUnknown() {
		resourceConfig.Fluid = r.Fluid.ValueBoolPointer()
	}
	if !onDemandConcurrentBuilds.IsUnknown() {
		if resourceConfig == nil {
			resourceConfig = &client.ResourceConfig{}
		}
		resourceConfig.ElasticConcurrencyEnabled = onDemandConcurrentBuilds.ValueBoolPointer()
	}
	if !buildMachineType.IsUnknown() {
		if resourceConfig == nil {
			resourceConfig = &client.ResourceConfig{}
		}
		resourceConfig.BuildMachineType = buildMachineType.ValueStringPointer()
	}
	return resourceConfig
}

func (t *OptionsAllowlist) toUpdateProjectRequest() *client.OptionsAllowlist {
	if t == nil {
		return nil
	}

	var paths = []client.OptionsAllowlistPath{}
	for _, path := range t.Paths {
		paths = append(paths, client.OptionsAllowlistPath{
			Value: path.Value.ValueString(),
		})
	}

	return &client.OptionsAllowlist{
		Paths: paths,
	}
}

/*
* In the Vercel API the following fields are coerced to null during project creation

* This causes an issue when they are specified, but falsy, as the
* terraform configuration explicitly sets a value for them, but the Vercel
* API returns a different value. This causes an inconsistent plan error.

* We avoid this issue by choosing to use values from the terraform state,
* but only if they are _explicitly stated_ *and* they are _falsy_ values
* *and* the response value was null. This is important as drift detection
* would fail to work if the value was always selected, so this is as stringent
* as possible to allow drift-detection in the majority of scenarios.

* This is implemented in the below uncoerceString and uncoerceBool functions.
 */
type projectCoercedFields struct {
	BuildCommand                      types.String
	DevCommand                        types.String
	InstallCommand                    types.String
	OutputDirectory                   types.String
	PublicSource                      types.Bool
	EnableAffectedProjectsDeployments types.Bool
}

func (p *Project) coercedFields() projectCoercedFields {
	return projectCoercedFields{
		BuildCommand:                      p.BuildCommand,
		DevCommand:                        p.DevCommand,
		InstallCommand:                    p.InstallCommand,
		OutputDirectory:                   p.OutputDirectory,
		PublicSource:                      p.PublicSource,
		EnableAffectedProjectsDeployments: p.EnableAffectedProjectsDeployments,
	}
}

func uncoerceString(plan, res types.String) types.String {
	if plan.ValueString() == "" && !plan.IsNull() && res.IsNull() {
		return plan
	}
	return res
}
func uncoerceBool(plan, res types.Bool) types.Bool {
	if !plan.ValueBool() && !plan.IsNull() && res.IsNull() {
		return plan
	}
	return res
}

var envVariableElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
		"target": types.SetType{
			ElemType: types.StringType,
		},
		"custom_environment_ids": types.SetType{
			ElemType: types.StringType,
		},
		"git_branch": types.StringType,
		"id":         types.StringType,
		"sensitive":  types.BoolType,
		"comment":    types.StringType,
	},
}

var gitCommentsAttrTypes = map[string]attr.Type{
	"on_commit":       types.BoolType,
	"on_pull_request": types.BoolType,
}

func isSameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range a {
		if !contains(b, v) {
			return false
		}
	}
	return true
}

var deployHookType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name": types.StringType,
		"ref":  types.StringType,
		"url":  types.StringType,
		"id":   types.StringType,
	},
}

type deployHook struct {
	Name string `tfsdk:"name"`
	Ref  string `tfsdk:"ref"`
	URL  string `tfsdk:"url"`
	ID   string `tfsdk:"id"`
}

func fromSkewProtectionMaxAge(sp int) types.String {
	if sp == 0 {
		return types.StringNull()
	}
	var ages = map[int]string{
		1800:   "30 minutes",
		43200:  "12 hours",
		86400:  "1 day",
		604800: "7 days",
	}
	v, ok := ages[sp]
	if !ok {
		return types.StringValue(fmt.Sprintf("unknown - %d seconds", sp))
	}
	return types.StringValue(v)
}

func convertResponseToProject(ctx context.Context, response client.ProjectResponse, plan Project, environmentVariables []client.EnvironmentVariable) (Project, error) {
	fields := plan.coercedFields()

	var gr *GitRepository
	if repo := response.Repository(); repo != nil {
		gr = &GitRepository{
			Type:             types.StringValue(repo.Type),
			Repo:             types.StringValue(repo.Repo),
			ProductionBranch: types.StringNull(),
			DeployHooks:      types.SetNull(deployHookType),
		}
		if repo.ProductionBranch != nil {
			gr.ProductionBranch = types.StringValue(*repo.ProductionBranch)
		}
		if repo.DeployHooks != nil && plan.GitRepository != nil && !plan.GitRepository.DeployHooks.IsNull() {
			var dh []deployHook
			for _, h := range repo.DeployHooks {
				dh = append(dh, deployHook{
					Name: h.Name,
					Ref:  h.Ref,
					URL:  h.URL,
					ID:   h.ID,
				})
			}
			hooks, diags := types.SetValueFrom(ctx, deployHookType, dh)
			if diags.HasError() {
				return Project{}, fmt.Errorf("error reading project deploy hooks: %s - %s", diags[0].Summary(), diags[0].Detail())
			}
			gr.DeployHooks = hooks
		}
	}

	var pp *PasswordProtectionWithPassword
	if response.PasswordProtection != nil {
		pass := types.StringValue("")
		if plan.PasswordProtection != nil {
			pass = plan.PasswordProtection.Password
		}
		pp = &PasswordProtectionWithPassword{
			Password:       pass,
			DeploymentType: fromApiDeploymentProtectionType(response.PasswordProtection.DeploymentType),
		}
	}

	va := types.ObjectValueMust(vercelAuthenticationAttrType.AttrTypes, map[string]attr.Value{
		"deployment_type": types.StringValue("none"),
	})
	if response.VercelAuthentication != nil {
		va = types.ObjectValueMust(vercelAuthenticationAttrType.AttrTypes, map[string]attr.Value{
			"deployment_type": fromApiDeploymentProtectionType(response.VercelAuthentication.DeploymentType),
		})
	}

	var tip *TrustedIps
	if response.TrustedIps != nil {
		var addresses []TrustedIpAddress
		for _, address := range response.TrustedIps.Addresses {
			var note = types.StringNull()
			if address.Note != nil {
				note = types.StringValue(*address.Note)
			}
			addresses = append(addresses, TrustedIpAddress{
				Value: types.StringValue(address.Value),
				Note:  note,
			})
		}
		tip = &TrustedIps{
			DeploymentType: fromApiDeploymentProtectionType(response.TrustedIps.DeploymentType),
			Addresses:      addresses,
			ProtectionMode: fromApiTrustedIpProtectionMode(response.TrustedIps.ProtectionMode),
		}
	}

	var oidcTokenConfig = &OIDCTokenConfig{
		Enabled:    types.BoolValue(true),
		IssuerMode: types.StringValue("team"),
	}
	if response.OIDCTokenConfig != nil {
		if response.OIDCTokenConfig.Enabled == nil {
			oidcTokenConfig.Enabled = types.BoolValue(true)
		} else {
			oidcTokenConfig.Enabled = types.BoolPointerValue(response.OIDCTokenConfig.Enabled)
		}
		oidcTokenConfig.IssuerMode = types.StringValue(response.OIDCTokenConfig.IssuerMode)
	}

	resourceConfig := types.ObjectNull(resourceConfigAttrType.AttrTypes)
	if response.ResourceConfig != nil {
		var functionDefaultRegions attr.Value
		regions := make([]attr.Value, 0, len(response.ResourceConfig.FunctionDefaultRegions))
		for _, region := range response.ResourceConfig.FunctionDefaultRegions {
			regions = append(regions, types.StringValue(region))
		}
		functionDefaultRegions = types.SetValueMust(types.StringType, regions)

		resourceConfig = types.ObjectValueMust(resourceConfigAttrType.AttrTypes, map[string]attr.Value{
			"function_default_cpu_type": types.StringPointerValue(response.ResourceConfig.FunctionDefaultMemoryType),
			"function_default_timeout":  types.Int64PointerValue(response.ResourceConfig.FunctionDefaultTimeout),
			"function_default_regions":  functionDefaultRegions,
			"fluid":                     types.BoolValue(response.ResourceConfig.Fluid),
		})
	}

	var oal *OptionsAllowlist
	if response.OptionsAllowlist != nil {
		var paths []OptionsAllowlistPath
		for _, path := range response.OptionsAllowlist.Paths {
			paths = append(paths, OptionsAllowlistPath{
				Value: types.StringValue(path.Value),
			})
		}
		oal = &OptionsAllowlist{
			Paths: paths,
		}
	}

	var env []attr.Value
	for _, e := range environmentVariables {
		var targetValue attr.Value
		if len(e.Target) > 0 {
			target := make([]attr.Value, 0, len(e.Target))
			for _, t := range e.Target {
				target = append(target, types.StringValue(t))
			}
			targetValue = types.SetValueMust(types.StringType, target)
		} else {
			targetValue = types.SetNull(types.StringType)
		}

		var customEnvIDsValue attr.Value
		if len(e.CustomEnvironmentIDs) > 0 {
			customEnvIDs := make([]attr.Value, 0, len(e.CustomEnvironmentIDs))
			for _, c := range e.CustomEnvironmentIDs {
				customEnvIDs = append(customEnvIDs, types.StringValue(c))
			}
			customEnvIDsValue = types.SetValueMust(types.StringType, customEnvIDs)
		} else {
			customEnvIDsValue = types.SetNull(types.StringType)
		}
		value := types.StringValue(e.Value)
		if e.Type == "sensitive" {
			value = types.StringNull()
			environment, err := plan.environment(ctx)
			if err != nil {
				return Project{}, fmt.Errorf("error reading project environment variables: %s", err)
			}
			for _, p := range environment {
				var target []string
				diags := p.Target.ElementsAs(ctx, &target, true)
				if diags.HasError() {
					return Project{}, fmt.Errorf("error reading project environment variables: %s", diags)
				}
				var customEnvironmentIDs []string
				diags = p.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
				if diags.HasError() {
					return Project{}, fmt.Errorf("error reading project environment variables: %s", diags)
				}
				if p.Key.ValueString() == e.Key && isSameStringSet(target, e.Target) && isSameStringSet(customEnvironmentIDs, e.CustomEnvironmentIDs) {
					value = p.Value
					break
				}
			}
		}

		env = append(env, types.ObjectValueMust(envVariableElemType.AttrTypes, map[string]attr.Value{
			"key":                    types.StringValue(e.Key),
			"value":                  value,
			"target":                 targetValue,
			"custom_environment_ids": customEnvIDsValue,
			"git_branch":             types.StringPointerValue(e.GitBranch),
			"id":                     types.StringValue(e.ID),
			"sensitive":              types.BoolValue(e.Type == "sensitive"),
			"comment":                types.StringValue(e.Comment),
		}))
	}

	protectionBypassSecret := types.StringNull()
	protectionBypass := types.BoolNull()
	for k, v := range response.ProtectionBypass {
		if v.Scope == "automation-bypass" {
			protectionBypass = types.BoolValue(true)
			protectionBypassSecret = types.StringValue(k)
			break
		}
	}
	if !plan.ProtectionBypassForAutomation.IsNull() && !plan.ProtectionBypassForAutomation.ValueBool() {
		protectionBypass = types.BoolValue(false)
	}
	if plan.ProtectionBypassForAutomationSecret.ValueString() != "" {
		protectionBypassSecret = types.StringValue(plan.ProtectionBypassForAutomationSecret.ValueString())
	}

	environmentEntry := types.SetValueMust(envVariableElemType, env)
	if plan.Environment.IsNull() {
		environmentEntry = types.SetNull(envVariableElemType)
	}

	gitComments := types.ObjectNull(gitCommentsAttrTypes)
	if response.GitComments != nil && !plan.GitComments.IsNull() {
		var diags diag.Diagnostics
		gitComments, diags = types.ObjectValueFrom(ctx, gitCommentsAttrTypes, &GitComments{
			OnPullRequest: types.BoolValue(response.GitComments.OnPullRequest),
			OnCommit:      types.BoolValue(response.GitComments.OnCommit),
		})
		if diags.HasError() {
			return Project{}, fmt.Errorf("error reading project git comments: %s - %s", diags[0].Summary(), diags[0].Detail())
		}
	}

	return Project{
		BuildCommand:                        uncoerceString(fields.BuildCommand, types.StringPointerValue(response.BuildCommand)),
		DevCommand:                          uncoerceString(fields.DevCommand, types.StringPointerValue(response.DevCommand)),
		Environment:                         environmentEntry,
		Framework:                           types.StringPointerValue(response.Framework),
		GitRepository:                       gr,
		ID:                                  types.StringValue(response.ID),
		IgnoreCommand:                       types.StringPointerValue(response.CommandForIgnoringBuildStep),
		InstallCommand:                      uncoerceString(fields.InstallCommand, types.StringPointerValue(response.InstallCommand)),
		Name:                                types.StringValue(response.Name),
		OutputDirectory:                     uncoerceString(fields.OutputDirectory, types.StringPointerValue(response.OutputDirectory)),
		PublicSource:                        uncoerceBool(fields.PublicSource, types.BoolPointerValue(response.PublicSource)),
		RootDirectory:                       types.StringPointerValue(response.RootDirectory),
		ServerlessFunctionRegion:            types.StringPointerValue(response.ServerlessFunctionRegion),
		TeamID:                              toTeamID(response.TeamID),
		PasswordProtection:                  pp,
		VercelAuthentication:                va,
		TrustedIps:                          tip,
		OIDCTokenConfig:                     oidcTokenConfig,
		OptionsAllowlist:                    oal,
		ProtectionBypassForAutomation:       protectionBypass,
		ProtectionBypassForAutomationSecret: protectionBypassSecret,
		AutoExposeSystemEnvVars:             types.BoolPointerValue(response.AutoExposeSystemEnvVars),
		PreviewComments:                     types.BoolPointerValue(response.EnablePreviewFeedback),
		EnablePreviewFeedback:               types.BoolPointerValue(response.EnablePreviewFeedback),
		EnableProductionFeedback:            types.BoolPointerValue(response.EnableProductionFeedback),
		EnableAffectedProjectsDeployments:   uncoerceBool(fields.EnableAffectedProjectsDeployments, types.BoolPointerValue(response.EnableAffectedProjectsDeployments)),
		PreviewDeploymentsDisabled:          types.BoolValue(response.PreviewDeploymentsDisabled),
		AutoAssignCustomDomains:             types.BoolValue(response.AutoAssignCustomDomains),
		GitLFS:                              types.BoolValue(response.GitLFS),
		FunctionFailover:                    types.BoolValue(response.ServerlessFunctionZeroConfigFailover),
		CustomerSuccessCodeVisibility:       types.BoolValue(response.CustomerSupportCodeVisibility),
		GitForkProtection:                   types.BoolValue(response.GitForkProtection),
		PrioritiseProductionBuilds:          types.BoolValue(response.ProductionDeploymentsFastLane),
		DirectoryListing:                    types.BoolValue(response.DirectoryListing),
		SkewProtection:                      fromSkewProtectionMaxAge(response.SkewProtectionMaxAge),
		GitComments:                         gitComments,
		ResourceConfig:                      resourceConfig,
		NodeVersion:                         types.StringValue(response.NodeVersion),
		OnDemandConcurrentBuilds:            types.BoolValue(response.ResourceConfig.ElasticConcurrencyEnabled),
		BuildMachineType:                    types.StringValue(response.ResourceConfig.BuildMachineType),
	}, nil
}

func (r *projectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config Project
	diags := req.Plan.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := config.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	// work out if there are any new env vars that are specifying sensitive = false
	var nonSensitiveEnvVars []path.Path
	for i, e := range environment {
		if e.ID.ValueString() != "" {
			continue
		}
		if e.Sensitive.IsUnknown() || e.Sensitive.IsNull() || e.Sensitive.ValueBool() {
			continue
		}
		nonSensitiveEnvVars = append(
			nonSensitiveEnvVars,
			path.Root("environment").
				AtSetValue(config.Environment.Elements()[i]).
				AtName("sensitive"),
		)
	}

	if len(nonSensitiveEnvVars) == 0 {
		return
	}

	// if sensitive is explicitly set to `false`, then validate that an env var can be created with the given
	// team sensitive environment variable policy.
	team, err := r.client.Team(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating project environment variable",
			"Could not validate project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	if team.SensitiveEnvironmentVariablePolicy == nil || *team.SensitiveEnvironmentVariablePolicy != "on" {
		// the policy isn't enabled
		return
	}

	for _, p := range nonSensitiveEnvVars {
		resp.Diagnostics.AddAttributeError(
			p,
			"Project Invalid",
			"This team has a policy that forces all environment variables to be sensitive. Please remove the `sensitive` field for your environment variables or set the `sensitive` field to `true` in your configuration.",
		)
	}
}

// Create will create a project within Vercel by calling the Vercel API.
// This is called automatically by the provider when a new resource should be created.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Project
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := plan.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	request, diags := plan.toCreateProjectRequest(ctx, environment)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	out, err := r.client.CreateProject(ctx, plan.TeamID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	environmentVariables, err := r.client.GetEnvironmentVariables(ctx, out.ID, out.TeamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	result, err := convertResponseToProject(ctx, out, plan, environmentVariables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting project response to model",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Error(ctx, "created project", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
		"project":    result,
	})
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.GitRepository != nil && !plan.GitRepository.DeployHooks.IsNull() && !plan.GitRepository.DeployHooks.IsUnknown() {
		var hooks []DeployHook
		diags := plan.GitRepository.DeployHooks.ElementsAs(ctx, &hooks, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, hook := range hooks {
			hook, err := r.client.CreateDeployHook(ctx, client.CreateDeployHookRequest{
				ProjectID: result.ID.ValueString(),
				TeamID:    result.TeamID.ValueString(),
				Name:      hook.Name.ValueString(),
				Ref:       hook.Ref.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating deploy hook",
					"Could not create project, unexpected error: "+err.Error(),
				)
				return
			}
			out.Link.DeployHooks = append(out.Link.DeployHooks, hook)
		}
		result, err := convertResponseToProject(ctx, out, plan, environmentVariables)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error converting project response to model",
				"Could not create project, unexpected error: "+err.Error(),
			)
			return
		}
		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Fields that have to be updated after the project is initially created.
	if plan.RequiresUpdateAfterCreation() {
		req, diags := plan.toUpdateProjectRequest(ctx, plan.Name.ValueString())
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		out, err = r.client.UpdateProject(ctx, result.ID.ValueString(), plan.TeamID.ValueString(), req)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project as part of creating project",
				"Could not update project, unexpected error: "+err.Error(),
			)
			return
		}

		result, err = convertResponseToProject(ctx, out, plan, environmentVariables)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error converting project response to model",
				"Could not create project, unexpected error: "+err.Error(),
			)
			return
		}
		tflog.Info(ctx, "updated newly created project", map[string]any{
			"team_id":    result.TeamID.ValueString(),
			"project_id": result.ID.ValueString(),
		})
		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if plan.ProtectionBypassForAutomation.ValueBool() {
		protectionBypassSecret, err := r.client.UpdateProtectionBypassForAutomation(ctx, client.UpdateProtectionBypassForAutomationRequest{
			ProjectID: result.ID.ValueString(),
			TeamID:    result.TeamID.ValueString(),
			NewValue:  true,
			Secret:    plan.ProtectionBypassForAutomationSecret.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding protection bypass for automation",
				"Failed to create project, an error occurred adding Protection Bypass For Automation: "+err.Error(),
			)
			return
		}
		result.ProtectionBypassForAutomationSecret = types.StringValue(protectionBypassSecret)
		result.ProtectionBypassForAutomation = types.BoolValue(true)
		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if plan.GitRepository == nil || plan.GitRepository.ProductionBranch.IsNull() || plan.GitRepository.ProductionBranch.IsUnknown() {
		return
	}

	out, err = r.client.UpdateProductionBranch(ctx, client.UpdateProductionBranchRequest{
		ProjectID: out.ID,
		Branch:    plan.GitRepository.ProductionBranch.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Failed to create project, an error occurred setting the production branch: "+err.Error(),
		)
		return
	}

	result, err = convertResponseToProject(ctx, out, plan, environmentVariables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting project response to model",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Info(ctx, "updated project production branch", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a project from the vercel API and provide terraform with information about it.
// It is called by the provider whenever values should be read to update state.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProject(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	environmentVariables, err := r.client.GetEnvironmentVariables(ctx, out.ID, out.TeamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not read project, unexpected error: "+err.Error(),
		)
		return
	}
	result, err := convertResponseToProject(ctx, out, state, environmentVariables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting project response to model",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Info(ctx, "read project", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// containsEnvVar is a helper function for working out whether a specific environment variable
// is present within a slice. It ensures that all properties of the environment variable match.
func containsEnvVar(env []EnvironmentItem, v EnvironmentItem) bool {
	for _, e := range env {
		if e.ID == v.ID {
			// we can rely on this because we use a known ID from state.
			// and new env vars have no ID
			return true
		}
	}
	return false
}

// diffEnvVars is used to determine the set of environment variables that need to be updated,
// and the set of environment variables that need to be removed.
func diffEnvVars(oldVars, newVars []EnvironmentItem) (toCreate, toRemove []EnvironmentItem) {
	toRemove = []EnvironmentItem{}
	toCreate = []EnvironmentItem{}
	for _, e := range oldVars {
		if !containsEnvVar(newVars, e) {
			toRemove = append(toRemove, e)
		}
	}
	for _, e := range newVars {
		if !containsEnvVar(oldVars, e) {
			toCreate = append(toCreate, e)
		}
	}
	return toCreate, toRemove
}

func containsDeployHook(hooks []DeployHook, h DeployHook) bool {
	for _, hook := range hooks {
		if hook.ID == h.ID {
			return true
		}
	}
	return false
}

func diffDeployHooks(ctx context.Context, new, old *GitRepository) (toCreate, toRemove []DeployHook, diags diag.Diagnostics) {
	if new == nil && old == nil {
		return nil, nil, nil
	}
	if new == nil {
		diags = old.DeployHooks.ElementsAs(ctx, &toRemove, false)
		return nil, toRemove, diags
	}
	if old == nil {
		diags = new.DeployHooks.ElementsAs(ctx, &toCreate, false)
		return toCreate, nil, diags
	}
	var oldHooks []DeployHook
	var newHooks []DeployHook
	diags = old.DeployHooks.ElementsAs(ctx, &oldHooks, false)
	if diags.HasError() {
		return nil, nil, diags
	}
	diags = new.DeployHooks.ElementsAs(ctx, &newHooks, false)
	if diags.HasError() {
		return nil, nil, diags
	}

	for _, h := range oldHooks {
		if !containsDeployHook(newHooks, h) {
			toRemove = append(toRemove, h)
		}
	}
	for _, h := range newHooks {
		if !containsDeployHook(oldHooks, h) {
			toCreate = append(toCreate, h)
		}
	}
	return toCreate, toRemove, diags
}

// Update will update a project and it's associated environment variables via the vercel API.
// Environment variables are manually diffed and updated individually. Once the environment
// variables are all updated, the project is updated too.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Project
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state Project
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	/* Update the environment variables first */
	planEnvs, err := plan.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}
	stateEnvs, err := state.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables from state",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "planEnvs", map[string]any{
		"plan_envs":  planEnvs,
		"state_envs": stateEnvs,
	})

	toCreate, toRemove := diffEnvVars(stateEnvs, planEnvs)
	for _, v := range toRemove {
		err := r.client.DeleteEnvironmentVariable(ctx, state.ID.ValueString(), state.TeamID.ValueString(), v.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.ValueString(),
					v.ID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Info(ctx, "deleted environment variable", map[string]any{
			"team_id":        plan.TeamID.ValueString(),
			"project_id":     plan.ID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}

	var items []client.EnvironmentVariableRequest
	for _, v := range toCreate {
		vv, diags := v.toEnvironmentVariableRequest(ctx)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		items = append(items, vv)
	}

	if items != nil {
		_, err = r.client.CreateEnvironmentVariables(
			ctx,
			client.CreateEnvironmentVariablesRequest{
				ProjectID:            plan.ID.ValueString(),
				TeamID:               plan.TeamID.ValueString(),
				EnvironmentVariables: items,
			},
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not upsert environment variables for project %s, unexpected error: %s",
					plan.ID.ValueString(),
					err,
				),
			)
		}
		tflog.Info(ctx, "upserted environment variables", map[string]any{
			"team_id":    plan.TeamID.ValueString(),
			"project_id": plan.ID.ValueString(),
		})
	}

	if state.ProtectionBypassForAutomation != plan.ProtectionBypassForAutomation {
		secret := state.ProtectionBypassForAutomationSecret.ValueString()
		if plan.ProtectionBypassForAutomationSecret.ValueString() != "" {
			secret = plan.ProtectionBypassForAutomationSecret.ValueString()
		}
		_, err := r.client.UpdateProtectionBypassForAutomation(ctx, client.UpdateProtectionBypassForAutomationRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			NewValue:  plan.ProtectionBypassForAutomation.ValueBool(),
			Secret:    secret,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not update project %s %s, unexpected error setting Protection Bypass For Automation: %s",
					state.TeamID.ValueString(),
					state.ID.ValueString(),
					err,
				),
			)
			return
		}
	}

	updateRequest, diags := plan.toUpdateProjectRequest(ctx, state.Name.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateProject(ctx, state.ID.ValueString(), state.TeamID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			fmt.Sprintf(
				"Could not update project %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	if plan.GitRepository == nil && state.GitRepository != nil {
		out, err = r.client.UnlinkGitRepoFromProject(ctx, plan.ID.ValueString(), plan.TeamID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not update project %s %s, unexpected error: %s",
					state.TeamID.ValueString(),
					state.ID.ValueString(),
					err,
				),
			)
			return
		}
	}

	wasUnlinked := false
	if plan.GitRepository.isDifferentRepo(state.GitRepository) {
		if state.GitRepository != nil {
			_, err = r.client.UnlinkGitRepoFromProject(ctx, plan.ID.ValueString(), plan.TeamID.ValueString())
			wasUnlinked = true
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating project",
					fmt.Sprintf(
						"Could not update project unlinking git repo %s %s, unexpected error: %s",
						state.TeamID.ValueString(),
						state.ID.ValueString(),
						err,
					),
				)
				return
			}
		}

		if plan.GitRepository != nil {
			out, err = r.client.LinkGitRepoToProject(ctx, client.LinkGitRepoToProjectRequest{
				ProjectID: plan.ID.ValueString(),
				TeamID:    plan.TeamID.ValueString(),
				Repo:      plan.GitRepository.Repo.ValueString(),
				Type:      plan.GitRepository.Type.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating project",
					fmt.Sprintf(
						"Could not update project git repo %s %s, unexpected error: %s",
						state.TeamID.ValueString(),
						state.ID.ValueString(),
						err,
					),
				)
				return
			}
		}
	}

	if plan.GitRepository != nil && !plan.GitRepository.ProductionBranch.IsUnknown() &&
		!plan.GitRepository.ProductionBranch.IsNull() && // we know the value the production branch _should_ be
		(wasUnlinked || // and we either unlinked the repo,
			(state.GitRepository == nil || // or the production branch was never set
				// or the production branch was/is something else
				state.GitRepository.ProductionBranch.ValueString() != plan.GitRepository.ProductionBranch.ValueString())) {

		out, err = r.client.UpdateProductionBranch(ctx, client.UpdateProductionBranchRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			Branch:    plan.GitRepository.ProductionBranch.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project",
				fmt.Sprintf(
					"Could not update project production branch %s %s to '%s', unexpected error: %s",
					state.TeamID.ValueString(),
					state.ID.ValueString(),
					plan.GitRepository.ProductionBranch.ValueString(),
					err,
				),
			)
			return
		}
	}

	hooksToCreate, hooksToRemove, diags := diffDeployHooks(ctx, plan.GitRepository, state.GitRepository)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, h := range hooksToRemove {
		err := r.client.DeleteDeployHook(ctx, client.DeleteDeployHookRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			ID:        h.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting deploy hook",
				"Could not update project, unexpected error: "+err.Error(),
			)
			return
		}
	}
	for _, h := range hooksToCreate {
		_, err := r.client.CreateDeployHook(ctx, client.CreateDeployHookRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			Name:      h.Name.ValueString(),
			Ref:       h.Ref.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating deploy hook",
				"Could not update project, unexpected error: "+err.Error(),
			)
			return
		}
	}
	if hooksToCreate != nil || hooksToRemove != nil {
		// Re-fetch the project to ensure the hooks afterwards are all correct
		out, err = r.client.GetProject(ctx, plan.ID.ValueString(), plan.TeamID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading project",
				"Could not update project, unexpected error: "+err.Error(),
			)
			return
		}
	}

	environmentVariables, err := r.client.GetEnvironmentVariables(ctx, out.ID, out.TeamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}
	result, err := convertResponseToProject(ctx, out, plan, environmentVariables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting project response to model",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Info(ctx, "updated project", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete a project and any associated environment variables from within terraform.
// Environment variables do not need to be explicitly deleted, as Vercel will automatically prune them.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Project
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProject(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			fmt.Sprintf(
				"Could not delete project %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted project", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ID.ValueString(),
	})
}

// ImportState takes an identifier and reads all the project information from the Vercel API.
// Note that environment variables are also read. The results are then stored in terraform state.
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetProject(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not get project %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	environmentVariables, err := r.client.GetEnvironmentVariables(ctx, out.ID, out.TeamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not import project, unexpected error: "+err.Error(),
		)
		return
	}
	result, err := convertResponseToProject(ctx, out, nullProject, environmentVariables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting project response to model",
			"Could not import project, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Info(ctx, "imported project", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
