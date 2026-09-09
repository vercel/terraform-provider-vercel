package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestDeleteAllAlertRulesDeletesOnlyAcceptanceTestRules(t *testing.T) {
	var deleted []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Errorf("teamId = %q, want team_123", got)
		}
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"rules":[{"id":"ar_test","type":"built-in","name":"test-acc-alert-rule-123","isDefault":false},{"id":"ar_shared","type":"built-in","name":"shared alert","isDefault":false},{"id":"ar_default","type":"built-in","name":"test-acc-alert-rule-default","isDefault":true},{"id":"ar_custom","type":"custom","name":"test-acc-alert-rule-custom","isDefault":false}],"pagination":{"next":null}}`))
		case http.MethodDelete:
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	t.Cleanup(server.Close)

	err := deleteAllAlertRules(context.Background(), client.New("TOKEN").WithBaseURL(server.URL), "team_123")
	if err != nil {
		t.Fatalf("deleteAllAlertRules() error = %v", err)
	}
	want := []string{"/alerts/v3/alert-rules/ar_test"}
	if !reflect.DeepEqual(deleted, want) {
		t.Fatalf("deleted = %v, want %v", deleted, want)
	}
}
