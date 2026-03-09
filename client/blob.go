package client

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type BlobStore struct {
	Access    string `json:"access"`
	Count     int64  `json:"count"`
	CreatedAt int64  `json:"createdAt"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	Size      int64  `json:"size"`
	Status    string `json:"status"`
	TeamID    string `json:"ownerId"`
	Type      string `json:"type"`
	UpdatedAt int64  `json:"updatedAt"`
}

type blobStoreResponse struct {
	Store BlobStore `json:"store"`
}

type blobStoresResponse struct {
	Stores []BlobStore `json:"stores"`
}

type CreateBlobStoreRequest struct {
	Access string `json:"access,omitempty"`
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
	TeamID string `json:"-"`
}

func (c *Client) CreateBlobStore(ctx context.Context, request CreateBlobStoreRequest) (b BlobStore, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/blob", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	body := string(mustMarshal(request))
	tflog.Info(ctx, "creating blob store", map[string]any{
		"body": body,
		"url":  url,
	})

	var response blobStoreResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   body,
	}, &response)
	if err != nil {
		return b, err
	}

	return response.Store, nil
}

func (c *Client) GetBlobStore(ctx context.Context, storeID, teamID string) (b BlobStore, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s", c.baseURL, storeID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "reading blob store", map[string]any{
		"url": url,
	})

	var response blobStoreResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return b, err
	}

	return response.Store, nil
}

func (c *Client) ListBlobStores(ctx context.Context, teamID string) (stores []BlobStore, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores", c.baseURL)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "listing blob stores", map[string]any{
		"url": url,
	})

	var response blobStoresResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return nil, err
	}

	for _, store := range response.Stores {
		if store.Type == "blob" {
			stores = append(stores, store)
		}
	}

	return stores, nil
}

type UpdateBlobStoreRequest struct {
	Name    string `json:"name"`
	StoreID string `json:"-"`
	TeamID  string `json:"-"`
}

func (c *Client) UpdateBlobStore(ctx context.Context, request UpdateBlobStoreRequest) (b BlobStore, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/blob/%s", c.baseURL, request.StoreID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	body := string(mustMarshal(request))
	tflog.Info(ctx, "updating blob store", map[string]any{
		"body": body,
		"url":  url,
	})

	var response blobStoreResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   body,
	}, &response)
	if err != nil {
		return b, err
	}

	return response.Store, nil
}

func (c *Client) DeleteBlobStore(ctx context.Context, storeID, teamID string) error {
	url := fmt.Sprintf("%s/v1/storage/stores/blob/%s", c.baseURL, storeID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "deleting blob store", map[string]any{
		"url": url,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

type BlobProjectConnection struct {
	EnvVarEnvironments   []string                         `json:"envVarEnvironments"`
	EnvVarPrefix         *string                          `json:"envVarPrefix"`
	ID                   string                           `json:"id"`
	ProductionDeployment *BlobProjectConnectionDeployment `json:"productionDeployment"`
	Project              BlobProject                      `json:"project"`
	ProjectID            string                           `json:"projectId"`
}

type BlobProject struct {
	Framework *string `json:"framework"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
}

type BlobProjectConnectionDeployment struct {
	ID  string  `json:"id"`
	URL *string `json:"url"`
}

type blobStoreConnectionsResponse struct {
	Connections []BlobProjectConnection `json:"connections"`
}

type CreateBlobStoreConnectionRequest struct {
	BlobStoreID  string
	Environments []string `json:"envVarEnvironments"`
	EnvVarPrefix string   `json:"envVarPrefix"`
	ProjectID    string   `json:"projectId"`
	TeamID       string   `json:"-"`
}

