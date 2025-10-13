package vercel

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &firewallBypassResource{}
	_ resource.ResourceWithConfigure   = &firewallBypassResource{}
	_ resource.ResourceWithImportState = &firewallBypassResource{}
)

func newFirewallBypassResource() resource.Resource {
	return &firewallBypassResource{}
}

type firewallBypassResource struct {
	client *client.Client
}

func (r *firewallBypassResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_bypass"
}

func (r *firewallBypassResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *firewallBypassResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Firewall Bypass Rule

Firewall Bypass Rules configure sets of domains and ip address to prevent bypass Vercel's system mitigations for.  The hosts used in a bypass rule must be a production domain assigned to the associated project.  Requests that bypass system mitigations will incur usage.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The identifier for the firewall bypass rule.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project to assign the bypass rule to ",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Required:      true,
				Description:   "The domain to configure the bypass rule for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_ip": schema.StringAttribute{
				Required:      true,
				Description:   "The source IP address to configure the bypass rule for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"note": schema.StringAttribute{
				Optional:      true,
				Description:   "A note to describe the bypass rule. Maximum length is 500 characters.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type FirewallBypassRule struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	Domain    types.String `tfsdk:"domain"`
	SourceIp  types.String `tfsdk:"source_ip"`
	Note      types.String `tfsdk:"note"`
}

func responseToBypassRule(out client.FirewallBypass) FirewallBypassRule {
	split := strings.Split(out.Id, "#")
	domain := out.Domain
	if out.IsProjectRule {
		domain = "*"
	}
	return FirewallBypassRule{
		ID:        types.StringValue(out.Id),
		TeamID:    types.StringValue(out.OwnerId),
		ProjectID: types.StringValue(split[0]),
		Domain:    types.StringValue(domain),
		SourceIp:  types.StringValue(out.Ip),
		Note:      types.StringValue(out.Note),
	}
}

func (r *firewallBypassResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallBypassRule
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateFirewallBypass(
		ctx,
		plan.TeamID.ValueString(),
		plan.ProjectID.ValueString(),
		client.FirewallBypassRule{
			Domain:   plan.Domain.ValueString(),
			SourceIp: plan.SourceIp.ValueString(),
			Note:     plan.Note.ValueString(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating firewall bypass",
			"Could not create Firewall Bypass, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToBypassRule(out)
	tflog.Info(ctx, "created firewall bypass rule", map[string]any{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallBypassResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallBypassRule
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetFirewallBypass(
		ctx,
		state.TeamID.ValueString(),
		state.ProjectID.ValueString(),
		client.FirewallBypassRule{
			Domain:   state.Domain.ValueString(),
			SourceIp: state.SourceIp.ValueString(),
		},
	)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Firewall Bypass Rule",
			fmt.Sprintf("Could not get Firewall Bypass %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToBypassRule(out)
	tflog.Info(ctx, "read firewall bypass rule", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update does nothing.
func (r *firewallBypassResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallBypassRule
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallBypassResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallBypassRule
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Disable on deletion
	_, err := r.client.RemoveFirewallBypass(ctx,
		state.TeamID.ValueString(),
		state.ProjectID.ValueString(),
		client.FirewallBypassRule{
			Domain:   state.Domain.ValueString(),
			SourceIp: state.SourceIp.ValueString(),
		},
	)
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Firewall Bypass Rule",
			fmt.Sprintf(
				"Could not delete Firewall Bypass Rule %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted firewall bypass rule", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

func splitBypassID(id string) (string, string, string, string, bool) {
	split := strings.SplitN(id, "/", 2)
	if len(split) != 2 {
		return "", "", "", "", false
	}
	teamId := split[0]

	idParts := strings.Split(split[1], "#")
	switch len(idParts) {
	case 2:
		return teamId, idParts[0], "*", idParts[1], true
	case 3:
		return teamId, idParts[0], idParts[1], idParts[2], true
	}
	return "", "", "", "", false
}

func (r *firewallBypassResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, domain, ip, ok := splitBypassID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Firewall Bypass Rule",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/bypass_rule_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetFirewallBypass(
		ctx,
		teamID,
		projectID,
		client.FirewallBypassRule{
			Domain:   domain,
			SourceIp: ip,
		},
	)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Firewall Bypass",
			fmt.Sprintf("Could not get Firewall Bypass %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	result := responseToBypassRule(out)
	tflog.Info(ctx, "import firewall bypass", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
