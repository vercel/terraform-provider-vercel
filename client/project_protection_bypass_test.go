package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactProtectionBypassPayloadRedactsNestedSecrets(t *testing.T) {
	note := "keep me"
	payload := string(mustMarshal(struct {
		Update updateBypassBody `json:"update"`
	}{
		Update: updateBypassBody{
			Secret: "abcdefghijklmnopqrstuvwxyz123456",
			Note:   &note,
		},
	}))

	redacted := redactProtectionBypassPayload(payload)

	if strings.Contains(redacted, "abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("redacted payload still contains secret: %s", redacted)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal([]byte(redacted), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := body["update"]["secret"]; got != redactedProtectionBypassSecret {
		t.Fatalf("update.secret = %v, want %q", got, redactedProtectionBypassSecret)
	}
	if got := body["update"]["note"]; got != note {
		t.Fatalf("update.note = %v, want %q", got, note)
	}
}

func TestRedactProtectionBypassPayloadRedactsGenerateAndRevokeSecrets(t *testing.T) {
	newSecret := "abcdefghijklmnopqrstuvwxyz123456"
	oldSecret := "12345678901234567890123456789012"
	payload := `{"generate":{"secret":"` + newSecret + `"},"revoke":{"regenerate":true,"secret":"` + oldSecret + `"}}`

	redacted := redactProtectionBypassPayload(payload)

	if strings.Contains(redacted, newSecret) {
		t.Fatalf("redacted payload still contains new secret: %s", redacted)
	}
	if strings.Contains(redacted, oldSecret) {
		t.Fatalf("redacted payload still contains old secret: %s", redacted)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal([]byte(redacted), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := body["generate"]["secret"]; got != redactedProtectionBypassSecret {
		t.Fatalf("generate.secret = %v, want %q", got, redactedProtectionBypassSecret)
	}
	if got := body["revoke"]["secret"]; got != redactedProtectionBypassSecret {
		t.Fatalf("revoke.secret = %v, want %q", got, redactedProtectionBypassSecret)
	}
}
