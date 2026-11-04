package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ list.ListResource              = &projectListResource{}
	_ list.ListResourceWithConfigure = &projectListResource{}
)

type projectListResource struct{ projectResource }

func newProjectListResource() list.ListResource { return &projectListResource{} }

func (r *projectListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Discovers existing projects for Terraform query and bulk import. Requires Terraform 1.14 or later. Uses the provider's default team unless team_id is set. All supplied filters must match. Generating configuration adopts the whole vercel_project resource, including deletion on destroy. Review the generated settings and supply any required sensitive inputs, such as passwords and Trusted IP addresses, before applying. The API does not return existing protection passwords. Protection filters describe configured settings, not effective access after bypasses or exceptions.",
		Attributes: map[string]schema.Attribute{
			"team_id":                       schema.StringAttribute{Optional: true, Description: "The team ID to query. Defaults to the team configured on the provider, or the personal account when no team is configured."},
			"repository_url":                schema.StringAttribute{Optional: true, Description: "Filter by the connected Git repository URL, using the API's repository URL matching."},
			"framework":                     schema.StringAttribute{Optional: true, Description: "Filter by the exact framework slug, for example nextjs. Omit to include all frameworks."},
			"passport_enabled":              schema.BoolAttribute{Optional: true, Description: "Filter by whether Passport is configured. Omit to include both enabled and disabled projects."},
			"vercel_authentication_enabled": schema.BoolAttribute{Optional: true, Description: "Filter by whether Vercel Authentication is configured. Omit to include both enabled and disabled projects."},
			"password_protection_enabled":   schema.BoolAttribute{Optional: true, Description: "Filter by whether Password Protection is configured. Omit to include both enabled and disabled projects."},
			"trusted_ips_enabled":           schema.BoolAttribute{Optional: true, Description: "Filter by whether Trusted IPs is configured. Omit to include both enabled and disabled projects."},
		},
	}
}

func (r *projectListResource) List(ctx context.Context, req list.ListRequest, resp *list.ListResultsStream) {
	var config projectSearchConfig
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		resp.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}
	resp.Results = func(push func(list.ListResult) bool) {
		if req.Limit <= 0 {
			return
		}
		request := client.ListProjectsRequest{TeamID: config.TeamID.ValueString(), RepositoryURL: config.RepositoryURL.ValueString()}
		seenCursors := make(map[int64]bool)
		seenProjects := make(map[string]bool)
		var count int64
		fail := func(err error) {
			push(list.ListResult{Diagnostics: diag.Diagnostics{diag.NewErrorDiagnostic("Error listing projects", err.Error())}})
		}
		for {
			if err := ctx.Err(); err != nil {
				fail(err)
				return
			}
			page, err := r.client.ListProjectsPage(ctx, request)
			if err != nil {
				fail(err)
				return
			}
			for _, project := range page.Projects {
				if seenProjects[project.ID] {
					continue
				}
				seenProjects[project.ID] = true
				if !config.matches(summarizeProject(project)) {
					continue
				}
				result := req.NewListResult(ctx)
				result.DisplayName = project.Name
				if req.IncludeResource {
					project, err = r.client.GetProject(ctx, project.ID, project.TeamID)
					if err != nil {
						fail(err)
						return
					}
					model, err := convertResponseToProject(ctx, project, nullProject, nil)
					if err != nil {
						fail(fmt.Errorf("converting project %s: %w", project.ID, err))
						return
					}
					result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
				}
				result.Diagnostics.Append(result.Identity.Set(ctx, projectIdentity{ProjectID: types.StringValue(project.ID), TeamID: toTeamID(project.TeamID)})...)
				if !push(result) || result.Diagnostics.HasError() {
					return
				}
				count++
				if count >= req.Limit {
					return
				}
			}
			next := page.Pagination.Next
			if next == nil {
				return
			}
			if seenCursors[*next] {
				fail(fmt.Errorf("project pagination cursor repeated"))
				return
			}
			seenCursors[*next] = true
			request.Until = next
		}
	}
}
