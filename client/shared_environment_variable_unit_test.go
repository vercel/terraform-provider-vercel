package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateSharedEnvironmentVariableSendsFalseApplyToAllCustomEnvironments(t *testing.T) {
	applyToAll := false
	var requestBody struct {
		Updates map[string]struct {
			ApplyToAllCustomEnvironments *bool `json:"applyToAllCustomEnvironments"`
		} `json:"updates"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() returned error: %v", err)
		}
		fmt.Fprintln(w, `{"updated":[{"id":"env_123","key":"SECRET","value":"encrypted","type":"encrypted","target":["preview"],"projectId":[],"applyToAllCustomEnvironments":false}]}`)
	}))
	t.Cleanup(server.Close)

	cl := New("INVALID")
	cl.baseURL = server.URL
	_, err := cl.UpdateSharedEnvironmentVariable(context.Background(), UpdateSharedEnvironmentVariableRequest{
		EnvID:                        "env_123",
		Value:                        "secret",
		Type:                         "encrypted",
		ApplyToAllCustomEnvironments: &applyToAll,
	})
	if err != nil {
		t.Fatalf("UpdateSharedEnvironmentVariable() returned error: %v", err)
	}

	update := requestBody.Updates["env_123"]
	if update.ApplyToAllCustomEnvironments == nil {
		t.Fatal("applyToAllCustomEnvironments was omitted")
	}
	if *update.ApplyToAllCustomEnvironments {
		t.Fatal("applyToAllCustomEnvironments = true, want false")
	}
}
