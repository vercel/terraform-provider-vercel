package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
	"github.com/vercel/terraform-provider-vercel/client"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
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

~> Terraform currently provides both a standalone Project Environment Variable resource (a single Environment Variable), and a Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
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
					stringLengthBetween(1, 52),
					stringRegex(
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
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The region on Vercel's network to which your Serverless Functions are deployed. It should be close to any data source your Serverless Function might depend on. A new Deployment is required for your changes to take effect. Please see [Vercel's documentation](https://vercel.com/docs/concepts/edge-network/regions) for a full list of regions.",
				Validators: []validator.String{
					validateServerlessFunctionRegion(),
				},
			},
			"environment": schema.SetNestedAttribute{
				Description: "A set of Environment Variables that should be configured for the project.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.SetAttribute{
							Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
							ElementType: types.StringType,
							Validators: []validator.Set{
								stringSetItemsIn("production", "preview", "development"),
								stringSetMinCount(1),
							},
							Required: true,
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
							Description: "Whether the Environment Variable is sensitive or not.",
							Optional:    true,
							Computed:    true,
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
							stringOneOf("github", "gitlab", "bitbucket"),
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
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"deployment_type": types.StringType,
					},
					map[string]attr.Value{
						"deployment_type": types.StringValue("standard_protection"),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Required:      true,
						Description:   "The deployment environment to protect. Must be one of `standard_protection`, `all_deployments`, `only_preview_deployments`, or `none`.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringOneOf("standard_protection", "all_deployments", "only_preview_deployments", "none"),
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
							stringLengthBetween(1, 72),
						},
					},
					"deployment_type": schema.StringAttribute{
						Required:      true,
						Description:   "The deployment environment to protect. Must be one of `standard_protection`, `all_deployments`, or `only_preview_deployments`.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringOneOf("standard_protection", "all_deployments", "only_preview_deployments"),
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
							stringSetMinCount(1),
						},
					},
					"deployment_type": schema.StringAttribute{
						Required:      true,
						Description:   "The deployment environment to protect. Must be one of `standard_protection`, `all_deployments`, `only_production_deployments`, or `only_preview_deployments`.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringOneOf("standard_protection", "all_deployments", "only_production_deployments", "only_preview_deployments"),
						},
					},
					"protection_mode": schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						Default:       stringdefault.StaticString("trusted_ip_required"),
						Description:   "Whether or not Trusted IPs is optional to access a deployment. Must be either `trusted_ip_required` or `trusted_ip_optional`. `trusted_ip_optional` is only available with Standalone Trusted IPs.",
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringOneOf("trusted_ip_required", "trusted_ip_optional"),
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
				Description: "Allow automation services to bypass Vercel Authentication and Password Protection for both Preview and Production Deployments on this project when using an HTTP header named `x-vercel-protection-bypass` with a value of the `password_protection_for_automation_secret` field.",
			},
			"protection_bypass_for_automation_secret": schema.StringAttribute{
				Computed:    true,
				Description: "If `protection_bypass_for_automation` is enabled, use this value in the `x-vercel-protection-bypass` header to bypass Vercel Authentication and Password Protection for both Preview and Production Deployments.",
			},
			"automatically_expose_system_environment_variables": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Vercel provides a set of Environment Variables that are automatically populated by the System, such as the URL of the Deployment or the name of the Git branch deployed. To expose them to your Deployments, enable this field",
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
				Optional:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Whether to enable comments on your Preview Deployments. If omitted, comments are controlled at the team level (default behaviour).",
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
					stringOneOf("30 minutes", "12 hours", "1 day", "7 days"),
				},
			},
		},
	}
}

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
	OutputDirectory                     types.String                    `tfsdk:"output_directory"`
	PublicSource                        types.Bool                      `tfsdk:"public_source"`
	RootDirectory                       types.String                    `tfsdk:"root_directory"`
	ServerlessFunctionRegion            types.String                    `tfsdk:"serverless_function_region"`
	TeamID                              types.String                    `tfsdk:"team_id"`
	VercelAuthentication                *VercelAuthentication           `tfsdk:"vercel_authentication"`
	PasswordProtection                  *PasswordProtectionWithPassword `tfsdk:"password_protection"`
	TrustedIps                          *TrustedIps                     `tfsdk:"trusted_ips"`
	ProtectionBypassForAutomation       types.Bool                      `tfsdk:"protection_bypass_for_automation"`
	ProtectionBypassForAutomationSecret types.String                    `tfsdk:"protection_bypass_for_automation_secret"`
	AutoExposeSystemEnvVars             types.Bool                      `tfsdk:"automatically_expose_system_environment_variables"`
	GitComments                         types.Object                    `tfsdk:"git_comments"`
	PreviewComments                     types.Bool                      `tfsdk:"preview_comments"`
	AutoAssignCustomDomains             types.Bool                      `tfsdk:"auto_assign_custom_domains"`
	GitLFS                              types.Bool                      `tfsdk:"git_lfs"`
	FunctionFailover                    types.Bool                      `tfsdk:"function_failover"`
	CustomerSuccessCodeVisibility       types.Bool                      `tfsdk:"customer_success_code_visibility"`
	GitForkProtection                   types.Bool                      `tfsdk:"git_fork_protection"`
	PrioritiseProductionBuilds          types.Bool                      `tfsdk:"prioritise_production_builds"`
	DirectoryListing                    types.Bool                      `tfsdk:"directory_listing"`
	SkewProtection                      types.String                    `tfsdk:"skew_protection"`
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
		p.VercelAuthentication != nil ||
		p.TrustedIps != nil ||
		!p.AutoExposeSystemEnvVars.IsNull() ||
		p.GitComments.IsNull() ||
		!p.PreviewComments.IsNull() ||
		(!p.AutoAssignCustomDomains.IsNull() && !p.AutoAssignCustomDomains.ValueBool()) ||
		!p.GitLFS.IsNull() ||
		!p.FunctionFailover.IsNull() ||
		!p.CustomerSuccessCodeVisibility.IsNull() ||
		(!p.GitForkProtection.IsNull() && !p.GitForkProtection.ValueBool()) ||
		!p.PrioritiseProductionBuilds.IsNull() ||
		!p.DirectoryListing.IsNull() ||
		!p.SkewProtection.IsNull()
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
	if p.Environment.IsNull() {
		return nil, nil
	}

	var vars []EnvironmentItem
	err := p.Environment.ElementsAs(ctx, &vars, true)
	if err != nil {
		return nil, fmt.Errorf("error reading project environment variables: %s", err)
	}
	return vars, nil
}

