package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Define the element types for the lists
var AutomaticRollingReleaseElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"target_percentage": types.Int64Type,
		"duration":          types.Int64Type,
	},
}

var ManualRollingReleaseElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"target_percentage": types.Int64Type,
	},
}

// Define the stage types
type AutomaticStage struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
	Duration         types.Int64 `tfsdk:"duration"`
}

type ManualStage struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
}
