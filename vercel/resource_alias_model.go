package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Alias represents the terraform state for an alias resource.
type Alias struct {
	Alias        types.String `tfsdk:"alias"`
	UID          types.String `tfsdk:"uid"`
	DeploymentId types.String `tfsdk:"deployment_id"`
	TeamID       types.String `tfsdk:"team_id"`
}

// convertResponseToAlias is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToAlias(response client.AliasResponse, plan Alias) Alias {
	return Alias{
		Alias:        plan.Alias,
		UID:          types.String{Value: response.UID},
		DeploymentId: plan.DeploymentId,
		TeamID:       plan.TeamID,
	}
}
