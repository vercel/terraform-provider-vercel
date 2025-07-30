package vercel

import (
	"context"
	"fmt"

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

func (r *hostedZoneAssociationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create implements resource.Resource.
func (r *hostedZoneAssociationResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {
	panic("unimplemented")
}

// Delete implements resource.Resource.
func (r *hostedZoneAssociationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	panic("unimplemented")
}

// ImportState implements resource.ResourceWithImportState.
func (r *hostedZoneAssociationResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
	panic("unimplemented")
}

func (r *hostedZoneAssociationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_zone_association"
}

// Read implements resource.Resource.
func (r *hostedZoneAssociationResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
	panic("unimplemented")
}

// Schema implements resource.Resource.
func (r *hostedZoneAssociationResource) Schema(context.Context, resource.SchemaRequest, *resource.SchemaResponse) {
	panic("unimplemented")
}

// Update implements resource.Resource.
func (r *hostedZoneAssociationResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	panic("unimplemented")
}
