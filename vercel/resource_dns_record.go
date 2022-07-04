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

type resourceDNSRecordType struct{}

func (r resourceDNSRecordType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides a DNS Record resource.

DNS records are instructions that live in authoritative DNS servers and provide information about a domain.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/custom-domains#dns-records)
        `,
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"team_id": {
				Optional:      true,
				Description:   "The team ID that the domain and DNS records belong to.",
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Type:          types.StringType,
			},
			"domain": {
				Description:   "The domain name, or zone, that the DNS record should be created beneath.",
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Required:      true,
				Type:          types.StringType,
			},
			"name": {
				Description: "The subdomain name of the record. This should be an empty string if the rercord is for the root domain.",
				Required:    true,
				Type:        types.StringType,
			},
			"type": {
				Description:   "The type of DNS record.",
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
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

// NewResource instantiates a new Resource of this ResourceType.
func (r resourceDNSRecordType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceDNSRecord{
		p: *(p.(*provider)),
	}, nil
}

type resourceDNSRecord struct {
	p provider
}

// ValidateConfig validates the Resource configuration.
func (r resourceDNSRecord) ValidateConfig(ctx context.Context, req tfsdk.ValidateResourceConfigRequest, resp *tfsdk.ValidateResourceConfigResponse) {
	var config DNSRecord
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Type.Value == "SRV" && config.SRV == nil {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"A DNS Record type of 'SRV' requires the `srv` attribute to be set",
		)
	}

	if config.Type.Value == "SRV" && !config.Value.Null {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"The `value` attribute should not be set on records of `type` 'SRV'",
		)
	}

	if config.Type.Value != "SRV" && config.SRV != nil {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"The `srv` attribute should only be set on records of `type` 'SRV'",
		)
	}

	if config.Type.Value != "MX" && !config.MXPriority.Null {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"The `mx_priority` attribute should only be set on records of `type` 'MX'",
		)
	}

	if config.Type.Value == "MX" && config.MXPriority.Null {
		resp.Diagnostics.AddError(
			"DNS Record Invalid",
			"A DNS Record type of 'MX' requires the `mx_priority` attribute to be set",
		)
	}
}

// Create will create a DNS record within Vercel by calling the Vercel API.
// This is called automatically by the provider when a new resource should be created.
func (r resourceDNSRecord) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var plan DNSRecord
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.CreateDNSRecord(ctx, plan.TeamID.Value, plan.toCreateDNSRecordRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating DNS Record",
			"Could not create DNS Record, unexpected error: "+err.Error(),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, plan.TeamID, plan.Value, plan.SRV)
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
func (r resourceDNSRecord) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state DNSRecord
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetDNSRecord(ctx, state.ID.Value, state.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DNS Record",
			fmt.Sprintf("Could not read DNS Record %s %s, unexpected error: %s",
				state.TeamID.Value,
				state.ID.Value,
				err,
			),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, state.TeamID, state.Value, state.SRV)
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
func (r resourceDNSRecord) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
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

	out, err := r.p.client.UpdateDNSRecord(
		ctx,
		plan.TeamID.Value,
		state.ID.Value,
		plan.toUpdateRequest(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating DNS Record",
			fmt.Sprintf(
				"Could not update DNS Record %s for domain %s, unexpected error: %s",
				state.ID.Value,
				state.Domain.Value,
				err,
			),
		)
		return
	}

	result, err := convertResponseToDNSRecord(out, plan.TeamID, plan.Value, plan.SRV)
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
func (r resourceDNSRecord) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state DNSRecord
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.p.client.DeleteDNSRecord(ctx, state.Domain.Value, state.ID.Value, state.TeamID.Value)
	var apiErr client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		// The DNS Record is already gone - do nothing.
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting DNS Record",
			fmt.Sprintf(
				"Could not delete DNS Record %s for domain %s, unexpected error: %s",
				state.ID.Value,
				state.Domain.Value,
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
func (r resourceDNSRecord) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	teamID, recordID, ok := splitID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing DNS Record",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/record_id\" or \"record_id\"", req.ID),
		)
	}

	out, err := r.p.client.GetDNSRecord(ctx, recordID, teamID)
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
	stringTypeTeamID := types.String{Value: teamID}
	if teamID == "" {
		stringTypeTeamID.Null = true
	}

	result, err := convertResponseToDNSRecord(out, stringTypeTeamID, types.String{}, nil)
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
