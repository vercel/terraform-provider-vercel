package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestAllResourcesHaveIDAttribute ensures every resource has an "id" computed attribute.
// This is required for Pulumi bridge compatibility - without it, pulumi-vercel maintainers
// must manually add ComputeID functions for each resource.
//
// If this test fails, add an "id" computed attribute to your resource's Schema():
//
//	"id": schema.StringAttribute{
//	    Computed:    true,
//	    Description: "The unique identifier for this resource.",
//	    PlanModifiers: []planmodifier.String{
//	        stringplanmodifier.UseStateForUnknown(),
//	    },
//	},
func TestAllResourcesHaveIDAttribute(t *testing.T) {
	ctx := context.Background()
	provider := &vercelProvider{}

	resourceFactories := provider.Resources(ctx)

	for _, factory := range resourceFactories {
		res := factory()

		metaReq := resource.MetadataRequest{ProviderTypeName: "vercel"}
		metaResp := &resource.MetadataResponse{}
		res.Metadata(ctx, metaReq, metaResp)
		resourceName := metaResp.TypeName

		schemaReq := resource.SchemaRequest{}
		schemaResp := &resource.SchemaResponse{}
		res.Schema(ctx, schemaReq, schemaResp)

		if !hasIDAttribute(schemaResp.Schema) {
			t.Errorf("Resource %q is missing required 'id' attribute. "+
				"All resources must have a computed 'id' attribute for Pulumi bridge compatibility. "+
				"See TestAllResourcesHaveIDAttribute doc comment for the pattern to add.",
				resourceName)
		}
	}
}

func hasIDAttribute(s schema.Schema) bool {
	if s.Attributes == nil {
		return false
	}

	idAttr, exists := s.Attributes["id"]
	if !exists {
		return false
	}

	stringAttr, ok := idAttr.(schema.StringAttribute)
	// Must be Computed (with optional input allowed) - we need a computed output identifier for Pulumi
	// Some resources like team_config and edge_config_schema need Optional+Computed to accept user input
	return ok && stringAttr.Computed
}
