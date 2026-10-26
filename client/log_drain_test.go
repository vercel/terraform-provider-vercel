package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogDrainResponseErrorsDoNotExposeSigningSecret(t *testing.T) {
	t.Parallel()

	const signingSecret = "signing-secret-that-must-not-reach-diagnostics"
	for _, response := range []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "malformed success response",
			status: http.StatusOK,
			body:   `{"id":123,"delivery":{"secret":"` + signingSecret + `"}}`,
		},
		{
			name:   "API error response",
			status: http.StatusBadRequest,
			body:   `{"error":{"code":"bad_request","message":"` + signingSecret + `"}}`,
		},
	} {
		for _, operation := range []struct {
			name string
			run  func(*Client) error
		}{
			{
				name: "create",
				run: func(client *Client) error {
					_, err := client.CreateLogDrain(context.Background(), CreateLogDrainRequest{
						TeamID:         "team_123",
						Name:           "example",
						DeliveryFormat: "ndjson",
						Environments:   []string{"production"},
						Secret:         signingSecret,
						Sources:        []string{"static"},
						Endpoint:       "https://example.com/logs",
					})
					return err
				},
			},
			{
				name: "get",
				run: func(client *Client) error {
					_, err := client.GetLogDrain(context.Background(), "ld_123", "team_123")
					return err
				},
			},
		} {
			t.Run(operation.name+"/"+response.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(response.status)
					fmt.Fprint(w, response.body)
				}))
				defer server.Close()

				err := operation.run(New("test").WithBaseURL(server.URL))
				if err == nil {
					t.Fatal("expected response error")
				}
				if strings.Contains(err.Error(), signingSecret) {
					t.Fatalf("error exposed Log Drain signing secret: %s", err)
				}
			})
		}
	}
}

func TestCreateLogDrainPreservesSafeAPIErrorContext(t *testing.T) {
	t.Parallel()

	const signingSecret = "configured-secret-that-must-not-reach-diagnostics"
	const headerSecret = "short-header-secret"
	const projectID = "prj_b2OSHL6PZD5KEASIpKtPI7RJEfOh"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(
			w,
			`{"error":{"code":"bad_request","message":"invalid project ids: %s; secret=%s; header=%s"}}`,
			projectID,
			signingSecret,
			headerSecret,
		)
	}))
	defer server.Close()

	_, err := New("test").WithBaseURL(server.URL).CreateLogDrain(context.Background(), CreateLogDrainRequest{
		TeamID:         "team_123",
		Name:           "example",
		DeliveryFormat: "ndjson",
		Environments:   []string{"production"},
		Headers:        map[string]string{"x-private": headerSecret},
		Secret:         signingSecret,
		Sources:        []string{"static"},
		Endpoint:       "https://example.com/logs",
	})
	if err == nil {
		t.Fatal("expected response error")
	}
	if strings.Contains(err.Error(), signingSecret) || strings.Contains(err.Error(), headerSecret) {
		t.Fatalf("error exposed Log Drain request secret: %s", err)
	}
	if !strings.Contains(err.Error(), "bad_request - invalid project ids") ||
		!strings.Contains(err.Error(), projectID) {
		t.Fatalf("error omitted safe API context: %s", err)
	}
}

func TestDeleteLogDrainErrorsDoNotExposeSigningSecret(t *testing.T) {
	t.Parallel()

	const signingSecret = "signing-secret-that-must-not-reach-diagnostics"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":{"code":"bad_request","message":%q}}`, signingSecret)
	}))
	defer server.Close()

	err := New("test").WithBaseURL(server.URL).DeleteLogDrain(context.Background(), "ld_123", "team_123")
	if err == nil {
		t.Fatal("expected response error")
	}
	if strings.Contains(err.Error(), signingSecret) {
		t.Fatalf("error exposed Log Drain signing secret: %s", err)
	}
}
