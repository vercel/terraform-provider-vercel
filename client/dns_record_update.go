package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type SRVUpdate struct {
	Port     *int64  `json:"port"`
	Priority *int64  `json:"priority"`
	Target   *string `json:"target"`
	Weight   *int64  `json:"weight"`
}

type UpdateDNSRecordRequest struct {
	MXPriority *int64     `json:"mxPriority,omitempty"`
	Name       *string    `json:"name,omitempty"`
	SRV        *SRVUpdate `json:"srv,omitempty"`
	TTL        *int64     `json:"ttl,omitempty"`
	Value      *string    `json:"value,omitempty"`
}

func (c *Client) UpdateDNSRecord(ctx context.Context, teamID, recordID string, request UpdateDNSRecordRequest) (r DNSRecord, err error) {
	url := fmt.Sprintf("%s/v4/domains/records/%s", c.baseURL, recordID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
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
	return r, err
}
