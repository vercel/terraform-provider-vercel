package vercel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
	"github.com/vercel/vercel"
)

var (
	_ resource.Resource              = &dnsRecordResource{}
	_ resource.ResourceWithConfigure = &dnsRecordResource{}
)

func newDNSRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

type dnsRecordResource struct {
	client *client.Client
	sdk    *vercel.Vercel
}

func (r *dnsRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = providerData.client
	r.sdk = providerData.sdk
}

func (r *dnsRecordResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a DNS Record resource.

DNS records are instructions that live in authoritative DNS servers and provide information about a domain.

~> The ` + "`value` field" + ` must be specified on all DNS record types except ` + "`SRV`" + `. When using ` + "`SRV`" + ` DNS records, the ` + "`srv`" + ` field must be specified.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/custom-domains#dns-records)
        `,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The team ID that the domain and DNS records belong to. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Description:   "The domain name, or zone, that the DNS record should be created beneath.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
			},
			"name": schema.StringAttribute{
				Description: "The subdomain name of the record. This should be an empty string if the rercord is for the root domain.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description:   "The type of DNS record. Available types: " + "`A`" + ", " + "`AAAA`" + ", " + "`ALIAS`" + ", " + "`CAA`" + ", " + "`CNAME`" + ", " + "`MX`" + ", " + "`NS`" + ", " + "`SRV`" + ", " + "`TXT`" + ".",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
				Validators: []validator.String{
					stringvalidator.OneOf("A", "AAAA", "ALIAS", "CAA", "CNAME", "MX", "NS", "SRV", "TXT"),
				},
			},
			"value": schema.StringAttribute{
				// required if any record type apart from SRV.
				Description: "The value of the DNS record. The format depends on the 'type' property.\nFor an 'A' record, this should be a valid IPv4 address.\nFor an 'AAAA' record, this should be an IPv6 address.\nFor 'ALIAS' records, this should be a hostname.\nFor 'CAA' records, this should specify specify which Certificate Authorities (CAs) are allowed to issue certificates for the domain.\nFor 'CNAME' records, this should be a different domain name.\nFor 'MX' records, this should specify the mail server responsible for accepting messages on behalf of the domain name.\nFor 'TXT' records, this can contain arbitrary text.",
				Optional:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "The TTL value in seconds. Must be a number between 60 and 2147483647. If unspecified, it will default to 60 seconds.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(60),
					int64validator.AtMost(2147483647),
				},
			},
			"mx_priority": schema.Int64Attribute{
				Description: "The priority of the MX record. The priority specifies the sequence that an email server receives emails. A smaller value indicates a higher priority.",
				Optional:    true, // required for MX records.
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
					int64validator.AtMost(65535),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment explaining what the DNS record is for.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 500),
				},
			},
			"srv": schema.SingleNestedAttribute{
				Description: "Settings for an SRV record.",
				Optional:    true, // required for SRV records.
				Attributes: map[string]schema.Attribute{
					"weight": schema.Int64Attribute{
						Description: "A relative weight for records with the same priority, higher value means higher chance of getting picked.",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(65535),
						},
					},
					"port": schema.Int64Attribute{
						Description: "The TCP or UDP port on which the service is to be found.",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(65535),
						},
					},
					"priority": schema.Int64Attribute{
						Description: "The priority of the target host, lower value means more preferred.",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(65535),
						},
					},
					"target": schema.StringAttribute{
						Description: "The canonical hostname of the machine providing the service, ending in a dot.",
						Required:    true,
					},
				},
			},
		},
	}
}

// SRV reflect the state terraform stores internally for a nested SRV Record.
type SRV struct {
	Port     types.Int64  `tfsdk:"port"`
	Priority types.Int64  `tfsdk:"priority"`
	Target   types.String `tfsdk:"target"`
	Weight   types.Int64  `tfsdk:"weight"`
}

// DNSRecord reflects the state terraform stores internally for a DNS Record.
type DNSRecord struct {
	ID         types.String `tfsdk:"id"`
	Domain     types.String `tfsdk:"domain"`
	MXPriority types.Int64  `tfsdk:"mx_priority"`
	Name       types.String `tfsdk:"name"`
	SRV        *SRV         `tfsdk:"srv"`
	TTL        types.Int64  `tfsdk:"ttl"`
	TeamID     types.String `tfsdk:"team_id"`
	Type       types.String `tfsdk:"type"`
	Value      types.String `tfsdk:"value"`
	Comment    types.String `tfsdk:"comment"`
}

func (d DNSRecord) toCreateDNSRecordRequest() client.CreateDNSRecordRequest {
	var srv *client.SRV = nil
	if d.Type.ValueString() == "SRV" {
		srv = &client.SRV{
			Port:     d.SRV.Port.ValueInt64(),
			Priority: d.SRV.Priority.ValueInt64(),
			Target:   d.SRV.Target.ValueString(),
			Weight:   d.SRV.Weight.ValueInt64(),
		}
	}

	return client.CreateDNSRecordRequest{
		Domain:     d.Domain.ValueString(),
		MXPriority: d.MXPriority.ValueInt64(),
		Name:       d.Name.ValueString(),
		TTL:        d.TTL.ValueInt64(),
		Type:       d.Type.ValueString(),
		Value:      d.Value.ValueString(),
		SRV:        srv,
		Comment:    d.Comment.ValueString(),
	}
}

func (d DNSRecord) toUpdateRequest() client.UpdateDNSRecordRequest {
	var srv *client.SRVUpdate = nil
	if d.SRV != nil {
		srv = &client.SRVUpdate{
			Port:     d.SRV.Port.ValueInt64Pointer(),
			Priority: d.SRV.Priority.ValueInt64Pointer(),
			Target:   d.SRV.Target.ValueStringPointer(),
			Weight:   d.SRV.Weight.ValueInt64Pointer(),
		}
	}
	return client.UpdateDNSRecordRequest{
		MXPriority: d.MXPriority.ValueInt64Pointer(),
		Name:       d.Name.ValueStringPointer(),
		SRV:        srv,
		TTL:        d.TTL.ValueInt64Pointer(),
		Value:      d.Value.ValueStringPointer(),
		Comment:    d.Comment.ValueString(),
	}
}

func convertResponseToDNSRecord(r client.DNSRecord, value types.String, srv *SRV) (record DNSRecord, err error) {
	record = DNSRecord{
		Domain:     types.StringValue(r.Domain),
		ID:         types.StringValue(r.ID),
		MXPriority: types.Int64Null(),
		Name:       types.StringValue(r.Name),
		TTL:        types.Int64Value(r.TTL),
		TeamID:     toTeamID(r.TeamID),
		Type:       types.StringValue(r.RecordType),
		Comment:    types.StringValue(r.Comment),
	}

	if r.RecordType == "SRV" {
		// The returned 'Value' field is comprised of the various parts of the SRV block.
		// So instead, we want to parse the SRV block back out.
		split := strings.Split(r.Value, " ")
		if len(split) != 4 && len(split) != 3 {
			return record, fmt.Errorf("expected a 3 or 4 part value '{priority} {weight} {port} {target}', but got %s", r.Value)
		}
		priority, err := strconv.Atoi(split[0])
		if err != nil {
			return record, fmt.Errorf("expected SRV record weight to be an int, but got %s", split[0])
		}
		weight, err := strconv.Atoi(split[1])
		if err != nil {
			return record, fmt.Errorf("expected SRV record port to be an int, but got %s", split[1])
		}
		port, err := strconv.Atoi(split[2])
		if err != nil {
			return record, fmt.Errorf("expected SRV record port to be an int, but got %s", split[1])
		}
		target := ""
		if len(split) == 4 {
			target = split[3]
		}
		record.SRV = &SRV{
			Weight:   types.Int64Value(int64(weight)),
			Port:     types.Int64Value(int64(port)),
			Priority: types.Int64Value(int64(priority)),
			Target:   types.StringValue(target),
		}
		// SRV records have no value
		record.Value = types.StringNull()
		if srv != nil && fmt.Sprintf("%s.", srv.Target.ValueString()) == record.SRV.Target.ValueString() {
			record.SRV.Target = srv.Target
		}
		return record, nil
	}

	if r.RecordType == "MX" {
		split := strings.Split(r.Value, " ")
		if len(split) != 2 {
			return record, fmt.Errorf("expected a 2 part value '{priority} {value}', but got %s", r.Value)
		}
		priority, err := strconv.Atoi(split[0])
		if err != nil {
			return record, fmt.Errorf("expected MX priority to be an int, but got %s", split[0])
		}

		record.MXPriority = types.Int64Value(int64(priority))
		record.Value = types.StringValue(split[1])
		if split[1] == fmt.Sprintf("%s.", value.ValueString()) {
			record.Value = value
		}
		return record, nil
	}

	record.Value = types.StringValue(r.Value)
	if r.Value == fmt.Sprintf("%s.", value.ValueString()) {
		record.Value = value
	}
	return record, nil
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
	tflog.Info(ctx, "created DNS Record", map[string]interface{}{
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
	tflog.Info(ctx, "read DNS record", map[string]interface{}{
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
	tflog.Info(ctx, "updated DNS record", map[string]interface{}{
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

	tflog.Info(ctx, "delete DNS record", map[string]interface{}{
		"domain":    state.Domain,
		"record_id": state.ID,
		"team_id":   state.TeamID,
	})
}

// ImportState takes an identifier and reads all the DNS Record information from the Vercel API.
func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, recordID, ok := splitInto1Or2(req.ID)
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
