package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &projectsDataSource{}
	_ datasource.DataSourceWithConfigure = &projectsDataSource{}
)

type projectsDataSource struct{ client *client.Client }

type projectSearchConfig struct {
	TeamID                      types.String `tfsdk:"team_id"`
	RepositoryURL               types.String `tfsdk:"repository_url"`
	Framework                   types.String `tfsdk:"framework"`
	PassportEnabled             types.Bool   `tfsdk:"passport_enabled"`
	VercelAuthenticationEnabled types.Bool   `tfsdk:"vercel_authentication_enabled"`
	PasswordProtectionEnabled   types.Bool   `tfsdk:"password_protection_enabled"`
	TrustedIPsEnabled           types.Bool   `tfsdk:"trusted_ips_enabled"`
}

type projectsDataSourceModel struct {
	projectSearchConfig
	Projects map[string]projectSummary `tfsdk:"projects"`
}

type projectProtectionSummary struct {
	Enabled        types.Bool   `tfsdk:"enabled"`
	DeploymentType types.String `tfsdk:"deployment_type"`
}

type projectSummary struct {
	ID                   types.String             `tfsdk:"id"`
	Name                 types.String             `tfsdk:"name"`
	TeamID               types.String             `tfsdk:"team_id"`
	Framework            types.String             `tfsdk:"framework"`
	GitRepository        types.Object             `tfsdk:"git_repository"`
	Passport             projectProtectionSummary `tfsdk:"passport"`
	VercelAuthentication projectProtectionSummary `tfsdk:"vercel_authentication"`
	PasswordProtection   projectProtectionSummary `tfsdk:"password_protection"`
	TrustedIPs           projectProtectionSummary `tfsdk:"trusted_ips"`
}

var projectRepositorySummaryTypes = map[string]attr.Type{
	"type": types.StringType,
	"repo": types.StringType,
}

func newProjectsDataSource() datasource.DataSource { return &projectsDataSource{} }

func (d *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
	}
}

func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := projectSearchDataSourceAttributes()
	attributes["projects"] = schema.MapNestedAttribute{
		Computed:    true,
		Description: "Matching projects keyed by their stable project IDs. Only project metadata and protection settings are returned; environment variables, passwords and bypass secrets are excluded.",
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, Description: "The project ID."},
			"name":      schema.StringAttribute{Computed: true, Description: "The current project name."},
			"team_id":   schema.StringAttribute{Computed: true, Description: "The owning team ID. Null for a personal project."},
			"framework": schema.StringAttribute{Computed: true, Description: "The configured framework. Null when no framework is configured."},
			"git_repository": schema.SingleNestedAttribute{
				Computed: true, Description: "The connected Git repository. Null when no repository is connected.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true, Description: "The Git provider: github, gitlab or bitbucket."},
					"repo": schema.StringAttribute{Computed: true, Description: "The repository path, including its owner or namespace."},
				},
			},
			"passport":              projectProtectionSummarySchema("Passport"),
			"vercel_authentication": projectProtectionSummarySchema("Vercel Authentication"),
			"password_protection":   projectProtectionSummarySchema("Password Protection"),
			"trusted_ips":           projectProtectionSummarySchema("Trusted IPs"),
		}},
	}
	resp.Schema = schema.Schema{
		Description: "Lists all projects visible to the authenticated user in the selected team or personal account. Uses the provider's default team unless team_id is set. All supplied filters must match. Protection filters describe configured settings, not effective access after bypasses or exceptions. Use protection filters for discovery and audits; using them as resource for_each keys can remove resources from configuration after protection changes.",
		Attributes:  attributes,
	}
}

func projectSearchDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"team_id":                       schema.StringAttribute{Optional: true, Computed: true, Description: "The team ID to query. Defaults to the team configured on the provider, or the personal account when no team is configured."},
		"repository_url":                schema.StringAttribute{Optional: true, Description: "Filter by the connected Git repository URL, for example https://github.com/vercel/next.js. Uses the API's repository URL matching."},
		"framework":                     schema.StringAttribute{Optional: true, Description: "Filter by the exact framework slug, for example nextjs. Omit to include all frameworks and projects without a configured framework."},
		"passport_enabled":              schema.BoolAttribute{Optional: true, Description: "Filter by whether Passport is configured. Omit to include both enabled and disabled projects."},
		"vercel_authentication_enabled": schema.BoolAttribute{Optional: true, Description: "Filter by whether Vercel Authentication is configured. Omit to include both enabled and disabled projects."},
		"password_protection_enabled":   schema.BoolAttribute{Optional: true, Description: "Filter by whether Password Protection is configured. Omit to include both enabled and disabled projects."},
		"trusted_ips_enabled":           schema.BoolAttribute{Optional: true, Description: "Filter by whether Trusted IPs is configured. Omit to include both enabled and disabled projects."},
	}
}

func projectProtectionSummarySchema(name string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: true, Description: name + " configuration. Bypasses and exceptions can still allow access.",
		Attributes: map[string]schema.Attribute{
			"enabled":         schema.BoolAttribute{Computed: true, Description: "Whether this protection mechanism is configured."},
			"deployment_type": schema.StringAttribute{Computed: true, Description: "The API deployment scope. Null when disabled."},
		},
	}
}

func protectionSummary(enabled bool, scope string) projectProtectionSummary {
	result := projectProtectionSummary{Enabled: types.BoolValue(enabled), DeploymentType: types.StringNull()}
	if enabled {
		result.DeploymentType = types.StringValue(scope)
	}
	return result
}

func summarizeProject(project client.ProjectResponse) projectSummary {
	result := projectSummary{
		ID: types.StringValue(project.ID), Name: types.StringValue(project.Name),
		TeamID: toTeamID(project.TeamID), Framework: types.StringPointerValue(project.Framework),
		GitRepository: types.ObjectNull(projectRepositorySummaryTypes),
		Passport:      protectionSummary(false, ""), VercelAuthentication: protectionSummary(false, ""),
		PasswordProtection: protectionSummary(false, ""), TrustedIPs: protectionSummary(false, ""),
	}
	if repo := project.Repository(); repo != nil {
		result.GitRepository = types.ObjectValueMust(projectRepositorySummaryTypes, map[string]attr.Value{"type": types.StringValue(repo.Type), "repo": types.StringValue(repo.Repo)})
	}
	if project.Passport != nil {
		result.Passport = protectionSummary(true, project.Passport.DeploymentType)
	}
	if project.VercelAuthentication != nil {
		result.VercelAuthentication = protectionSummary(project.VercelAuthentication.DeploymentType != "none", project.VercelAuthentication.DeploymentType)
	}
	if project.PasswordProtection != nil {
		result.PasswordProtection = protectionSummary(project.PasswordProtection.DeploymentType != "none", project.PasswordProtection.DeploymentType)
	}
	if project.TrustedIps != nil {
		result.TrustedIPs = protectionSummary(project.TrustedIps.DeploymentType != "none", project.TrustedIps.DeploymentType)
	}
	return result
}

func (config projectSearchConfig) matches(project projectSummary) bool {
	if !config.Framework.IsNull() && !config.Framework.Equal(project.Framework) {
		return false
	}
	for _, filter := range []struct{ wanted, actual types.Bool }{
		{config.PassportEnabled, project.Passport.Enabled},
		{config.VercelAuthenticationEnabled, project.VercelAuthentication.Enabled},
		{config.PasswordProtectionEnabled, project.PasswordProtection.Enabled},
		{config.TrustedIPsEnabled, project.TrustedIPs.Enabled},
	} {
		if !filter.wanted.IsNull() && !filter.wanted.Equal(filter.actual) {
			return false
		}
	}
	return true
}

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model projectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projects, err := d.client.ListProjectsWithOptions(ctx, client.ListProjectsRequest{
		TeamID: model.TeamID.ValueString(), RepositoryURL: model.RepositoryURL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing projects", err.Error())
		return
	}
	model.TeamID = toTeamID(d.client.TeamID(model.TeamID.ValueString()))
	model.Projects = make(map[string]projectSummary)
	for _, project := range projects {
		summary := summarizeProject(project)
		if model.projectSearchConfig.matches(summary) {
			model.Projects[project.ID] = summary
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
