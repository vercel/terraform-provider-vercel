package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ datasource.DataSource              = &projectRoutesDataSource{}
	_ datasource.DataSourceWithConfigure = &projectRoutesDataSource{}
)

func newProjectRoutesDataSource() datasource.DataSource {
	return &projectRoutesDataSource{}
}

type projectRoutesDataSource struct {
	client *client.Client
}

func (d *projectRoutesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_routes"
}

func (d *projectRoutesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectRoutesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about the live project routing rules configured for a Vercel project.

This data source intentionally reads the current live version and ignores unpublished staged drafts.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this data source. Format: team_id/project_id or project_id for personal accounts.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the project to read routing rules for.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"rules": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The ordered list of live routing rules for the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The rule ID managed by Vercel.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "A human-readable name for the rule.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "An optional description of the rule.",
						},
						"enabled": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the rule is enabled.",
						},
						"src_syntax": schema.StringAttribute{
							Computed:    true,
							Description: "The source pattern syntax inferred or stored by Vercel.",
							Validators: []validator.String{
								stringvalidator.OneOf("equals", "path-to-regexp", "regex"),
							},
						},
						"route_type": schema.StringAttribute{
							Computed:    true,
							Description: "The computed route type returned by Vercel. One of `rewrite`, `redirect`, `set_status`, or `transform`.",
						},
						"route": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "The routing rule definition.",
							Attributes: map[string]schema.Attribute{
								"src": schema.StringAttribute{
									Computed:    true,
									Description: "The source pattern to match.",
								},
								"dest": schema.StringAttribute{
									Computed:    true,
									Description: "The destination for rewrites or redirects.",
								},
								"headers": schema.MapAttribute{
									Computed:    true,
									ElementType: types.StringType,
									Description: "Headers to set for the matched request.",
								},
								"case_sensitive": schema.BoolAttribute{
									Computed:    true,
									Description: "Whether the `src` matcher is case-sensitive.",
								},
								"status": schema.Int64Attribute{
									Computed:    true,
									Description: "The HTTP status code to set for redirects or status-only rules.",
								},
								"has": schema.ListNestedAttribute{
									Computed:    true,
									Description: "Conditions that must be present for the rule to match.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Computed: true,
											},
											"key": schema.StringAttribute{
												Computed: true,
											},
											"value": schema.StringAttribute{
												Computed: true,
											},
										},
									},
								},
								"missing": schema.ListNestedAttribute{
									Computed:    true,
									Description: "Conditions that must be absent for the rule to match.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Computed: true,
											},
											"key": schema.StringAttribute{
												Computed: true,
											},
											"value": schema.StringAttribute{
												Computed: true,
											},
										},
									},
								},
								"transforms": schema.ListNestedAttribute{
									Computed:    true,
									Description: "Transforms applied to the request or response when the rule matches.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Computed: true,
											},
											"op": schema.StringAttribute{
												Computed: true,
											},
											"target": schema.StringAttribute{
												Computed: true,
											},
											"args": schema.StringAttribute{
												Computed: true,
											},
											"env": schema.ListAttribute{
												Computed:    true,
												ElementType: types.StringType,
											},
										},
									},
								},
								"respect_origin_cache_control": schema.BoolAttribute{
									Computed:    true,
									Description: "Whether the rule should respect cache control headers from the origin.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *projectRoutesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectRoutesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, diags, err := readProjectRoutes(ctx, d.client, config.ProjectID.ValueString(), config.TeamID.ValueString(), nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project routes",
			fmt.Sprintf("Could not get project routes %s %s, unexpected error: %s", config.TeamID.ValueString(), config.ProjectID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "read project routes data source", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