func (c *Client) CreateBlobStoreConnection(ctx context.Context, request CreateBlobStoreConnectionRequest) (connection BlobProjectConnection, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s/connections", c.baseURL, request.BlobStoreID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	body := string(mustMarshal(struct {
		EnvVarEnvironments []string `json:"envVarEnvironments"`
		EnvVarPrefix       string   `json:"envVarPrefix"`
		ProjectID          string   `json:"projectId"`
	}{
		EnvVarEnvironments: request.Environments,
		EnvVarPrefix:       request.EnvVarPrefix,
		ProjectID:          request.ProjectID,
	}))

	tflog.Info(ctx, "creating blob store project connection", map[string]any{
		"body": body,
		"url":  url,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   body,
	}, nil)
	if err != nil {
		return connection, err
	}

	connections, err := c.ListBlobStoreConnections(ctx, request.BlobStoreID, request.TeamID)
	if err != nil {
		return connection, err
	}

	for _, candidate := range connections {
		if candidate.ProjectID != request.ProjectID {
			continue
		}

		if sameStringSet(candidate.EnvVarEnvironments, request.Environments) {
			return candidate, nil
		}
	}

	return connection, fmt.Errorf("blob store connection for store %s and project %s was not found after create", request.BlobStoreID, request.ProjectID)
}

func (c *Client) ListBlobStoreConnections(ctx context.Context, storeID, teamID string) (connections []BlobProjectConnection, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s/connections", c.baseURL, storeID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "listing blob store connections", map[string]any{
		"url": url,
	})

	var response blobStoreConnectionsResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response.Connections, nil
}

func (c *Client) GetBlobStoreConnection(ctx context.Context, storeID, connectionID, teamID string) (connection BlobProjectConnection, err error) {
	connections, err := c.ListBlobStoreConnections(ctx, storeID, teamID)
	if err != nil {
		return connection, err
	}

	for _, candidate := range connections {
		if candidate.ID == connectionID {
			return candidate, nil
		}
	}

	return connection, APIError{
		Code:       "not_found",
		Message:    "Blob store project connection not found",
		StatusCode: 404,
	}
}

type UpdateBlobStoreConnectionRequest struct {
	BlobStoreID  string
	ConnectionID string
	Environments []string `json:"envVarEnvironments"`
	EnvVarPrefix string   `json:"envVarPrefix"`
	TeamID       string   `json:"-"`
}

func (c *Client) UpdateBlobStoreConnection(ctx context.Context, request UpdateBlobStoreConnectionRequest) (connection BlobProjectConnection, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s/connections/%s", c.baseURL, request.BlobStoreID, request.ConnectionID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	body := string(mustMarshal(struct {
		EnvVarEnvironments []string `json:"envVarEnvironments"`
		EnvVarPrefix       string   `json:"envVarPrefix"`
	}{
		EnvVarEnvironments: request.Environments,
		EnvVarPrefix:       request.EnvVarPrefix,
	}))

	tflog.Info(ctx, "updating blob store project connection", map[string]any{
		"body": body,
		"url":  url,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   body,
	}, &connection)
	if err != nil {
		return connection, err
	}

	return connection, nil
}

func (c *Client) DeleteBlobStoreConnection(ctx context.Context, storeID, connectionID, teamID string) error {
	url := fmt.Sprintf("%s/v1/storage/stores/%s/connections/%s", c.baseURL, storeID, connectionID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "deleting blob store project connection", map[string]any{
		"url": url,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

type BlobStoreSecrets struct {
	ReadWriteToken string `json:"rwToken"`
}

func (c *Client) GetBlobStoreSecrets(ctx context.Context, storeID, teamID string) (secrets BlobStoreSecrets, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s/secrets", c.baseURL, storeID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "reading blob store secrets", map[string]any{
		"url": url,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &secrets)
	if err != nil {
		return secrets, err
	}

	return secrets, nil
}

func sameStringSet(left, right []string) bool {
	leftCopy := slices.Clone(left)
	rightCopy := slices.Clone(right)
	slices.Sort(leftCopy)
	slices.Sort(rightCopy)
	return slices.Equal(leftCopy, rightCopy)
}
