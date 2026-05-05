package client

import (
	"encoding/json"
	"testing"
)

func TestUpdateSharedEnvironmentVariableRequestIncludesEmptyValues(t *testing.T) {
	payload := string(mustMarshal(struct {
		Updates map[string]UpdateSharedEnvironmentVariableRequest `json:"updates"`
	}{
		Updates: map[string]UpdateSharedEnvironmentVariableRequest{
			"env_123": {
				Value:                        "",
				Type:                         "encrypted",
				ProjectIDs:                   []string{},
				ApplyToAllCustomEnvironments: false,
				Target:                       []string{},
				Comment:                      "",
			},
		},
	}))

	var out struct {
		Updates map[string]map[string]any `json:"updates"`
	}
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	update := out.Updates["env_123"]
	for _, field := range []string{"value", "projectId", "applyToAllCustomEnvironments", "target", "comment"} {
		if _, ok := update[field]; !ok {
			t.Fatalf("field %q was omitted from payload %s", field, payload)
		}
	}
}
