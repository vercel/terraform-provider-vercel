package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dataSourceDomainType struct{}

func (r dataSourceDomainType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Required: true,
				Type:     types.StringType,
			},
			"team_id": {
				Optional: true,
				Type:     types.StringType,
			},
			"suffix": {
				Computed: true,
				Type:     types.BoolType,
			},
			"verified": {
				Computed: true,
				Type:     types.BoolType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"nameservers": {
				Computed: true,
				Type: types.ListType{
					ElemType: types.StringType,
				},
			},
			"creator": {
				Computed: true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"username": {
						Computed: true,
						Type:     types.StringType,
					},
					"email": {
						Computed: true,
						Type:     types.StringType,
					},
					"customer_id": {
						Computed: true,
						Type:     types.StringType,
					},
					"id": {
						Computed: true,
						Type:     types.StringType,
					},
					"is_domain_reseller": {
						Computed: true,
						Type:     types.BoolType,
					},
				}),
			},
			"created_at": {
				Computed: true,
				Type:     types.Int64Type,
			},
			"expires_at": {
				Computed: true,
				Type:     types.Int64Type,
			},
			"bought_at": {
				Computed: true,
				Type:     types.Int64Type,
			},
			"transferred_at": {
				Computed: true,
				Type:     types.Int64Type,
			},
			"transfer_started_at": {
				Computed: true,
				Type:     types.Int64Type,
			},
			"service_type": {
				Computed: true,
				Type:     types.StringType,
			},
			"renew": {
				Computed: true,
				Type:     types.BoolType,
			},
		},
	}, nil
}

func (r dataSourceDomainType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceDomain{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceDomain struct {
	p provider
}

type CreatorData struct {
	Username         types.String `tfsdk:"username"`
	Email            types.String `tfsdk:"email"`
	CustomerID       types.String `tfsdk:"customer_id"`
	ID               types.String `tfsdk:"id"`
	IsDomainReseller types.Bool   `tfsdk:"is_domain_reseller"`
}

type DomainData struct {
	Name              types.String `tfsdk:"name"`
	TeamID            types.String `tfsdk:"team_id"`
	Suffix            types.Bool   `tfsdk:"suffix"`
	Verified          types.Bool   `tfsdk:"verified"`
	ID                types.String `tfsdk:"id"`
	Nameservers       []string     `tfsdk:"nameservers"`
	Creator           *CreatorData `tfsdk:"creator"`
	CreatedAt         types.Int64  `tfsdk:"created_at"`
	ExpiresAt         types.Int64  `tfsdk:"expires_at"`
	BoughtAt          types.Int64  `tfsdk:"bought_at"`
	TransferredAt     types.Int64  `tfsdk:"transferred_at"`
	TransferStartedAt types.Int64  `tfsdk:"transfer_started_at"`
	ServiceType       types.String `tfsdk:"service_type"`
	Renew             types.Bool   `tfsdk:"renew"`
}

func (r dataSourceDomain) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var config DomainData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.p.client.GetDomain(ctx, config.Name.Value, config.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading domain",
			fmt.Sprintf("Could not read domain %s, unexpected error: %s", config.Name.Value, err),
		)
		return
	}
	tflog.Trace(ctx, "READ DOMAIN", "domain", domain)
	diags = resp.State.Set(ctx, &DomainData{
		Name:        types.String{Value: domain.Name},
		TeamID:      types.String{Null: config.TeamID.Unknown || config.TeamID.Null, Value: config.TeamID.Value},
		Suffix:      types.Bool{Value: domain.Suffix},
		Verified:    types.Bool{Value: domain.Verified},
		ID:          types.String{Value: domain.ID},
		Nameservers: domain.Nameservers,
		Creator: &CreatorData{
			Username:         types.String{Value: domain.Creator.Username},
			Email:            types.String{Value: domain.Creator.Email},
			CustomerID:       fromStringPointer(domain.Creator.CustomerID),
			ID:               types.String{Value: domain.Creator.ID},
			IsDomainReseller: fromBoolPointer(domain.Creator.IsDomainReseller),
		},
		CreatedAt:         types.Int64{Value: domain.CreatedAt},
		ExpiresAt:         fromInt64Pointer(domain.ExpiresAt),
		BoughtAt:          fromInt64Pointer(domain.BoughtAt),
		TransferredAt:     fromInt64Pointer(domain.TransferredAt),
		TransferStartedAt: fromInt64Pointer(domain.TransferStartedAt),
		ServiceType:       types.String{Value: domain.ServiceType},
		Renew:             fromBoolPointer(domain.Renew),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
