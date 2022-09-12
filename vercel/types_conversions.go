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

func toInt64Pointer(v types.Int64) *int64 {
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

func fromInt64Pointer(v *int64) types.Int64 {
	if v == nil {
		return types.Int64{Null: true}
	}
	return types.Int64{Value: *v}
}

func toTeamID(v string) types.String {
	return types.String{Value: v, Null: v == ""}
}