func parseEnvironment(vars []EnvironmentItem) []client.EnvironmentVariable {
	out := []client.EnvironmentVariable{}
	for _, e := range vars {
		target := []string{}
		for _, t := range e.Target {
			target = append(target, t.ValueString())
		}

		var envVariableType string

		if e.Sensitive.ValueBool() {
			envVariableType = "sensitive"
		} else {
			envVariableType = "encrypted"
		}

		out = append(out, client.EnvironmentVariable{
			Key:       e.Key.ValueString(),
			Value:     e.Value.ValueString(),
			Target:    target,
			GitBranch: e.GitBranch.ValueStringPointer(),
			Type:      envVariableType,
			ID:        e.ID.ValueString(),
		})
	}
	return out
}

func (p *Project) toCreateProjectRequest(envs []EnvironmentItem) client.CreateProjectRequest {
	return client.CreateProjectRequest{
		BuildCommand:                p.BuildCommand.ValueStringPointer(),
		CommandForIgnoringBuildStep: p.IgnoreCommand.ValueStringPointer(),
		DevCommand:                  p.DevCommand.ValueStringPointer(),
		EnvironmentVariables:        parseEnvironment(envs),
		Framework:                   p.Framework.ValueStringPointer(),
		GitRepository:               p.GitRepository.toCreateProjectRequest(),
		InstallCommand:              p.InstallCommand.ValueStringPointer(),
		Name:                        p.Name.ValueString(),
		OutputDirectory:             p.OutputDirectory.ValueStringPointer(),
		PublicSource:                p.PublicSource.ValueBoolPointer(),
		RootDirectory:               p.RootDirectory.ValueStringPointer(),
		ServerlessFunctionRegion:    p.ServerlessFunctionRegion.ValueString(),
	}
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
	return ages[sp.ValueString()]
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
		ServerlessFunctionRegion:             p.ServerlessFunctionRegion.ValueString(),
		PasswordProtection:                   p.PasswordProtection.toUpdateProjectRequest(),
		VercelAuthentication:                 p.VercelAuthentication.toUpdateProjectRequest(),
		TrustedIps:                           p.TrustedIps.toUpdateProjectRequest(),
		AutoExposeSystemEnvVars:              p.AutoExposeSystemEnvVars.ValueBool(),
		EnablePreviewFeedback:                p.PreviewComments.ValueBoolPointer(),
		AutoAssignCustomDomains:              p.AutoAssignCustomDomains.ValueBool(),
		GitLFS:                               p.GitLFS.ValueBool(),
		ServerlessFunctionZeroConfigFailover: p.FunctionFailover.ValueBool(),
		CustomerSupportCodeVisibility:        p.CustomerSuccessCodeVisibility.ValueBool(),
		GitForkProtection:                    p.GitForkProtection.ValueBool(),
		ProductionDeploymentsFastLane:        p.PrioritiseProductionBuilds.ValueBool(),
		DirectoryListing:                     p.DirectoryListing.ValueBool(),
		SkewProtectionMaxAge:                 toSkewProtectionAge(p.SkewProtection),
		GitComments:                          gc.toUpdateProjectRequest(),
	}, nil
}

