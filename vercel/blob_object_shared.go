package vercel

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

const defaultBlobObjectCacheControlMaxAge int64 = 30 * 24 * 60 * 60

var blobObjectCacheControlMaxAgeRe = regexp.MustCompile(`(?:^|,)\s*max-age=(\d+)`)

type BlobObjectResourceModel struct {
	CacheControl       types.String `tfsdk:"cache_control"`
	CacheControlMaxAge types.Int64  `tfsdk:"cache_control_max_age"`
	ContentDisposition types.String `tfsdk:"content_disposition"`
	ContentType        types.String `tfsdk:"content_type"`
	DownloadURL        types.String `tfsdk:"download_url"`
	ETag               types.String `tfsdk:"etag"`
	ID                 types.String `tfsdk:"id"`
	Pathname           types.String `tfsdk:"pathname"`
	Size               types.Int64  `tfsdk:"size"`
	Source             types.String `tfsdk:"source"`
	SourceSHA256       types.String `tfsdk:"source_sha256"`
	StoreID            types.String `tfsdk:"store_id"`
	TeamID             types.String `tfsdk:"team_id"`
	UploadedAt         types.String `tfsdk:"uploaded_at"`
	URL                types.String `tfsdk:"url"`
}

type BlobObjectDataSourceModel struct {
	CacheControl       types.String `tfsdk:"cache_control"`
	CacheControlMaxAge types.Int64  `tfsdk:"cache_control_max_age"`
	ContentDisposition types.String `tfsdk:"content_disposition"`
	ContentType        types.String `tfsdk:"content_type"`
	DownloadURL        types.String `tfsdk:"download_url"`
	ETag               types.String `tfsdk:"etag"`
	ID                 types.String `tfsdk:"id"`
	Pathname           types.String `tfsdk:"pathname"`
	Size               types.Int64  `tfsdk:"size"`
	StoreID            types.String `tfsdk:"store_id"`
	TeamID             types.String `tfsdk:"team_id"`
	UploadedAt         types.String `tfsdk:"uploaded_at"`
	URL                types.String `tfsdk:"url"`
}

func blobObjectResourceModelFromResponse(source, sourceSHA256 types.String, storeID, teamID string, object client.BlobObject) BlobObjectResourceModel {
	return BlobObjectResourceModel{
		CacheControl:       types.StringValue(object.CacheControl),
		CacheControlMaxAge: types.Int64Value(parseBlobObjectCacheControlMaxAge(object.CacheControl)),
		ContentDisposition: types.StringValue(object.ContentDisposition),
		ContentType:        types.StringValue(object.ContentType),
		DownloadURL:        types.StringValue(object.DownloadURL),
		ETag:               types.StringValue(object.ETag),
		ID:                 types.StringValue(blobObjectID(storeID, object.Pathname)),
		Pathname:           types.StringValue(object.Pathname),
		Size:               types.Int64Value(object.Size),
		Source:             source,
		SourceSHA256:       sourceSHA256,
		StoreID:            types.StringValue(storeID),
		TeamID:             toTeamID(teamID),
		UploadedAt:         types.StringValue(object.UploadedAt),
		URL:                types.StringValue(object.URL),
	}
}

func blobObjectDataSourceModelFromResponse(storeID, teamID string, object client.BlobObject) BlobObjectDataSourceModel {
	return BlobObjectDataSourceModel{
		CacheControl:       types.StringValue(object.CacheControl),
		CacheControlMaxAge: types.Int64Value(parseBlobObjectCacheControlMaxAge(object.CacheControl)),
		ContentDisposition: types.StringValue(object.ContentDisposition),
		ContentType:        types.StringValue(object.ContentType),
		DownloadURL:        types.StringValue(object.DownloadURL),
		ETag:               types.StringValue(object.ETag),
		ID:                 types.StringValue(blobObjectID(storeID, object.Pathname)),
		Pathname:           types.StringValue(object.Pathname),
		Size:               types.Int64Value(object.Size),
		StoreID:            types.StringValue(storeID),
		TeamID:             toTeamID(teamID),
		UploadedAt:         types.StringValue(object.UploadedAt),
		URL:                types.StringValue(object.URL),
	}
}

func blobObjectID(storeID, pathname string) string {
	return fmt.Sprintf("%s/%s", storeID, pathname)
}

func parseBlobObjectCacheControlMaxAge(cacheControl string) int64 {
	matches := blobObjectCacheControlMaxAgeRe.FindStringSubmatch(cacheControl)
	if len(matches) != 2 {
		return defaultBlobObjectCacheControlMaxAge
	}

	maxAge, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return defaultBlobObjectCacheControlMaxAge
	}

	return maxAge
}

func validateManagedBlobObjectPathname(pathname string) error {
	if pathname == "" {
		return fmt.Errorf("pathname cannot be empty")
	}

	if strings.HasPrefix(pathname, "/") {
		return fmt.Errorf("pathname must not start with '/'")
	}

	if strings.HasSuffix(pathname, "/") {
		return fmt.Errorf("pathname must not end with '/' for a managed blob object")
	}

	if len(pathname) > 950 {
		return fmt.Errorf("pathname is too long, maximum length is 950 characters")
	}

	return nil
}

func inferBlobObjectContentType(pathname string) string {
	contentType := mime.TypeByExtension(path.Ext(pathname))
	contentType = strings.TrimSuffix(contentType, "; charset=utf-8")
	if contentType == "" {
		return "application/octet-stream"
	}

	return contentType
}

func readBlobObjectSource(filename string) (content []byte, sourceSHA256 string, etag string, err error) {
	content, err = os.ReadFile(filename)
	if err != nil {
		return nil, "", "", err
	}

	shaSum := sha256.Sum256(content)
	md5Sum := md5.Sum(content)

	return content, hex.EncodeToString(shaSum[:]), hex.EncodeToString(md5Sum[:]), nil
}
