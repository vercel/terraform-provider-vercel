package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTraceDrainUsesOTLPHTTPDelivery(t *testing.T) {
	var got struct {
		Name       string   `json:"name"`
		Projects   string   `json:"projects"`
		ProjectIDs []string `json:"projectIds"`
		Schemas    map[string]struct {
			Version string `json:"version"`
		} `json:"schemas"`
		Delivery struct {
			Type     string `json:"type"`
			Endpoint struct {
				Traces string `json:"traces"`
			} `json:"endpoint"`
			Encoding string            `json:"encoding"`
			Headers  map[string]string `json:"headers"`
			Secret   string            `json:"secret"`
		} `json:"delivery"`
		Sampling []struct {
			Type        string  `json:"type"`
			Rate        float64 `json:"rate"`
			Env         string  `json:"env"`
			RequestPath string  `json:"requestPath"`
		} `json:"sampling"`
		Source struct {
			Kind string `json:"kind"`
		} `json:"source"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/drains" {
			t.Fatalf("path = %s, want /v1/drains", r.URL.Path)
		}
		if r.URL.Query().Get("teamId") != "team_123" {
			t.Fatalf("teamId = %s, want team_123", r.URL.Query().Get("teamId"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"drn_123",
			"ownerId":"team_123",
			"name":"trace",
			"projectIds":["prj_123"],
			"delivery":{
				"type":"otlphttp",
				"endpoint":{"traces":"https://example.com/v1/traces"},
				"encoding":"json",
				"headers":{"authorization":"bearer token"},
				"secret":"abcdefghijklmnopqrstuvwxyz123456"
			},
			"sampling":[{"type":"head_sampling","rate":0.25,"env":"production","requestPath":"/api"}]
		}`))
	}))
	t.Cleanup(server.Close)

	traceDrain, err := New("TOKEN").WithBaseURL(server.URL).CreateTraceDrain(context.Background(), CreateTraceDrainRequest{
		TeamID:         "team_123",
		Name:           "trace",
		DeliveryFormat: "json",
		Headers:        map[string]string{"authorization": "bearer token"},
		ProjectIDs:     []string{"prj_123"},
		SamplingRules: []TraceDrainSamplingRule{
			{Rate: 0.25, Environment: "production", RequestPath: "/api"},
		},
		Secret:   "abcdefghijklmnopqrstuvwxyz123456",
		Endpoint: "https://example.com/v1/traces",
	})
	if err != nil {
		t.Fatalf("CreateTraceDrain() error = %v", err)
	}

	if got.Delivery.Type != "otlphttp" {
		t.Fatalf("delivery.type = %s, want otlphttp", got.Delivery.Type)
	}
	if got.Delivery.Endpoint.Traces != "https://example.com/v1/traces" {
		t.Fatalf("delivery.endpoint.traces = %s", got.Delivery.Endpoint.Traces)
	}
	if got.Schemas["trace"].Version != "v1" {
		t.Fatalf("schemas.trace.version = %s, want v1", got.Schemas["trace"].Version)
	}
	if got.Sampling[0].Rate != 0.25 || got.Sampling[0].Env != "production" || got.Sampling[0].RequestPath != "/api" {
		t.Fatalf("sampling = %#v", got.Sampling[0])
	}
	if traceDrain.Endpoint != "https://example.com/v1/traces" {
		t.Fatalf("traceDrain.Endpoint = %s", traceDrain.Endpoint)
	}
	if len(traceDrain.SamplingRules) != 1 || traceDrain.SamplingRules[0].Rate != 0.25 {
		t.Fatalf("traceDrain.SamplingRules = %#v", traceDrain.SamplingRules)
	}
}
