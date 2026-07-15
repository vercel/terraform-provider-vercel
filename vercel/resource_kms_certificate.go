package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource              = &kmsCertificateResource{}
	_ resource.ResourceWithConfigure = &kmsCertificateResource{}
)

func newKMSCertificateResource() resource.Resource {
	return &kmsCertificateResource{}
}

type kmsCertificateResource struct {
	client *client.Client
}

type kmsCertificateSubjectModel struct {
	OU types.String `tfsdk:"ou"`
	C  types.String `tfsdk:"c"`
	ST types.String `tfsdk:"st"`
	L  types.String `tfsdk:"l"`
}

type kmsCertificateResourceModel struct {
	ID           types.String                `tfsdk:"id"`
	TeamID       types.String                `tfsdk:"team_id"`
	IssuerID     types.String                `tfsdk:"issuer_id"`
	Keepers      types.Map                   `tfsdk:"keepers"`
	NotBefore    types.String                `tfsdk:"not_before"`
	NotAfter     types.String                `tfsdk:"not_after"`
	Subject      *kmsCertificateSubjectModel `tfsdk:"subject"`
	Certificate  types.String                `tfsdk:"certificate"`
	KeyID        types.String                `tfsdk:"key_id"`
	SerialNumber types.String                `tfsdk:"serial_number"`
	KMSIssuerURL types.String                `tfsdk:"kms_issuer_url"`
}

func (r *kmsCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_certificate"
}

func (r *kmsCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *kmsCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a self-signed X.509 certificate minted for a Vercel KMS issuer's active
signing key.

The certificate is generated when the resource is created and is not persisted
server-side, so it exists only in Terraform state. Changing any input — or the
` + "`keepers`" + ` map — mints a fresh certificate with a new serial number.
Reading and deleting are no-ops, and this resource cannot be imported.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier for this resource, equal to the certificate serial number.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the issuer exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"issuer_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the issuer to mint a certificate for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"keepers": schema.MapAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Arbitrary map of values that, when changed, mints a fresh certificate.",
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"not_before": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The certificate validity start timestamp (RFC3339). Defaults to the time of creation. Changing this mints a fresh certificate.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"not_after": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The certificate validity end timestamp (RFC3339). Defaults to 12 months from creation. Changing this mints a fresh certificate.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"subject": schema.SingleNestedAttribute{
				Optional:      true,
				Description:   "The subject fields to include in the certificate. Changing this mints a fresh certificate.",
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"ou": schema.StringAttribute{Optional: true, Description: "Organizational unit."},
					"c":  schema.StringAttribute{Optional: true, Description: "Two-letter country code."},
					"st": schema.StringAttribute{Optional: true, Description: "State or province."},
					"l":  schema.StringAttribute{Optional: true, Description: "Locality."},
				},
			},
			"certificate": schema.StringAttribute{
				Computed:      true,
				Description:   "The PEM-encoded self-signed X.509 certificate.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the signing key the certificate was minted for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"serial_number": schema.StringAttribute{
				Computed:      true,
				Description:   "The hexadecimal serial number of the certificate.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kms_issuer_url": schema.StringAttribute{
				Computed:      true,
				Description:   "The issuer URL embedded in the certificate.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func kmsCertificateSubjectRequest(subject *kmsCertificateSubjectModel) *client.KMSCertificateSubject {
	if subject == nil {
		return nil
	}
	return &client.KMSCertificateSubject{
		OU: subject.OU.ValueString(),
		C:  subject.C.ValueString(),
		ST: subject.ST.ValueString(),
		L:  subject.L.ValueString(),
	}
}

func (r *kmsCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan kmsCertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateKMSCertificate(ctx, client.CreateKMSCertificateRequest{
		IssuerID:  plan.IssuerID.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
		NotBefore: plan.NotBefore.ValueString(),
		NotAfter:  plan.NotAfter.ValueString(),
		Subject:   kmsCertificateSubjectRequest(plan.Subject),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating KMS certificate",
			"Could not create KMS certificate, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "created kms certificate", map[string]any{
		"issuer_id":     plan.IssuerID.ValueString(),
		"serial_number": out.SerialNumber,
	})

	result := kmsCertificateResourceModel{
		ID:           types.StringValue(out.SerialNumber),
		TeamID:       toTeamID(r.client.TeamID(plan.TeamID.ValueString())),
		IssuerID:     plan.IssuerID,
		Keepers:      plan.Keepers,
		NotBefore:    types.StringValue(out.NotBefore),
		NotAfter:     types.StringValue(out.NotAfter),
		Subject:      plan.Subject,
		Certificate:  types.StringValue(out.Certificate),
		KeyID:        types.StringValue(out.KeyID),
		SerialNumber: types.StringValue(out.SerialNumber),
		KMSIssuerURL: types.StringValue(out.KMSIssuerURL),
	}
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

// Read is a no-op: the certificate is not persisted server-side, so there is
// nothing to refresh. resp.State is pre-populated with the prior state by the
// framework, so leaving it untouched preserves it.
func (r *kmsCertificateResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

// Update is unreachable in practice because every configurable attribute forces
// replacement, but the framework requires the method.
func (r *kmsCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan kmsCertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete is a no-op: there is nothing to delete server-side. Removing the
// resource only drops the certificate from Terraform state.
func (r *kmsCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state kmsCertificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "removing kms certificate from state; certificates are ephemeral and have no server-side lifecycle", map[string]any{
		"issuer_id":     state.IssuerID.ValueString(),
		"serial_number": state.SerialNumber.ValueString(),
	})
}
