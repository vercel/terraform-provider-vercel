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
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &projectDataSource{}
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
			"automatically_expose_system_environment_variables": schema.BoolAttribute{
				Computed:    true,
				Description: "Vercel provides a set of Environment Variables that are automatically populated by the System, such as the URL of the Deployment or the name of the Git branch deployed. To expose them to your Deployments, enable this field",
			},
		},
	}
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

	out, err := d.client.GetProject(ctx, config.Name.ValueString(), config.TeamID.ValueString(), true)
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

	result, err := convertResponseToProjectDataSource(ctx, out, nullProject)
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
