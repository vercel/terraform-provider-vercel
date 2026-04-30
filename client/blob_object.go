package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const blobDataPlaneAPIVersion = "12"
const blobDataPlaneURL = "https://vercel.com/api/blob"
const blobDataPlaneTransientAttempts = 8
const blobDataPlaneProvisioningAttempts = 6
const blobDataPlaneProvisioningRetryBase = 200 * time.Millisecond

type BlobObject struct {
	CacheControl       string `json:"cacheControl"`
	ContentDisposition string `json:"contentDisposition"`
	ContentType        string `json:"contentType"`
	DownloadURL        string `json:"downloadUrl"`
	ETag               string `json:"etag"`
	Pathname           string `json:"pathname"`
	Size               int64  `json:"size"`
	UploadedAt         string `json:"uploadedAt"`
	URL                string `json:"url"`
}

type GetBlobObjectRequest struct {
	Pathname string
	StoreID  string
	TeamID   string
}

type PutBlobObjectRequest struct {
	Body               []byte
	CacheControlMaxAge int64
	ContentType        string
	Pathname           string
	StoreID            string
	TeamID             string
}

func (c *Client) GetBlobObject(ctx context.Context, request GetBlobObjectRequest) (object BlobObject, err error) {
	query := url.Values{}
	query.Set("url", request.Pathname)
	endpoint := fmt.Sprintf("%s?%s", blobDataPlaneURL, query.Encode())

	tflog.Info(ctx, "reading blob object", map[string]any{
		"pathname": request.Pathname,
		"store_id": request.StoreID,
		"url":      endpoint,
	})

	requestID := blobDataPlaneRequestID(request.StoreID)
	headers := c.blobDataPlaneHeaders(request.StoreID, request.TeamID)
	headers["x-api-version"] = blobDataPlaneAPIVersion
	headers["x-api-blob-request-id"] = requestID

	for attempt := 1; attempt <= blobDataPlaneTransientAttempts; attempt++ {
		headers["x-api-blob-request-attempt"] = strconv.Itoa(attempt - 1)
		err = c.doRequest(clientRequest{
			ctx:     ctx,
			method:  "GET",
			url:     endpoint,
			headers: headers,
		}, &object)
		if err == nil {
			object.ETag = normalizeBlobObjectETag(object.ETag)
			return object, nil
		}

		maxAttempts := blobDataPlaneRetryMaxAttempts(err)
		if maxAttempts == 0 || attempt == maxAttempts {
			return object, err
		}

		time.Sleep(blobDataPlaneProvisioningRetryBase * time.Duration(1<<(attempt-1)))
	}

	return object, err
}

func (c *Client) PutBlobObject(ctx context.Context, request PutBlobObjectRequest) (object BlobObject, err error) {
	store, err := c.GetBlobStore(ctx, request.StoreID, request.TeamID)
	if err != nil {
		return object, err
	}

	query := url.Values{}
	query.Set("pathname", request.Pathname)
	endpoint := fmt.Sprintf("%s?%s", blobDataPlaneURL, query.Encode())

	headers := c.blobDataPlaneHeaders(request.StoreID, request.TeamID)
	headers["x-add-random-suffix"] = "0"
	headers["x-allow-overwrite"] = "1"
	headers["x-api-version"] = blobDataPlaneAPIVersion
	headers["x-api-blob-request-id"] = blobDataPlaneRequestID(request.StoreID)
	headers["x-vercel-blob-access"] = store.Access
	if request.ContentType != "" {
		headers["x-content-type"] = request.ContentType
	}
	if request.CacheControlMaxAge > 0 {
		headers["x-cache-control-max-age"] = strconv.FormatInt(request.CacheControlMaxAge, 10)
	}

	tflog.Info(ctx, "writing blob object", map[string]any{
		"pathname": request.Pathname,
		"store_id": request.StoreID,
		"url":      endpoint,
	})

	for attempt := 1; attempt <= blobDataPlaneTransientAttempts; attempt++ {
		headers["x-api-blob-request-attempt"] = strconv.Itoa(attempt - 1)
		err = c.doRequest(clientRequest{
			ctx:       ctx,
			method:    "PUT",
			url:       endpoint,
			bodyBytes: request.Body,
			headers:   headers,
		}, &object)
		if err == nil {
			object.ETag = normalizeBlobObjectETag(object.ETag)
			if request.CacheControlMaxAge > 0 && object.CacheControl == "" {
				object.CacheControl = fmt.Sprintf("public, max-age=%d", request.CacheControlMaxAge)
			}
			if object.Size == 0 && len(request.Body) > 0 {
				object.Size = int64(len(request.Body))
			}
			if object.UploadedAt == "" {
				object.UploadedAt = time.Now().UTC().Format(time.RFC3339)
			}
			return object, nil
		}

		maxAttempts := blobDataPlaneRetryMaxAttempts(err)
		if maxAttempts == 0 || attempt == maxAttempts {
			return object, err
		}

		time.Sleep(blobDataPlaneProvisioningRetryBase * time.Duration(1<<(attempt-1)))
	}

	return object, err
}

func (c *Client) DeleteBlobObject(ctx context.Context, storeID, pathname, teamID string) error {
	endpoint := fmt.Sprintf("%s/delete", blobDataPlaneURL)
	body := string(mustMarshal(struct {
		URLs []string `json:"urls"`
	}{
		URLs: []string{pathname},
	}))

	tflog.Info(ctx, "deleting blob object", map[string]any{
		"pathname": pathname,
		"store_id": storeID,
		"url":      endpoint,
	})

	return c.doRequest(clientRequest{
		ctx:         ctx,
		method:      "POST",
		url:         endpoint,
		body:        body,
		contentType: "application/json",
		headers:     c.blobDataPlaneHeaders(storeID, teamID),
	}, nil)
}

func (c *Client) blobDataPlaneHeaders(storeID, teamID string) map[string]string {
	headers := map[string]string{
		"x-api-version":          blobDataPlaneAPIVersion,
		"x-vercel-blob-store-id": strings.TrimPrefix(storeID, "store_"),
	}

	if resolvedTeamID := c.TeamID(teamID); resolvedTeamID != "" {
		headers["x-vercel-blob-team-id"] = resolvedTeamID
	}

	return headers
}

func normalizeBlobObjectETag(etag string) string {
	return strings.Trim(etag, "\"")
}

func blobDataPlaneRetryMaxAttempts(err error) int {
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		return 0
	}

	switch apiErr.Code {
	case "not_found", "store_not_found":
		return blobDataPlaneProvisioningAttempts
	case "internal_server_error", "service_unavailable", "unknown_error":
		return blobDataPlaneTransientAttempts
	default:
		return 0
	}
}

func blobDataPlaneRequestID(scope string) string {
	return fmt.Sprintf("%s:%d", scope, time.Now().UnixNano())
}
