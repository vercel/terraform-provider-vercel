package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type DsyncGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) GetDsyncGroups(ctx context.Context, TeamID string) ([]DsyncGroup, error) {
	var allGroups []DsyncGroup
	var after *string

	for {
		url := fmt.Sprintf("%s/teams/%s/dsync/groups", c.baseURL, TeamID)
		if after != nil {
			url = fmt.Sprintf("%s?after=%s", url, *after)
		}
		tflog.Info(ctx, "getting dsync groups", map[string]any{
			"url": url,
		})

		var response struct {
			Groups     []DsyncGroup `json:"groups"`
			Pagination struct {
				Before *string `json:"before"`
				After  *string `json:"after"`
			} `json:"pagination"`
		}
		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "GET",
			url:    url,
			body:   "",
		}, &response)
		if err != nil {
			return nil, err
		}

		allGroups = append(allGroups, response.Groups...)

		if response.Pagination.After == nil {
			break
		}
		after = response.Pagination.After
	}

	return allGroups, nil
}
