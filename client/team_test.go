package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTeam(t *testing.T) {
	type TestCase struct {
		Name         string
		ResponseJSON string
	}

	for _, tc := range []TestCase{
		{
			Name:         "SAML",
			ResponseJSON: `{ "saml": { "roles": { "A": "OWNER", "B": { "accessGroupId": "foo" } } } }`,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintln(w, tc.ResponseJSON)
			}))
			cl := New("INVALID")
			cl.baseURL = fmt.Sprintf("http://%s", h.Listener.Addr().String())
			_, err := cl.GetTeam(context.Background(), "INVALID")
			if err != nil {
				t.Error(err)
			}
		})
	}
}
