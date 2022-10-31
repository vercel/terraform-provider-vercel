package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

func newDNSRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

type dnsRecordResource struct {
	client *client.Client
}

func (r *dnsRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsRecordResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a DNS Record resource.

DNS records are instructions that live in authoritative DNS servers and provide information about a domain.

~> The ` + "`value` field" + ` must be specified on all DNS record types except ` + "`SRV`" + `. When using ` + "`SRV`" + ` DNS records, the ` + "`srv`" + ` field must be specified.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/custom-domains#dns-records)
        `,
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"team_id": {
				Optional:      true,
				Computed:      true,
				Description:   "The team ID that the domain and DNS records belong to.",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace(), resource.UseStateForUnknown()},
				Type:          types.StringType,
			},
			"domain": {
				Description:   "The domain name, or zone, that the DNS record should be created beneath.",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Required:      true,
				Type:          types.StringType,
			},
			"name": {
				Description: "The subdomain name of the record. This should be an empty string if the rercord is for the root domain.",
				Required:    true,
				Type:        types.StringType,
			},
			"type": {
				Description:   "The type of DNS record. Available types: " + "`A`" + ", " + "`AAAA`" + ", " + "`ALIAS`" + ", " + "`CAA`" + ", " + "`CNAME`" + ", " + "`MX`" + ", " + "`NS`" + ", " + "`SRV`" + ", " + "`TXT`" + ".",
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.RequiresReplace()},
				Required:      true,
				Type:          types.StringType,
				Validators: []tfsdk.AttributeValidator{
					stringOneOf("A", "AAAA", "ALIAS", "CAA", "CNAME", "MX", "NS", "SRV", "TXT"),
				},
			},
			"value": {
				// required if any record type apart from SRV.
				Description: "The value of the DNS record. The format depends on the 'type' property.\nFor an 'A' record, this should be a valid IPv4 address.\nFor an 'AAAA' record, this should be an IPv6 address.\nFor 'ALIAS' records, this should be a hostname.\nFor 'CAA' records, this should specify specify which Certificate Authorities (CAs) are allowed to issue certificates for the domain.\nFor 'CNAME' records, this should be a different domain name.\nFor 'MX' records, this should specify the mail server responsible for accepting messages on behalf of the domain name.\nFor 'TXT' records, this can contain arbitrary text.",
				Optional:    true,
				Type:        types.StringType,
			},
			"ttl": {
				Description: "The TTL value in seconds. Must be a number between 60 and 2147483647. If unspecified, it will default to 60 seconds.",
				Optional:    true,
				Computed:    true,
				Type:        types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					int64GreaterThan(60),
					int64LessThan(2147483647),
				},
			},
			"mx_priority": {
				Description: "The priority of the MX record. The priority specifies the sequence that an email server receives emails. A smaller value indicates a higher priority.",
				Optional:    true, // required for MX records.
				Type:        types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					int64GreaterThan(0),
					int64LessThan(65535),
				},
			},
			"srv": {
				Description: "Settings for an SRV record.",
				Optional:    true, // required for SRV records.
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"weight": {
						Description: "A relative weight for records with the same priority, higher value means higher chance of getting picked.",
						Type:        types.Int64Type,
						Required:    true,
						Validators: []tfsdk.AttributeValidator{
							int64GreaterThan(0),
							int64LessThan(65535),
						},
					},
					"port": {
						Description: "The TCP or UDP port on which the service is to be found.",
						Type:        types.Int64Type,
						Required:    true,
						Validators: []tfsdk.AttributeValidator{
							int64GreaterThan(0),
							int64LessThan(65535),
						},
					},
					"priority": {
						Description: "The priority of the target host, lower value means more preferred.",
						Type:        types.Int64Type,
						Required:    true,
						Validators: []tfsdk.AttributeValidator{
							int64GreaterThan(0),
							int64LessThan(65535),
						},
					},
					"target": {
						Description: "The canonical hostname of the machine providing the service, ending in a dot.",
						Type:        types.StringType,
						Required:    true,
					},
				}),
			},
		},
	}, nil
}

// ValidateConfig validates the Resource configuration.
func (r *dnsRecordResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config DNSRecord
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Type.ValueString() == "SRV" && config.SRV == nil {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"A DNS Record type of 'SRV' requires the `srv` attribute to be set",
		)
	}

	if config.Type.ValueString() != "SRV" && config.Value.IsNull() {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			fmt.Sprintf("The `value` attribute must be set on records of `type` '%s'", config.Type.ValueString()),
		)
	}

	if config.Type.ValueString() == "SRV" && !config.Value.IsNull() {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"The `value` attribute should not be set on records of `type` 'SRV'",
		)
	}

	if config.Type.ValueString() != "SRV" && config.SRV != nil {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"The `srv` attribute should only be set on records of `type` 'SRV'",
		)
	}

	if config.Type.ValueString() != "MX" && !config.MXPriority.IsNull() {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"The `mx_priority` attribute should only be set on records of `type` 'MX'",
		)
	}

	if config.Type.ValueString() == "MX" && config.MXPriority.IsNull() {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"A DNS Record type of 'MX' requires the `mx_priority` attribute to be set",
		)
	}
}

// Create will create a DNS record within Vercel by calling the Vercel API.
// This is called automatically by the provider when a new resource should be created.
func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSRecord
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateDNSRecord(ctx, plan.TeamID.ValueString(), plan.toCreateDNSRecordRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating DNS Record",
			"Could not create DNS Record, unexpected error: "+err.Error(),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, plan.Value, plan.SRV)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing DNS Record response",
			"Could not parse create DNS Record response, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Trace(ctx, "created DNS Record", map[string]interface{}{
		"team_id":   result.TeamID,
		"record_id": result.ID,
		"domain":    result.Domain,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a DNS record from the vercel API and provide terraform with information about it.
// It is called by the provider whenever values should be read to update state.
func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSRecord
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetDNSRecord(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DNS Record",
			fmt.Sprintf("Could not read DNS Record %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, state.Value, state.SRV)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing DNS Record response",
			"Could not parse create DNS Record response, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Trace(ctx, "read DNS record", map[string]interface{}{
		"team_id":   result.TeamID,
		"record_id": result.ID,
		"domain":    result.Domain,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update will update a DNS record via the vercel API.
func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSRecord
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DNSRecord
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateDNSRecord(
		ctx,
		plan.TeamID.ValueString(),
		state.ID.ValueString(),
		plan.toUpdateRequest(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating DNS Record",
			fmt.Sprintf(
				"Could not update DNS Record %s for domain %s, unexpected error: %s",
				state.ID.ValueString(),
				state.Domain.ValueString(),
				err,
			),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, plan.Value, plan.SRV)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing DNS Record response",
			"Could not parse create DNS Record response, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Trace(ctx, "updated DNS record", map[string]interface{}{
		"team_id":   result.TeamID,
		"record_id": result.ID,
		"domain":    result.Domain,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete a DNS record from within terraform.
func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSRecord
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDNSRecord(ctx, state.Domain.ValueString(), state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		// The DNS Record is already gone - do nothing.
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting DNS Record",
			fmt.Sprintf(
				"Could not delete DNS Record %s for domain %s, unexpected error: %s",
				state.ID.ValueString(),
				state.Domain.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Trace(ctx, "delete DNS record", map[string]interface{}{
		"domain":    state.Domain,
		"record_id": state.ID,
		"team_id":   state.TeamID,
	})
}

// ImportState takes an identifier and reads all the DNS Record information from the Vercel API.
func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, recordID, ok := splitID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing DNS Record",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/record_id\" or \"record_id\"", req.ID),
		)
	}

	out, err := r.client.GetDNSRecord(ctx, recordID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DNS Record",
			fmt.Sprintf("Could not get DNS Record %s %s, unexpected error: %s",
				teamID,
				recordID,
				err,
			),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, types.String{}, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error processing DNS Record response",
			fmt.Sprintf("Could not process DNS Record API response %s %s, unexpected error: %s",
				teamID,
				recordID,
				err,
			),
		)
	}

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
