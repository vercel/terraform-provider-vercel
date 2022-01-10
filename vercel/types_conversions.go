package vercel

import "github.com/hashicorp/terraform-plugin-framework/types"

func toStrPointer(v types.String) *string {
	if v.Null || v.Unknown {
		return nil
	}
	return &v.Value
}

func toBoolPointer(v types.Bool) *bool {
	if v.Null || v.Unknown {
		return nil
	}
	return &v.Value
}

func fromStringPointer(v *string) types.String {
	if v == nil {
		return types.String{Null: true}
	}
	return types.String{Value: *v}
}

func fromBoolPointer(v *bool) types.Bool {
	if v == nil {
		return types.Bool{Null: true}
	}
	return types.Bool{Value: *v}
}
