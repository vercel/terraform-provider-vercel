package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAuditLogDrainHTTP(t *testing.T) {
	compression := "gzip"
	secret := "signing-secret"
	var got struct {
		Name     string                         `json:"name"`
		Projects string                         `json:"projects"`
		Schemas  map[string]drainsSchemaVersion `json:"schemas"`
		Delivery auditLogDrainsDeliveryHTTP     `json:"delivery"`
		Source   drainsSource                   `json:"source"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/drains" {
			t.Fatalf("request = %s %s, want POST /v1/drains", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"drn_123","ownerId":"team_123","name":"security",
			"schemas":{"audit_log":{}},
			"delivery":{"type":"http","endpoint":"https://example.com/audit","encoding":"ndjson","compression":"gzip","headers":{"Authorization":"Bearer token"},"secret":"signing-secret"}
		}`))
	}))
	t.Cleanup(server.Close)

	drain, err := New("TOKEN").WithBaseURL(server.URL).CreateAuditLogDrain(context.Background(), CreateAuditLogDrainRequest{
		TeamID: "team_123",
		Name:   "security",
		HTTP: &AuditLogDrainHTTPDelivery{
			Endpoint:    "https://example.com/audit",
			Encoding:    "ndjson",
			Compression: &compression,
			Headers:     map[string]string{"Authorization": "Bearer token"},
			Secret:      &secret,
		},
	})
	if err != nil {
		t.Fatalf("CreateAuditLogDrain() error = %v", err)
	}
	if got.Projects != "all" || got.Schemas["audit_log"].Version != "v1" || got.Source.Kind != "self-served" {
		t.Fatalf("create envelope = %#v", got)
	}
	if got.Delivery.Type != "http" || got.Delivery.Endpoint != "https://example.com/audit" || got.Delivery.Encoding != "ndjson" {
		t.Fatalf("delivery = %#v", got.Delivery)
	}
	if got.Delivery.Compression == nil || *got.Delivery.Compression != "gzip" || got.Delivery.Secret == nil || *got.Delivery.Secret != "signing-secret" {
		t.Fatalf("delivery optional fields = %#v", got.Delivery)
	}
	if drain.HTTP == nil || drain.S3 != nil || drain.HTTP.Secret == nil || *drain.HTTP.Secret != "signing-secret" {
		t.Fatalf("drain = %#v", drain)
	}
}

func TestCreateAuditLogDrainS3(t *testing.T) {
	encryption := "aws:kms"
	objectACL := "bucket-owner-full-control"
	var got struct {
		Projects string                         `json:"projects"`
		Schemas  map[string]drainsSchemaVersion `json:"schemas"`
		Delivery auditLogDrainsDeliveryS3       `json:"delivery"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"drn_s3","ownerId":"team_123","name":"archive",
			"schemas":{"audit_log":{}},
			"delivery":{"type":"s3","endpoint":"s3://audit/prefix","encoding":"ndjson","compression":"none","fileStructure":"hive","roleArn":"arn:aws:iam::123:role/drain","region":"eu-west-1","serverSideEncryption":"aws:kms","objectAcl":"bucket-owner-full-control"}
		}`))
	}))
	t.Cleanup(server.Close)

	drain, err := New("TOKEN").WithBaseURL(server.URL).CreateAuditLogDrain(context.Background(), CreateAuditLogDrainRequest{
		Name: "archive",
		S3: &AuditLogDrainS3Delivery{
			Endpoint:             "s3://audit/prefix",
			Encoding:             "ndjson",
			RoleARN:              "arn:aws:iam::123:role/drain",
			Region:               "eu-west-1",
			ServerSideEncryption: &encryption,
			ObjectACL:            &objectACL,
		},
	})
	if err != nil {
		t.Fatalf("CreateAuditLogDrain() error = %v", err)
	}
	if got.Projects != "all" || got.Schemas["audit_log"].Version != "v1" {
		t.Fatalf("create envelope = %#v", got)
	}
	if got.Delivery.Type != "s3" || got.Delivery.Compression != "none" || got.Delivery.FileStructure != "hive" {
		t.Fatalf("delivery constants = %#v", got.Delivery)
	}
	if drain.S3 == nil || drain.HTTP != nil || drain.S3.ObjectACL == nil || *drain.S3.ObjectACL != objectACL {
		t.Fatalf("drain = %#v", drain)
	}
}

func TestCreateAuditLogDrainRequiresExactlyOneDelivery(t *testing.T) {
	client := New("TOKEN")
	for _, request := range []CreateAuditLogDrainRequest{
		{},
		{HTTP: &AuditLogDrainHTTPDelivery{}, S3: &AuditLogDrainS3Delivery{}},
	} {
		_, err := client.CreateAuditLogDrain(context.Background(), request)
		if err == nil || !strings.Contains(err.Error(), "exactly one") {
			t.Fatalf("CreateAuditLogDrain() error = %v, want exactly-one error", err)
		}
	}
}

func TestGetAuditLogDrainValidatesSchemaAndDelivery(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  string
	}{
		{
			name:     "presence-only audit schema",
			response: `{"id":"drn_123","schemas":{"audit_log":{}},"delivery":{"type":"http","endpoint":"https://example.com","encoding":"json","headers":{}}}`,
		},
		{
			name:     "missing audit schema",
			response: `{"id":"drn_123","schemas":{"log":{"version":"v1"}},"delivery":{"type":"http"}}`,
			wantErr:  "audit_log schema is missing",
		},
		{
			name:     "unsupported audit schema version",
			response: `{"id":"drn_123","schemas":{"audit_log":{"version":"v2"}},"delivery":{"type":"http"}}`,
			wantErr:  `unsupported audit_log schema version "v2"`,
		},
		{
			name:     "unsupported delivery",
			response: `{"id":"drn_123","schemas":{"audit_log":{"version":"v1"}},"delivery":{"type":"splunk"}}`,
			wantErr:  `unsupported Audit Log Drain delivery type "splunk"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/drains/drn_123" {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(tt.response))
			}))
			t.Cleanup(server.Close)

			_, err := New("TOKEN").WithBaseURL(server.URL).GetAuditLogDrain(context.Background(), "drn_123", "")
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("GetAuditLogDrain() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("GetAuditLogDrain() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetAuditLogDrainPreservesOmittedHTTPSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"id":"drn_123","schemas":{"audit_log":{"version":"v1"}},
			"delivery":{"type":"http","endpoint":"https://example.com","encoding":"json"}
		}`))
	}))
	t.Cleanup(server.Close)

	drain, err := New("TOKEN").WithBaseURL(server.URL).GetAuditLogDrain(context.Background(), "drn_123", "")
	if err != nil {
		t.Fatalf("GetAuditLogDrain() error = %v", err)
	}
	if drain.HTTP.Headers != nil || drain.HTTP.Secret != nil || drain.HTTP.Compression != nil {
		t.Fatalf("HTTP optional fields = %#v, want nil", drain.HTTP)
	}
}

func TestDeleteAuditLogDrain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/drains/drn_123" || r.URL.Query().Get("teamId") != "team_123" {
			t.Fatalf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	if err := New("TOKEN").WithBaseURL(server.URL).DeleteAuditLogDrain(context.Background(), "drn_123", "team_123"); err != nil {
		t.Fatalf("DeleteAuditLogDrain() error = %v", err)
	}
}
