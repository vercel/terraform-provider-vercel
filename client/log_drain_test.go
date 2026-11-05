package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestCreateLogDrainPreservesAPIErrorCode(t *testing.T) {
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
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "bad_request" || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("error omitted API error code or status: %s", err)
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

func TestCreateLogDrainSampling(t *testing.T) {
	t.Parallel()

	legacyRate := 0.25
	zeroRate := 0.0
	for _, tc := range []struct {
		name         string
		rate         *float64
		rules        []DrainSamplingRule
		wantSampling string
	}{
		{
			name:         "omitted sampling sends all logs",
			wantSampling: `null`,
		},
		{
			name:         "legacy rate",
			rate:         &legacyRate,
			wantSampling: `[{"type":"head_sampling","rate":0.25}]`,
		},
		{
			name:         "explicit legacy zero drops all logs",
			rate:         &zeroRate,
			wantSampling: `[{"type":"head_sampling","rate":0}]`,
		},
		{
			name: "ordered conditional rules and fallback",
			rules: []DrainSamplingRule{
				{Rate: 0, Environment: "production", RequestPath: "/health"},
				{Rate: 0.25, Environment: "production", RequestPath: "/api"},
				{Rate: 1},
			},
			wantSampling: `[{"type":"head_sampling","rate":0,"env":"production","requestPath":"/health"},{"type":"head_sampling","rate":0.25,"env":"production","requestPath":"/api"},{"type":"head_sampling","rate":1}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotSampling json.RawMessage
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v1/drains" {
					t.Errorf("request = %s %s, want POST /v1/drains", r.Method, r.URL.Path)
				}
				var payload map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decoding request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				gotSampling = payload["sampling"]
				if len(gotSampling) == 0 {
					gotSampling = json.RawMessage(`null`)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"drn_123","sampling":%s}`, gotSampling)
			}))
			defer server.Close()

			drain, err := New("test").WithBaseURL(server.URL).CreateLogDrain(context.Background(), CreateLogDrainRequest{
				TeamID:         "team_123",
				Name:           "sampling-test",
				DeliveryFormat: "ndjson",
				Environments:   []string{"production", "preview"},
				Sources:        []string{"lambda"},
				Endpoint:       "https://example.com/logs",
				SamplingRate:   tc.rate,
				SamplingRules:  tc.rules,
			})
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(gotSampling, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.wantSampling), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("sampling = %s, want %s", gotSampling, tc.wantSampling)
			}
			if len(tc.rules) > 0 && !reflect.DeepEqual(drain.SamplingRules, tc.rules) {
				t.Fatalf("returned sampling rules = %#v, want %#v", drain.SamplingRules, tc.rules)
			}
			if tc.rate != nil && (drain.SamplingRate == nil || *drain.SamplingRate != *tc.rate) {
				t.Fatalf("returned legacy sampling rate = %v, want %v", drain.SamplingRate, *tc.rate)
			}
		})
	}
}

func TestGetLogDrainSampling(t *testing.T) {
	t.Parallel()

	legacyRate := 0.25
	for _, tc := range []struct {
		name      string
		sampling  string
		wantRules []DrainSamplingRule
		wantRate  *float64
	}{
		{name: "no sampling", sampling: `[]`},
		{
			name:      "single unconditional rule preserves legacy rate",
			sampling:  `[{"type":"head_sampling","rate":0.25}]`,
			wantRules: []DrainSamplingRule{{Rate: 0.25}},
			wantRate:  &legacyRate,
		},
		{
			name:      "single conditional rule is not a global rate",
			sampling:  `[{"type":"head_sampling","rate":0.25,"env":"preview","requestPath":"/api"}]`,
			wantRules: []DrainSamplingRule{{Rate: 0.25, Environment: "preview", RequestPath: "/api"}},
		},
		{
			name:     "multiple rules preserve conditions and order",
			sampling: `[{"type":"head_sampling","rate":0,"env":"production","requestPath":"/health"},{"type":"head_sampling","rate":0.25,"env":"production"},{"type":"head_sampling","rate":1,"env":"preview"}]`,
			wantRules: []DrainSamplingRule{
				{Rate: 0, Environment: "production", RequestPath: "/health"},
				{Rate: 0.25, Environment: "production"},
				{Rate: 1, Environment: "preview"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/drains/drn_123" {
					t.Errorf("request = %s %s, want GET /v1/drains/drn_123", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"drn_123","sampling":%s}`, tc.sampling)
			}))
			defer server.Close()

			drain, err := New("test").WithBaseURL(server.URL).GetLogDrain(context.Background(), "drn_123", "team_123")
			if err != nil {
				t.Fatal(err)
			}
			if len(drain.SamplingRules) != len(tc.wantRules) || (len(tc.wantRules) > 0 && !reflect.DeepEqual(drain.SamplingRules, tc.wantRules)) {
				t.Fatalf("sampling rules = %#v, want %#v", drain.SamplingRules, tc.wantRules)
			}
			if !reflect.DeepEqual(drain.SamplingRate, tc.wantRate) {
				t.Fatalf("legacy sampling rate = %v, want %v", drain.SamplingRate, tc.wantRate)
			}
		})
	}
}
