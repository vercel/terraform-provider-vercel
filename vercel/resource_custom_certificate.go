package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &customCertificateResource{}
	_ resource.ResourceWithConfigure = &customCertificateResource{}
)

func newCustomCertificateResource() resource.Resource {
	return &customCertificateResource{}
}

type customCertificateResource struct {
	client *client.Client
}

func (r *customCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_certificate"
}

func (r *customCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customCertificateResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Custom Certificate Resource, allowing Custom Certificates to be uploaded to Vercel.

By default, Vercel provides all domains with a custom SSL certificates. However, Enterprise teams can upload their own custom SSL certificate.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/domains/custom-SSL-certificate).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Custom Certificate.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Custom Certificate should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"private_key": schema.StringAttribute{
				Description:   "The private key of the Certificate. Should be in PEM format.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"certificate": schema.StringAttribute{
				Description:   "The certificate itself. Should be in PEM format.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"certificate_authority_certificate": schema.StringAttribute{
				Description:   "The Certificate Authority root certificate such as one of Let's Encrypt's ISRG root certificates. This will be provided by your certificate issuer and is different to the core certificate. This may be included in their download process or available for download on their website. Should be in PEM format.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type CustomCertificate struct {
	ID                              types.String `tfsdk:"id"`
	TeamID                          types.String `tfsdk:"team_id"`
	PrivateKey                      types.String `tfsdk:"private_key"`
	Certificate                     types.String `tfsdk:"certificate"`
	CertificateAuthorityCertificate types.String `tfsdk:"certificate_authority_certificate"`
}

func (r *customCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomCertificate
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UploadCustomCertificate(ctx, client.UploadCustomCertificateRequest{
		TeamID:                          plan.TeamID.ValueString(),
		PrivateKey:                      plan.PrivateKey.ValueString(),
		Certificate:                     plan.Certificate.ValueString(),
		CertificateAuthorityCertificate: plan.CertificateAuthorityCertificate.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error uploading Custom Certificate",
			"Could not upload Custom Certificate, unexpected error: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(out.ID)
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	tflog.Info(ctx, "uploaded custom certificate", map[string]any{
		"team_id": plan.TeamID.ValueString(),
		"id":      out.ID,
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomCertificate
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// This is basically a check that it still exists, as the cert itself is immutable.
	out, err := r.client.GetCustomCertificate(ctx, client.GetCustomCertificateRequest{
		ID:     state.ID.ValueString(),
		TeamID: state.TeamID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Custom Certificate",
			fmt.Sprintf("Could not get Custom Certificate %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "read certificate", map[string]any{
		"team_id": state.TeamID.ValueString(),
		"id":      out.ID,
	})

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a Custom Certificate is not supported",
		"Updating a Custom Certificate is not supported",
	)
}

func (r *customCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomCertificate
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCustomCertificate(ctx, client.DeleteCustomCertificateRequest{
		TeamID: state.TeamID.ValueString(),
		ID:     state.ID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Custom Certificate",
			fmt.Sprintf(
				"Could not delete Custom Certificate %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted custom certificate", map[string]any{
		"team_id": state.TeamID.ValueString(),
		"id":      state.ID.ValueString(),
	})
}
