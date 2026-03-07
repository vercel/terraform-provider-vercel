package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var bulkRedirectObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"source":         types.StringType,
		"destination":    types.StringType,
		"status_code":    types.Int64Type,
		"case_sensitive": types.BoolType,
		"query":          types.BoolType,
	},
}

type bulkRedirectModel struct {
	Source        types.String `tfsdk:"source"`
	Destination   types.String `tfsdk:"destination"`
	StatusCode    types.Int64  `tfsdk:"status_code"`
	CaseSensitive types.Bool   `tfsdk:"case_sensitive"`
	Query         types.Bool   `tfsdk:"query"`
}

func (r bulkRedirectModel) toClient() client.BulkRedirect {
	redirect := client.BulkRedirect{
		Source:      r.Source.ValueString(),
		Destination: r.Destination.ValueString(),
	}

	if !r.StatusCode.IsNull() && !r.StatusCode.IsUnknown() {
		value := r.StatusCode.ValueInt64()
		redirect.StatusCode = &value
	}
	if !r.CaseSensitive.IsNull() && !r.CaseSensitive.IsUnknown() {
		value := r.CaseSensitive.ValueBool()
		redirect.CaseSensitive = &value
	}
	if !r.Query.IsNull() && !r.Query.IsUnknown() {
		value := r.Query.ValueBool()
		redirect.Query = &value
	}

	return redirect
}

func expandBulkRedirects(ctx context.Context, list types.Set) ([]client.BulkRedirect, diag.Diagnostics) {
	var redirects []bulkRedirectModel
	diags := list.ElementsAs(ctx, &redirects, false)
	if diags.HasError() {
		return nil, diags
	}

	request := make([]client.BulkRedirect, 0, len(redirects))
	for _, redirect := range redirects {
		request = append(request, redirect.toClient())
	}

	return request, diags
}

func flattenBulkRedirects(redirects []client.BulkRedirect) types.Set {
	values := make([]attr.Value, 0, len(redirects))
	for _, redirect := range redirects {
		statusCode := int64PointerValue(redirect.StatusCode)
		if redirect.Permanent != nil && redirect.StatusCode == nil {
			if *redirect.Permanent {
				statusCode = types.Int64Value(308)
			} else {
				statusCode = types.Int64Value(307)
			}
		}

		values = append(values, types.ObjectValueMust(
			bulkRedirectObjectType.AttrTypes,
			map[string]attr.Value{
				"source":         types.StringValue(redirect.Source),
				"destination":    types.StringValue(redirect.Destination),
				"status_code":    statusCode,
				"case_sensitive": boolDefaultFalseValue(redirect.CaseSensitive),
				"query":          boolDefaultFalseValue(redirect.Query),
			},
		))
	}

	return types.SetValueMust(bulkRedirectObjectType, values)
}

func findLiveBulkRedirectVersion(versions []client.BulkRedirectVersion) (client.BulkRedirectVersion, bool) {
	for _, version := range versions {
		if version.IsLive {
			return version, true
		}
	}

	return client.BulkRedirectVersion{}, false
}

func readLiveBulkRedirects(ctx context.Context, c *client.Client, projectID, teamID string) (client.BulkRedirects, bool, error) {
	versions, err := c.GetBulkRedirectVersions(ctx, projectID, teamID)
	if err != nil {
		return client.BulkRedirects{}, false, err
	}

	version, ok := findLiveBulkRedirectVersion(versions)
	if !ok {
		return client.BulkRedirects{
			ProjectID: projectID,
			TeamID:    c.TeamID(teamID),
			Redirects: []client.BulkRedirect{},
		}, false, nil
	}

	redirects, err := c.GetBulkRedirects(ctx, client.GetBulkRedirectsRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		VersionID: version.ID,
	})
	if err != nil {
		return client.BulkRedirects{}, false, err
	}

	if redirects.Version == nil {
		redirects.Version = &version
	}

	return redirects, true, nil
}

func bulkRedirectVersionID(version *client.BulkRedirectVersion) types.String {
	if version == nil || version.ID == "" {
		return types.StringNull()
	}

	return types.StringValue(version.ID)
}

func int64PointerValue(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*v)
}

func boolDefaultFalseValue(v *bool) types.Bool {
	if v == nil {
		return types.BoolValue(false)
	}

	return types.BoolValue(*v)
}
