package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithConfigure   = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

func newDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *client.Client
}

func (r *domainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a domain resource.
func (r *domainResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Domain resource.

This adds an existing apex domain to a Vercel account or team. This is distinct from a ` + "`vercel_project_domain`" + `,
which associates a domain name with a specific project.`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "The name of the domain to add.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the domain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"cdn_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the domain has the Vercel Edge Network enabled or not. This can only be set when the domain is created.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplaceIfConfigured(), boolplanmodifier.UseStateForUnknown()},
			},
			"zone": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether a DNS zone should be created for the domain on Vercel. Set to `true` if using Vercel nameservers.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Description:   "The unique identifier of the domain.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"verified": schema.BoolAttribute{
				Description: "Whether the domain has its ownership verified.",
				Computed:    true,
			},
			"nameservers": schema.ListAttribute{
				Description: "A list of the current nameservers of the domain.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"intended_nameservers": schema.ListAttribute{
				Description: "A list of the intended nameservers for the domain to point to Vercel DNS.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"custom_nameservers": schema.ListAttribute{
				Description: "A list of custom nameservers for the domain to point to. Only applies to domains purchased with Vercel.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.Int64Attribute{
				Description: "Timestamp in milliseconds when the domain was created in the registry.",
				Computed:    true,
			},
			"expires_at": schema.Int64Attribute{
				Description: "Timestamp in milliseconds at which the domain is set to expire. Null if not bought with Vercel.",
				Computed:    true,
			},
			"bought_at": schema.Int64Attribute{
				Description: "Timestamp in milliseconds when the domain was purchased, if it was purchased through Vercel.",
				Computed:    true,
			},
		},
	}
}

type Domain struct {
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	TeamID              types.String `tfsdk:"team_id"`
	CDNEnabled          types.Bool   `tfsdk:"cdn_enabled"`
	Zone                types.Bool   `tfsdk:"zone"`
	Verified            types.Bool   `tfsdk:"verified"`
	Nameservers         types.List   `tfsdk:"nameservers"`
	IntendedNameservers types.List   `tfsdk:"intended_nameservers"`
	CustomNameservers   types.List   `tfsdk:"custom_nameservers"`
	CreatedAt           types.Int64  `tfsdk:"created_at"`
	ExpiresAt           types.Int64  `tfsdk:"expires_at"`
	BoughtAt            types.Int64  `tfsdk:"bought_at"`
}

func responseToDomain(ctx context.Context, out client.Domain) (Domain, diag.Diagnostics) {
	var diags diag.Diagnostics

	nameservers, diag := types.ListValueFrom(ctx, types.StringType, out.Nameservers)
	diags.Append(diag...)
	intendedNameservers, diag := types.ListValueFrom(ctx, types.StringType, out.IntendedNameservers)
	diags.Append(diag...)
	customNameservers, diag := types.ListValueFrom(ctx, types.StringType, out.CustomNameservers)
	diags.Append(diag...)

	return Domain{
		Name:                types.StringValue(out.Name),
		ID:                  types.StringValue(out.ID),
		TeamID:              toTeamID(out.TeamID),
		CDNEnabled:          types.BoolValue(out.CDNEnabled),
		Zone:                types.BoolValue(out.Zone),
		Verified:            types.BoolValue(out.Verified),
		Nameservers:         nameservers,
		IntendedNameservers: intendedNameservers,
		CustomNameservers:   customNameservers,
		CreatedAt:           types.Int64PointerValue(out.CreatedAt),
		ExpiresAt:           types.Int64PointerValue(out.ExpiresAt),
		BoughtAt:            types.Int64PointerValue(out.BoughtAt),
	}, diags
}

// Create adds an existing apex domain to Vercel.
func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Domain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateDomain(ctx, client.CreateDomainRequest{
		Name:       plan.Name.ValueString(),
		CDNEnabled: plan.CDNEnabled.ValueBoolPointer(),
		Zone:       plan.Zone.ValueBoolPointer(),
		TeamID:     plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Domain",
			"Could not create Domain, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToDomain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created Domain", map[string]any{
		"team_id": result.TeamID.ValueString(),
		"domain":  result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

// Read reads domain information from the Vercel API and updates terraform with it.
func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Domain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetDomain(ctx, state.Name.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Domain",
			fmt.Sprintf("Could not get Domain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.Name.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToDomain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read Domain", map[string]any{
		"team_id": result.TeamID.ValueString(),
		"domain":  result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

// Update updates the DNS zone of an existing apex domain. All other attributes require replacement.
func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Domain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateDomain(ctx, client.UpdateDomainRequest{
		Name:   plan.Name.ValueString(),
		Zone:   plan.Zone.ValueBoolPointer(),
		TeamID: plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Domain",
			fmt.Sprintf("Could not update Domain %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.Name.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := responseToDomain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "updated Domain", map[string]any{
		"team_id": result.TeamID.ValueString(),
		"domain":  result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

// Delete removes an apex domain from Vercel.
func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Domain
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDomain(ctx, state.Name.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Domain",
			fmt.Sprintf("Could not delete Domain %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.Name.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Domain", map[string]any{
		"team_id": state.TeamID.ValueString(),
		"domain":  state.Name.ValueString(),
	})
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, name, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Domain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/domain_name\" or \"domain_name\"", req.ID),
		)
		return
	}

	out, err := r.client.GetDomain(ctx, name, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Domain",
			fmt.Sprintf("Could not get Domain %s %s, unexpected error: %s",
				teamID,
				name,
				err,
			),
		)
		return
	}

	result, diags := responseToDomain(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "imported Domain", map[string]any{
		"team_id": result.TeamID.ValueString(),
		"domain":  result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
