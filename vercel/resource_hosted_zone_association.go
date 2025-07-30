package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

func (r *hostedZoneAssociationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostedZoneAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_zone_association"
}

// Read implements resource.Resource.
func (r *hostedZoneAssociationResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
	panic("unimplemented")
}

func (r *hostedZoneAssociationResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Hosted Zone Association resource.

Hosted Zone Associations provide a way to associate an AWS Route53 Hosted Zone with a Secure Compute network.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs).
`,
		Attributes: map[string]schema.Attribute{
			"hostedZoneId": schema.StringAttribute{
				Description: "The ID of the Hosted Zone.",
				Required:    true,
			},
			"hostedZoneName": schema.StringAttribute{
				Description: "The name of the Hosted Zone.",
				Required:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The ID of the AWS Account that owns the Hosted Zone.",
				Required:    true,
			},
		},
	}
}

// Update implements resource.Resource.
func (r *hostedZoneAssociationResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	panic("unimplemented")
}
