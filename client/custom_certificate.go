package client

import (
	"context"
	"fmt"
)

type UploadCustomCertificateRequest struct {
	TeamID                          string `json:"-"`
	PrivateKey                      string `json:"key"`
	Certificate                     string `json:"cert"`
	CertificateAuthorityCertificate string `json:"ca"`
}

type CertificateResponse struct {
	ID string `json:"id"`
}

func (c *Client) UploadCustomCertificate(ctx context.Context, request UploadCustomCertificateRequest) (cr CertificateResponse, err error) {
	url := fmt.Sprintf("%s/v8/certs", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   payload,
	}, &cr)
	return cr, err
}

type GetCustomCertificateRequest struct {
	TeamID string `json:"-"`
	ID     string `json:"-"`
}

func (c *Client) GetCustomCertificate(ctx context.Context, request GetCustomCertificateRequest) (cr CertificateResponse, err error) {
	url := fmt.Sprintf("%s/v8/certs/%s", c.baseURL, request.ID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &cr)
	return cr, err
}

type DeleteCustomCertificateRequest struct {
	TeamID string `json:"-"`
	ID     string `json:"-"`
}

func (c *Client) DeleteCustomCertificate(ctx context.Context, request DeleteCustomCertificateRequest) error {
	url := fmt.Sprintf("%s/v8/certs/%s", c.baseURL, request.ID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
	return err
}
