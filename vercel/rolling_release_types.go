package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Define the element type for stages
var RollingReleaseStageElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"target_percentage": types.Int64Type,
		"duration":          types.Int64Type,
	},
}

// Define the stage type
type RollingReleaseStage struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
	Duration         types.Int64 `tfsdk:"duration"`
}
