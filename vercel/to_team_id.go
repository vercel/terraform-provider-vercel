package vercel

import "github.com/hashicorp/terraform-plugin-framework/types"

func toTeamID(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
