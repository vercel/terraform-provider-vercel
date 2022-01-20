package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type DomainResponse struct {
	Suffix      bool     `json:"suffix"`
	Verified    bool     `json:"verified"`
	Nameservers []string `json:"nameservers"`
	Creator     struct {
		Username         string  `json:"username"`
		Email            string  `json:"email"`
		CustomerID       *string `json:"customerId"`
		ID               string  `json:"id"`
		IsDomainReseller *bool   `json:"isDomainReseller"`
	} `json:"creator"`
	ID                string `json:"id"`
	Name              string `json:"name"`
	CreatedAt         int64  `json:"createdAt"`
	ExpiresAt         *int64 `json:"expiresAt"`
	BoughtAt          *int64 `json:"boughtAt"`
	TransferredAt     *int64 `json:"transferredAt"`
	TransferStartedAt *int64 `json:"transferStartedAt"`
	ServiceType       string `json:"serviceType"`
	Renew             *bool  `json:"renew"`
}

func (c *Client) GetDomain(ctx context.Context, domain, teamID string) (r DomainResponse, err error) {
	url := fmt.Sprintf("%s/v5/domains/%s", c.baseURL, domain)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, strings.NewReader(""))
	if err != nil {
		return r, err
	}

	domainResponse := struct {
		Domain DomainResponse `json:"domain"`
	}{}
	err = c.doRequest(req, &domainResponse)
	if err != nil {
		return domainResponse.Domain, fmt.Errorf("error getting domain %s: %w", domain, err)
	}

	return domainResponse.Domain, nil
}
