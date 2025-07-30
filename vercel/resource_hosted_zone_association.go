package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &hostedZoneAssociationResource{}
	_ resource.ResourceWithConfigure   = &hostedZoneAssociationResource{}
	_ resource.ResourceWithImportState = &hostedZoneAssociationResource{}
)

type hostedZoneAssociationResource struct {
	client *client.Client
}

// Configure implements resource.ResourceWithConfigure.
func (h *hostedZoneAssociationResource) Configure(context.Context, resource.ConfigureRequest, *resource.ConfigureResponse) {
	panic("unimplemented")
}

// Create implements resource.Resource.
func (h *hostedZoneAssociationResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {
	panic("unimplemented")
}

// Delete implements resource.Resource.
func (h *hostedZoneAssociationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	panic("unimplemented")
}

// ImportState implements resource.ResourceWithImportState.
func (h *hostedZoneAssociationResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
	panic("unimplemented")
}

func (h *hostedZoneAssociationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_zone_association"
}

// Read implements resource.Resource.
func (h *hostedZoneAssociationResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
	panic("unimplemented")
}

// Schema implements resource.Resource.
func (h *hostedZoneAssociationResource) Schema(context.Context, resource.SchemaRequest, *resource.SchemaResponse) {
	panic("unimplemented")
}

// Update implements resource.Resource.
func (h *hostedZoneAssociationResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	panic("unimplemented")
}
