package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DONE: convert to redis from edge config
type RedisInstance struct {
	Slug          string   `json:"endpoint"`
	ID            string   `json:"id"`
	TeamID        string   `json:"ownerId"`
	Name          string   `json:"name"`
	PrimaryRegion string   `json:"primaryRegion"`
	ReadRegions   []string `json:"readRegions"`
	Eviction      bool     `json:"eviction"`
	Type          string   `json:"type"` // should always be "redis"
}

// DONE: convert to redis from edge config
type CreateRedisInstanceRequest struct {
	Name          string   `json:"name"`
	TeamID        string   `json:"-"`
	PrimaryRegion string   `json:"primaryRegion"`
	ReadRegions   []string `json:"readRegions"`
	Eviction      bool
}

// Generic struct wrappers over responses from storage resources
type Store[T any] struct {
	Store T `json:"store"`
}

type Stores[T any] struct {
	Stores []T `json:"stores"`
}

// DONE: convert to redis
func (c *Client) CreateRedis(ctx context.Context, request CreateRedisInstanceRequest) (e RedisInstance, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/redis?", c.baseURL)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating redis instance", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	var res Store[RedisInstance]
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &res)
	if err == nil {
		e = res.Store
	}
	return e, err
}

// DONE: convert to redis
func (c *Client) GetRedisInstance(ctx context.Context, id, teamID string) (e RedisInstance, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s", c.baseURL, id)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "reading redis instance", map[string]interface{}{
		"url": url,
	})
	var res Store[RedisInstance]
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &res)
	if err == nil {
		e = res.Store
	}
	return e, err
}

// DONE: convert to redis
type UpdateRedisInstanceRequest struct {
	Slug          string   `json:"name"`
	ID            string   `json:"id"`
	TeamID        string   `json:"-"`
	PrimaryRegion string   `json:"primaryRegion"`
	ReadRegions   []string `json:"readRegions"`
	Eviction      bool
}

// DONE: convert to redis
func (c *Client) UpdateRedis(ctx context.Context, request UpdateRedisInstanceRequest) (e RedisInstance, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores/%s", c.baseURL, request.ID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	tflog.Trace(ctx, "updating redis instance", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	var res Store[RedisInstance]
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   payload,
	}, &res)
	if err == nil {
		e = res.Store
	}
	return e, err
}

// DONE: convert to redis
func (c *Client) DeleteRedis(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/storage/stores/redis/%s", c.baseURL, id)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "deleting redis instance", map[string]interface{}{
		"url": url,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

// DONE: convert to redis from edge config
func (c *Client) ListRedisInstances(ctx context.Context, teamID string) (e []RedisInstance, err error) {
	url := fmt.Sprintf("%s/v1/storage/stores", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "listing redis instances", map[string]interface{}{
		"url": url,
	})
	var res Stores[RedisInstance]
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &res)
	// filter out PG instances, currently no way to request just redis instances
	for _, instance := range res.Stores {
		if instance.Type == "redis" {
			e = append(e, instance)
		}
	}

	return e, err
}