// EnvironmentItem reflects the state terraform stores internally for a project's environment variable.
type EnvironmentItem struct {
	Target    []types.String `tfsdk:"target"`
	GitBranch types.String   `tfsdk:"git_branch"`
	Key       types.String   `tfsdk:"key"`
	Value     types.String   `tfsdk:"value"`
	ID        types.String   `tfsdk:"id"`
	Sensitive types.Bool     `tfsdk:"sensitive"`
}

func (e *EnvironmentItem) toEnvironmentVariableRequest() client.EnvironmentVariableRequest {
	target := []string{}
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}

	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.EnvironmentVariableRequest{
		Key:       e.Key.ValueString(),
		Value:     e.Value.ValueString(),
		Target:    target,
		GitBranch: e.GitBranch.ValueStringPointer(),
		Type:      envVariableType,
	}
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

func (v *VercelAuthentication) toUpdateProjectRequest() *client.VercelAuthentication {
	if v == nil {
		return nil
	}

	return &client.VercelAuthentication{
		DeploymentType: toApiDeploymentProtectionType(v.DeploymentType),
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
			Note:  address.Note.ValueString(),
		})
	}

	return &client.TrustedIps{
		Addresses:      addresses,
		DeploymentType: toApiDeploymentProtectionType(t.DeploymentType),
		ProtectionMode: toApiTrustedIpProtectionMode(t.ProtectionMode),
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
	BuildCommand    types.String
	DevCommand      types.String
	InstallCommand  types.String
	OutputDirectory types.String
	PublicSource    types.Bool
}

func (p *Project) coercedFields() projectCoercedFields {
	return projectCoercedFields{
		BuildCommand:    p.BuildCommand,
		DevCommand:      p.DevCommand,
		InstallCommand:  p.InstallCommand,
		OutputDirectory: p.OutputDirectory,
		PublicSource:    p.PublicSource,
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
		"git_branch": types.StringType,
		"id":         types.StringType,
		"sensitive":  types.BoolType,
	},
}

var gitCommentsAttrTypes = map[string]attr.Type{
	"on_commit":       types.BoolType,
	"on_pull_request": types.BoolType,
}

