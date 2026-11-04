package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProjectDeploymentProtectionRedactsErrorResponse(t *testing.T) {
	const secret = "private-password-fixture"
	for _, body := range []string{`{"error":{"code":"bad_request","message":"` + secret + `"}}`, secret} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(400); fmt.Fprint(w, body) }))
		c := New("test").WithBaseURL(server.URL)
		protection := &PasswordProtectionWithPassword{DeploymentType: "all", Password: secret}
		_, err := c.UpdateProjectDeploymentProtection(context.Background(), "prj_existing", "", ProjectDeploymentProtectionRequest{Password: &protection})
		server.Close()
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("sensitive error was not redacted: %v", err)
		}
	}
}
