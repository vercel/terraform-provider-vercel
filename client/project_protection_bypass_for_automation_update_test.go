package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPatchProtectionBypassForAutomation(t *testing.T) {
	const secret = "12345678912345678912345678912345"
	const note = "GitHub Actions"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/v10/projects/prj_test/protection-bypass" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_test" {
			t.Fatalf("unexpected teamId query parameter: %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("error reading request body: %v", err)
		}

		expectedBody := fmt.Sprintf(`{"generate":{"secret":"%s","note":"%s"}}`, secret, note)
		if string(body) != expectedBody {
			t.Fatalf("unexpected request body: %s", string(body))
		}

		fmt.Fprintf(w, `{"protectionBypass":{"%s":{"scope":"automation-bypass","isEnvVar":true,"note":"%s"}}}`, secret, note)
	}))
	defer server.Close()

	cl := New("INVALID")
	cl.baseURL = server.URL

	protectionBypass, err := cl.PatchProtectionBypassForAutomation(context.Background(), PatchProtectionBypassForAutomationRequest{
		ProjectID: "prj_test",
		TeamID:    "team_test",
		Generate: &GenerateProtectionBypassRequest{
			Secret: secret,
			Note:   stringPointer("GitHub Actions"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(protectionBypass) != 1 {
		t.Fatalf("unexpected protection bypass count: %d", len(protectionBypass))
	}
	if protectionBypass[secret].Scope != "automation-bypass" {
		t.Fatalf("unexpected protection bypass scope: %s", protectionBypass[secret].Scope)
	}
	if protectionBypass[secret].Note == nil || *protectionBypass[secret].Note != note {
		t.Fatalf("unexpected protection bypass note: %#v", protectionBypass[secret].Note)
	}
}

func TestUpdateProtectionBypassForAutomation(t *testing.T) {
	const oldSecret = "12345678912345678912345678912345"
	const newSecret = "abcdefghijklmnopqrstuvwxyz123456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("error reading request body: %v", err)
		}

		expectedBody := fmt.Sprintf(`{"generate":{"secret":"%s"},"revoke":{"regenerate":true,"secret":"%s"}}`, newSecret, oldSecret)
		if string(body) != expectedBody {
			t.Fatalf("unexpected request body: %s", string(body))
		}

		fmt.Fprintf(w, `{"protectionBypass":{"%s":{"scope":"automation-bypass","isEnvVar":true}}}`, newSecret)
	}))
	defer server.Close()

	cl := New("INVALID")
	cl.baseURL = server.URL

	secret, err := cl.UpdateProtectionBypassForAutomation(context.Background(), UpdateProtectionBypassForAutomationRequest{
		ProjectID: "prj_test",
		TeamID:    "team_test",
		NewValue:  true,
		NewSecret: newSecret,
		OldSecret: oldSecret,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != newSecret {
		t.Fatalf("unexpected returned secret: %s", secret)
	}
}

func stringPointer(value string) *string {
	return &value
}