func hasSameTarget(p EnvironmentItem, target []string) bool {
	if len(p.Target) != len(target) {
		return false
	}
	for _, t := range p.Target {
		v := t.ValueString()
		if !contains(target, v) {
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

	var va = &VercelAuthentication{
		DeploymentType: types.StringValue("none"),
	}
	if response.VercelAuthentication != nil {
		va = &VercelAuthentication{
			DeploymentType: fromApiDeploymentProtectionType(response.VercelAuthentication.DeploymentType),
		}
	}

	var tip *TrustedIps
	if response.TrustedIps != nil {
		var addresses []TrustedIpAddress
		for _, address := range response.TrustedIps.Addresses {
			addresses = append(addresses, TrustedIpAddress{
				Value: types.StringValue(address.Value),
				Note:  types.StringValue(address.Note),
			})
		}
		tip = &TrustedIps{
			DeploymentType: fromApiDeploymentProtectionType(response.TrustedIps.DeploymentType),
			Addresses:      addresses,
			ProtectionMode: fromApiTrustedIpProtectionMode(response.TrustedIps.ProtectionMode),
		}
	}

	var env []attr.Value
	for _, e := range environmentVariables {
		target := []attr.Value{}
		for _, t := range e.Target {
			target = append(target, types.StringValue(t))
		}
		value := types.StringValue(e.Value)
		if e.Type == "sensitive" {
			value = types.StringNull()
			environment, err := plan.environment(ctx)
			if err != nil {
				return Project{}, fmt.Errorf("error reading project environment variables: %s", err)
			}
			for _, p := range environment {
				if p.Sensitive.ValueBool() && p.Key.ValueString() == e.Key && hasSameTarget(p, e.Target) {
					value = p.Value
					break
				}
			}
		}

		env = append(env, types.ObjectValueMust(
			map[string]attr.Type{
				"key":   types.StringType,
				"value": types.StringType,
				"target": types.SetType{
					ElemType: types.StringType,
				},
				"git_branch": types.StringType,
				"id":         types.StringType,
				"sensitive":  types.BoolType,
			},
			map[string]attr.Value{
				"key":        types.StringValue(e.Key),
				"value":      value,
				"target":     types.SetValueMust(types.StringType, target),
				"git_branch": types.StringPointerValue(e.GitBranch),
				"id":         types.StringValue(e.ID),
				"sensitive":  types.BoolValue(e.Type == "sensitive"),
			},
		))
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
		ProtectionBypassForAutomation:       protectionBypass,
		ProtectionBypassForAutomationSecret: protectionBypassSecret,
		AutoExposeSystemEnvVars:             types.BoolPointerValue(response.AutoExposeSystemEnvVars),
		PreviewComments:                     types.BoolPointerValue(response.EnablePreviewFeedback),
		AutoAssignCustomDomains:             types.BoolValue(response.AutoAssignCustomDomains),
		GitLFS:                              types.BoolValue(response.GitLFS),
		FunctionFailover:                    types.BoolValue(response.ServerlessFunctionZeroConfigFailover),
		CustomerSuccessCodeVisibility:       types.BoolValue(response.CustomerSupportCodeVisibility),
		GitForkProtection:                   types.BoolValue(response.GitForkProtection),
		PrioritiseProductionBuilds:          types.BoolValue(response.ProductionDeploymentsFastLane),
		DirectoryListing:                    types.BoolValue(response.DirectoryListing),
		SkewProtection:                      fromSkewProtectionMaxAge(response.SkewProtectionMaxAge),
		GitComments:                         gitComments,
	}, nil
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

	out, err := r.client.CreateProject(ctx, plan.TeamID.ValueString(), plan.toCreateProjectRequest(environment))
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
	tflog.Info(ctx, "created project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
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
		tflog.Info(ctx, "updated newly created project", map[string]interface{}{
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
	tflog.Info(ctx, "updated project production branch", map[string]interface{}{
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

	tflog.Info(ctx, "planEnvs", map[string]interface{}{
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
		tflog.Info(ctx, "deleted environment variable", map[string]interface{}{
			"team_id":        plan.TeamID.ValueString(),
			"project_id":     plan.ID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}

	var items []client.EnvironmentVariableRequest
	for _, v := range toCreate {
		items = append(items, v.toEnvironmentVariableRequest())
	}

	if items != nil {
		err = r.client.CreateEnvironmentVariables(
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
		tflog.Info(ctx, "upserted environment variables", map[string]interface{}{
			"team_id":    plan.TeamID.ValueString(),
			"project_id": plan.ID.ValueString(),
		})
	}

	if state.ProtectionBypassForAutomation != plan.ProtectionBypassForAutomation {
		_, err := r.client.UpdateProtectionBypassForAutomation(ctx, client.UpdateProtectionBypassForAutomationRequest{
			ProjectID: plan.ID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			NewValue:  plan.ProtectionBypassForAutomation.ValueBool(),
			Secret:    state.ProtectionBypassForAutomationSecret.ValueString(),
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
	tflog.Info(ctx, "updated project", map[string]interface{}{
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

	tflog.Info(ctx, "deleted project", map[string]interface{}{
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
	tflog.Info(ctx, "imported project", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
