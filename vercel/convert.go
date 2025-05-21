package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func intPtrToInt64Ptr(i *int) *int64 {
	if i == nil {
		return nil
	}
	val := int64(*i)
	return &val
}

func stringsToSet(ctx context.Context, strings []string) (types.Set, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	stringSet := []attr.Value{}
	for _, s := range strings {
		stringSet = append(stringSet, types.StringValue(s))
	}

	set, d := types.SetValueFrom(ctx, types.StringType, stringSet)
	diags.Append(d...)
	return set, diags
}
