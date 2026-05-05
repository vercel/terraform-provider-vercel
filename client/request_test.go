package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoRequestDefaultsRetryAfterToSeconds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintln(w, `{"error":{"code":"rate_limited","message":"slow down"}}`)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	err = New("INVALID")._doRequest(req, nil, false)
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("_doRequest() error = %v, want APIError", err)
	}
	if apiErr.retryAfter != 1 {
		t.Fatalf("retryAfter = %d, want 1", apiErr.retryAfter)
	}
}
