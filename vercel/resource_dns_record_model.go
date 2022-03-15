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
	Port     int64  `tfsdk:"port"`
	Priority int64  `tfsdk:"priority"`
	Target   string `tfsdk:"target"`
	Weight   int64  `tfsdk:"weight"`
}

// DNSRecord reflects the state terraform stores internally for a DNS Record.
type DNSRecord struct {
	ID         types.String `tfsdk:"id"`
	Domain     string       `tfsdk:"domain"`
	MXPriority types.Int64  `tfsdk:"mx_priority"`
	Name       string       `tfsdk:"name"`
	SRV        *SRV         `tfsdk:"srv"`
	TTL        types.Int64  `tfsdk:"ttl"`
	TeamID     types.String `tfsdk:"team_id"`
	Type       string       `tfsdk:"type"`
	Value      types.String `tfsdk:"value"`
}

func (d DNSRecord) toCreateDNSRecordRequest() client.CreateDNSRecordRequest {
	var srv *client.SRV = nil
	if d.Type == "SRV" {
		srv = &client.SRV{
			Port:     d.SRV.Port,
			Priority: d.SRV.Priority,
			Target:   d.SRV.Target,
			Weight:   d.SRV.Weight,
		}
	}
	return client.CreateDNSRecordRequest{
		Domain:     d.Domain,
		MXPriority: d.MXPriority.Value,
		Name:       d.Name,
		TTL:        d.TTL.Value,
		Type:       d.Type,
		Value:      d.Value.Value,
		SRV:        srv,
	}
}

func (d DNSRecord) toUpdateRequest() client.UpdateDNSRecordRequest {
	var srv *client.SRVUpdate = nil
	if d.SRV != nil {
		srv = &client.SRVUpdate{
			Port:     &d.SRV.Port,
			Priority: &d.SRV.Priority,
			Target:   &d.SRV.Target,
			Weight:   &d.SRV.Weight,
		}
	}
	return client.UpdateDNSRecordRequest{
		MXPriority: toInt64Pointer(d.MXPriority),
		Name:       &d.Name,
		SRV:        srv,
		TTL:        toInt64Pointer(d.TTL),
		Value:      toStrPointer(d.Value),
	}
}

func convertResponseToDNSRecord(r client.DNSRecord, tid types.String, value types.String, srv *SRV) (record DNSRecord, err error) {
	teamID := types.String{Value: tid.Value}
	if tid.Unknown || tid.Null {
		teamID.Null = true
	}

	record = DNSRecord{
		Domain:     r.Domain,
		ID:         types.String{Value: r.ID},
		MXPriority: types.Int64{Null: true},
		Name:       r.Name,
		TTL:        types.Int64{Value: r.TTL},
		TeamID:     teamID,
		Type:       r.RecordType,
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
			Weight:   int64(weight),
			Port:     int64(port),
			Priority: int64(priority),
			Target:   target,
		}
		// SRV records have no value
		record.Value = types.String{Null: true}
		if srv != nil && fmt.Sprintf("%s.", srv.Target) == record.SRV.Target {
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

		record.MXPriority = types.Int64{Value: int64(priority)}
		record.Value = types.String{Value: split[1]}
		if split[1] == fmt.Sprintf("%s.", value.Value) {
			record.Value = value
		}
		return record, nil
	}

	record.Value = types.String{Value: r.Value}
	if r.Value == fmt.Sprintf("%s.", value.Value) {
		record.Value = value
	}
	return record, nil
}
