package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// ProjectDomain reflects the state terraform stores internally for a project domain.
type ProjectDomain struct {
	Domain             types.String `tfsdk:"domain"`
	GitBranch          types.String `tfsdk:"git_branch"`
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	Redirect           types.String `tfsdk:"redirect"`
	RedirectStatusCode types.Int64  `tfsdk:"redirect_status_code"`
	TeamID             types.String `tfsdk:"team_id"`
}

func convertResponseToProjectDomain(response client.ProjectDomainResponse) ProjectDomain {
	return ProjectDomain{
		Domain:             types.String{Value: response.Name},
		GitBranch:          fromStringPointer(response.GitBranch),
		ID:                 types.String{Value: response.Name},
		ProjectID:          types.String{Value: response.ProjectID},
		Redirect:           fromStringPointer(response.Redirect),
		RedirectStatusCode: fromInt64Pointer(response.RedirectStatusCode),
		TeamID:             toTeamID(response.TeamID),
	}
}

func (p *ProjectDomain) toCreateRequest() client.CreateProjectDomainRequest {
	return client.CreateProjectDomainRequest{
		GitBranch:          p.GitBranch.Value,
		Name:               p.Domain.Value,
		Redirect:           p.Redirect.Value,
		RedirectStatusCode: p.RedirectStatusCode.Value,
	}
}

func (p *ProjectDomain) toUpdateRequest() client.UpdateProjectDomainRequest {
	return client.UpdateProjectDomainRequest{
		GitBranch:          toStrPointer(p.GitBranch),
		Redirect:           toStrPointer(p.Redirect),
		RedirectStatusCode: toInt64Pointer(p.RedirectStatusCode),
	}
}
