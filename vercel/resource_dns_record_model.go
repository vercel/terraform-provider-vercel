package vercel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

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
	}
}

func (d DNSRecord) toUpdateRequest() client.UpdateDNSRecordRequest {
	var srv *client.SRVUpdate = nil
	if d.SRV != nil {
		srv = &client.SRVUpdate{
			Port:     toPtr(d.SRV.Port.ValueInt64()),
			Priority: toPtr(d.SRV.Priority.ValueInt64()),
			Target:   toPtr(d.SRV.Target.ValueString()),
			Weight:   toPtr(d.SRV.Weight.ValueInt64()),
		}
	}
	return client.UpdateDNSRecordRequest{
		MXPriority: toInt64Pointer(d.MXPriority),
		Name:       toPtr(d.Name.ValueString()),
		SRV:        srv,
		TTL:        toInt64Pointer(d.TTL),
		Value:      toStrPointer(d.Value),
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
