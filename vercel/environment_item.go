package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnvironmentItem reflects the state terraform stores internally for a project's environment variable.
type EnvironmentItem struct {
	Target               types.Set    `tfsdk:"target"`
	CustomEnvironmentIDs types.Set    `tfsdk:"custom_environment_ids"`
	GitBranch            types.String `tfsdk:"git_branch"`
	Value                types.String `tfsdk:"value"`
	ID                   types.String `tfsdk:"id"`
	Sensitive            types.Bool   `tfsdk:"sensitive"`
	Comment              types.String `tfsdk:"comment"`
}

func (e *EnvironmentItem) toAttrValue() attr.Value {
	return types.ObjectValueMust(EnvVariableElemType.AttrTypes, map[string]attr.Value{
		"id":                     e.ID,
		"value":                  e.Value,
		"target":                 e.Target,
		"custom_environment_ids": e.CustomEnvironmentIDs,
		"git_branch":             e.GitBranch,
		"sensitive":              e.Sensitive,
		"comment":                e.Comment,
	})
}

var EnvVariableElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"value": types.StringType,
		"target": types.SetType{
			ElemType: types.StringType,
		},
		"custom_environment_ids": types.SetType{
			ElemType: types.StringType,
		},
		"git_branch": types.StringType,
		"id":         types.StringType,
		"sensitive":  types.BoolType,
		"comment":    types.StringType,
	},
}