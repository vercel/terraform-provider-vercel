package vercel

import "github.com/hashicorp/terraform-plugin-framework/types"

func toPtr[T any](v T) *T {
	return &v
}

func toStrPointer(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return toPtr(v.ValueString())
}

func toBoolPointer(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return toPtr(v.ValueBool())
}

func toInt64Pointer(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return toPtr(v.ValueInt64())
}

func fromStringPointer(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func fromBoolPointer(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

func fromInt64Pointer(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func toTeamID(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
