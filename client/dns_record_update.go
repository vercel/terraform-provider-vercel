package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SRVUpdate defines the updatable fields within an SRV block of a DNS record.
type SRVUpdate struct {
	Port     *int64  `json:"port"`
	Priority *int64  `json:"priority"`
	Target   *string `json:"target"`
	Weight   *int64  `json:"weight"`
}

// UpdateDNSRecordRequest defines the structure of the request body for updating a DNS record.
type UpdateDNSRecordRequest struct {
	MXPriority *int64     `json:"mxPriority,omitempty"`
	Name       *string    `json:"name,omitempty"`
	SRV        *SRVUpdate `json:"srv,omitempty"`
	TTL        *int64     `json:"ttl,omitempty"`
	Value      *string    `json:"value,omitempty"`
}

// UpdateDNSRecord updates a DNS record for a specified domain name within Vercel.
func (c *Client) UpdateDNSRecord(ctx context.Context, teamID, recordID string, request UpdateDNSRecordRequest) (r DNSRecord, err error) {
	url := fmt.Sprintf("%s/v4/domains/records/%s", c.baseURL, recordID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "updating DNS record", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
