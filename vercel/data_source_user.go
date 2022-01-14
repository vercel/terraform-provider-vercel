package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceUserType struct{}

func (r dataSourceUserType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Computed: true,
				Type:     types.StringType,
			},
			"username": {
				Computed: true,
				Type:     types.StringType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"plan": {
				Computed: true,
				Type:     types.StringType,
			},
		},
	}, nil
}

func (r dataSourceUserType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceUser{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceUser struct {
	p provider
}

type UserData struct {
	Name     types.String `tfsdk:"name"`
	Username types.String `tfsdk:"username"`
	ID       types.String `tfsdk:"id"`
	Plan     types.String `tfsdk:"plan"`
}

func (r dataSourceUser) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var config UserData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.p.client.GetUser(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			fmt.Sprintf("Could not read user, unexpected error: %s", err),
		)
		return
	}
	diags = resp.State.Set(ctx, &UserData{
		Name:     types.String{Value: user.Name},
		Username: types.String{Value: user.Username},
		ID:       types.String{Value: user.Username},
		Plan:     types.String{Value: user.Billing.Plan},
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
