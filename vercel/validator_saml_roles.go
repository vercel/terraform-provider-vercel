package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type samlRolesValidator struct{}

var _ validator.Map = &samlRolesValidator{}

func validateSamlRoles() validator.Map {
	return &samlRolesValidator{}
}

func (v *samlRolesValidator) Description(ctx context.Context) string {
	return "Validates that exactly one of role or access_group_id is defined for each SAML role entry"
}

func (v *samlRolesValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that exactly one of role or access_group_id is defined for each SAML role entry"
}

func (v *samlRolesValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Get all the map keys
	keys := req.ConfigValue.Elements()

	// For each key in the map
	for key, value := range keys {
		// Convert the value to an object
		obj, ok := value.(types.Object)

		if !ok || obj.IsNull() || obj.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtMapKey(key),
				"Invalid SAML Role Configuration: "+key,
				"Expected an object with role or access_group_id",
			)
			continue
		}

		role := obj.Attributes()["role"]
		accessGroupID := obj.Attributes()["access_group_id"]

		// Check if both are set
		if !role.IsNull() && !role.IsUnknown() && !accessGroupID.IsNull() && !accessGroupID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtMapKey(key),
				"Invalid SAML Role Configuration: "+key,
				"Only one of 'role' or 'access_group_id' can be set, not both",
			)
			continue
		}

		// Check if neither is set
		if (role.IsNull() || role.IsUnknown()) && (accessGroupID.IsNull() || accessGroupID.IsUnknown()) {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtMapKey(key),
				"Invalid SAML Role Configuration: "+key,
				"Either 'role' or 'access_group_id' must be set",
			)
			continue
		}
	}
}
