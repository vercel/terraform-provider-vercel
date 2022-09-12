package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
Provides an Alias resource.

An Alias allows a ` + "`vercel_deployment` to be accessed through a different URL.",
		Attributes: map[string]tfsdk.Attribute{
			"alias": {
				Description:   "The Alias we want to assign to the deployment defined in the URL.",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Type:          types.StringType,
			},
			"deployment_id": {
				Description:   "The id of the Deployment the Alias should be associated with.",
				Required:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Type:          types.StringType,
			},
			"team_id": {
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Alias and Deployment exist under.",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace(), resource.UseStateForUnknown()},
				Type:          types.StringType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
		},
	}, nil
}

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceAliasType) NewResource(_ context.Context, p provider.Provider) (resource.Resource, diag.Diagnostics) {
	return resourceAlias{
		p: *(p.(*vercelProvider)),
	}, nil
}

type resourceAlias struct {
	p vercelProvider
}

// Create will create an alias within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r resourceAlias) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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
		return
	}

	out, err := r.p.client.CreateAlias(ctx, client.CreateAliasRequest{
		Alias: plan.Alias.Value,
	}, plan.DeploymentID.Value, plan.TeamID.Value)
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
		"deployment_id": plan.DeploymentID.Value,
		"alias_id":      result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read alias information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r resourceAlias) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Alias
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetAlias(ctx, state.ID.Value, state.TeamID.Value)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading alias",
			fmt.Sprintf("Could not get alias %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.ID.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToAlias(out, state)
	tflog.Trace(ctx, "read alias", map[string]interface{}{
		"team_id":  result.TeamID.Value,
		"alias_id": result.ID.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the Alias state.
func (r resourceAlias) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
func (r resourceAlias) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Alias
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.p.client.DeleteAlias(ctx, state.ID.Value, state.TeamID.Value)
	if client.NotFound(err) {
		return
	}
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

	tflog.Trace(ctx, "deleted alias", map[string]interface{}{
		"team_id":  state.TeamID.Value,
		"alias_id": state.ID.Value,
	})
}
