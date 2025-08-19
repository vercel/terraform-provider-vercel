package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type recommendedValue struct {
	Rank  int      `json:"rank"`
	Value []string `json:"value"`
}

type recommendedCNAMEValue struct {
	Rank  int    `json:"rank"`
	Value string `json:"value"`
}

type domainConfigAPIResponse struct {
	RecommendedCNAME []recommendedCNAMEValue `json:"recommendedCNAME"`
	RecommendedIPv4  []recommendedValue      `json:"recommendedIPv4"`
}

type DomainConfigResponse struct {
	RecommendedCNAME string
	RecommendedIPv4  []string
}

func (c *Client) GetDomainConfig(ctx context.Context, domain, projectIdOrName, teamID string) (DomainConfigResponse, error) {
	url := fmt.Sprintf("%s/v6/domains/%s/config?projectIdOrName=%s", c.baseURL, domain, projectIdOrName)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "getting domain config", map[string]any{
		"url":       url,
		"domain":    domain,
		"projectIdOrName": projectIdOrName,
	})

	var apiResponse domainConfigAPIResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &apiResponse)
	if err != nil {
		return DomainConfigResponse{}, fmt.Errorf("unable to get domain config: %w", err)
	}

	response := DomainConfigResponse{}

	for _, cname := range apiResponse.RecommendedCNAME {
		if cname.Rank == 1 {
			response.RecommendedCNAME = cname.Value
			break
		}
	}

	for _, ipv4 := range apiResponse.RecommendedIPv4 {
		if ipv4.Rank == 1 {
			response.RecommendedIPv4 = ipv4.Value
			break
		}
	}

	return response, nil
}
