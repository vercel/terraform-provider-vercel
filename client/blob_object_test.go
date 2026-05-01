package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestPutBlobObjectRetriesUnknownErrorUntilUploadSucceeds(t *testing.T) {
	originalBlobDataPlaneURL := blobDataPlaneURL
	originalBlobDataPlaneSleep := blobDataPlaneSleep
	defer func() {
		blobDataPlaneURL = originalBlobDataPlaneURL
		blobDataPlaneSleep = originalBlobDataPlaneSleep
	}()

	const successfulAttempt = blobDataPlaneTransientAttempts
	putAttempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/storage/stores/store_123":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for store lookup: %s", r.Method)
			}
			fmt.Fprint(w, `{"store":{"id":"store_123","access":"public"}}`)
		case "/v1/storage/stores/store_123/secrets":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for store secrets lookup: %s", r.Method)
			}
			fmt.Fprint(w, `{"rwToken":"vercel_blob_rw_store_123_test"}`)
		case "/blob":
			if r.Method != http.MethodPut {
				t.Fatalf("unexpected method for blob upload: %s", r.Method)
			}
			if pathname := r.URL.Query().Get("pathname"); pathname != "terraform/object.txt" {
				t.Fatalf("expected upload pathname terraform/object.txt, got %q", pathname)
			}

			putAttempts++
			if attempt := r.Header.Get("x-api-blob-request-attempt"); attempt != strconv.Itoa(putAttempts-1) {
				t.Fatalf("expected blob request attempt %d, got %q", putAttempts-1, attempt)
			}

			if putAttempts < successfulAttempt {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":{"code":"unknown_error","message":"Unknown error"}}`)
				return
			}

			fmt.Fprint(w, `{
				"cacheControl":"public, max-age=3600",
				"contentDisposition":"inline; filename=\"object.txt\"",
				"contentType":"text/plain",
				"downloadUrl":"https://example.com/terraform/object.txt?download=1",
				"etag":"\"etag-123\"",
				"pathname":"terraform/object.txt",
				"size":3,
				"uploadedAt":"2026-04-08T00:00:00Z",
				"url":"https://example.com/terraform/object.txt"
			}`)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	blobDataPlaneURL = server.URL + "/blob"
	blobDataPlaneSleep = func(time.Duration) {}

	client := New("test-token")
	client.baseURL = server.URL

	object, err := client.PutBlobObject(context.Background(), PutBlobObjectRequest{
		Body:               []byte("foo"),
		CacheControlMaxAge: 3600,
		ContentType:        "text/plain",
		Pathname:           "terraform/object.txt",
		StoreID:            "store_123",
	})
	if err != nil {
		t.Fatalf("PutBlobObject returned error: %v", err)
	}

	if putAttempts != successfulAttempt {
		t.Fatalf("expected %d upload attempts, got %d", successfulAttempt, putAttempts)
	}

	if object.ETag != "etag-123" {
		t.Fatalf("expected normalized ETag etag-123, got %q", object.ETag)
	}
}

func TestBlobDataPlaneRetryDelayCapsExponentialBackoff(t *testing.T) {
	testCases := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 200 * time.Millisecond},
		{attempt: 6, want: 6400 * time.Millisecond},
		{attempt: 7, want: 10 * time.Second},
		{attempt: 12, want: 10 * time.Second},
	}

	for _, tc := range testCases {
		if got := blobDataPlaneRetryDelay(tc.attempt); got != tc.want {
			t.Fatalf("expected retry delay %s on attempt %d, got %s", tc.want, tc.attempt, got)
		}
	}
}
