package vercel

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

type resourceAliasType struct{}

// GetSchema returns the schema information for an alias resource.
func (r resourceAliasType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides an alias resource.

An alias allows a deployment to be accessed through a different URL.`,
		Attributes: map[string]tfsdk.Attribute{
			"alias": {
				Description:   "The alias to be set on the deployment. It will become the subdomain of the Vercel project top level domain",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"deployment_id": {
				Description:   "The deployment id to alias",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"team_id": {
				Description:   "The team or scope id",
				Optional:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"uid": {
				Computed: true,
				Type:     types.StringType,
			},
		},
	}, nil
}

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceAliasType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceAlias{
		p: *(p.(*provider)),
	}, nil
}

type resourceAlias struct {
	p provider
}

// Create will create an alias within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r resourceAlias) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var plan Alias
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting alias plan",
			"Error getting alias plan",
		)
		return
	}

	car := client.CreateAliasRequest{
		Alias: plan.Alias.Value,
	}
	out, err := r.p.client.CreateAlias(ctx, car, plan.DeploymentId.Value, plan.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating alias",
			"Could not create alias, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToAlias(out, plan)
	tflog.Trace(ctx, "created alias", map[string]interface{}{
		"team_id":       plan.TeamID.Value,
		"deployment_id": plan.DeploymentId.Value,
		"alias_uid":     result.UID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read alias information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r resourceAlias) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state Alias
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetAlias(ctx, state.UID.Value, state.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading alias",
			fmt.Sprintf("Could not get alias %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.UID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToAlias(out, state)
	tflog.Trace(ctx, "read alias", map[string]interface{}{
		"team_id":   result.TeamID.Value,
		"alias_uid": result.UID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the Alias state.
func (r resourceAlias) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan Alias
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting alias plan",
			"Error getting alias plan",
		)
		return
	}

	var state Alias
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes an Alias.
func (r resourceAlias) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state Alias
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.p.client.DeleteAlias(ctx, state.UID.Value, state.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting alias",
			fmt.Sprintf(
				"Could not delete alias %s, unexpected error: %s",
				state.Alias.Value,
				err,
			),
		)
		return
	}
	tflog.Trace(ctx, "deleted alias")
	resp.State.RemoveResource(ctx)
}
