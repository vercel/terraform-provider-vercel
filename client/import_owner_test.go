package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChildReadDiscoversImportOwner(t *testing.T) {
	for _, op := range []struct {
		name, parentPath, parentJSON, childPath, childJSON string
		read                                               func(*Client, string) (string, error)
	}{
		{"edge config item", "/v1/edge-config/ec_1", `{"ownerId":"team_owner"}`, "/v1/edge-config/ec_1/item/key", `{"value":"test"}`, func(c *Client, team string) (string, error) {
			r, e := c.GetEdgeConfigItem(context.Background(), EdgeConfigItemRequest{EdgeConfigID: "ec_1", Key: "key", TeamID: team})
			return r.TeamID, e
		}},
		{"edge config schema", "/v1/edge-config/ec_1", `{"ownerId":"team_owner"}`, "/v1/edge-config/ec_1/schema", `{"definition":{}}`, func(c *Client, team string) (string, error) {
			r, e := c.GetEdgeConfigSchema(context.Background(), "ec_1", team)
			return r.TeamID, e
		}},
		{"edge config token", "/v1/edge-config/ec_1", `{"ownerId":"team_owner"}`, "/v1/edge-config/ec_1/token/token", `{"id":"tok_1"}`, func(c *Client, team string) (string, error) {
			r, e := c.GetEdgeConfigToken(context.Background(), EdgeConfigTokenRequest{EdgeConfigID: "ec_1", Token: "token", TeamID: team})
			return r.TeamID, e
		}},
		{"access group member", "/v1/access-groups/ag_1", `{"teamId":"team_owner"}`, "/v1/access-groups/ag_1/members", `{"members":[{"uid":"user_1"}]}`, func(c *Client, team string) (string, error) {
			r, e := c.GetAccessGroupMember(context.Background(), GetAccessGroupMemberRequest{AccessGroupID: "ag_1", UserID: "user_1", TeamID: team})
			return r.TeamID, e
		}},
		{"DNS record", "/v5/domains/example.com", `{"domain":{"teamId":"team_owner","userId":"user_1"}}`, "/domains/records/rec_1", `{"id":"rec_1","domain":"example.com"}`, func(c *Client, team string) (string, error) {
			r, e := c.GetDNSRecord(context.Background(), "rec_1", team)
			return r.TeamID, e
		}},
	} {
		t.Run(op.name, func(t *testing.T) {
			for _, tc := range []struct {
				name, request, provider string
				parentStatus            int
				parentJSON              string
				want                    string
				wantError               bool
			}{
				{name: "unscoped import", parentJSON: op.parentJSON, want: "team_owner"},
				{name: "explicit team", request: "team_explicit", want: "team_explicit"},
				{name: "provider team", provider: "team_provider", want: "team_provider"},
				{name: "parent lookup fails", parentStatus: 403, wantError: true},
				{name: "parent owner omitted", parentJSON: `{}`, wantError: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					parentCalls := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != "GET" {
							t.Errorf("unexpected method %s", r.Method)
						}
						switch r.URL.Path {
						case op.parentPath:
							parentCalls++
							if tc.parentStatus != 0 {
								w.WriteHeader(tc.parentStatus)
								fmt.Fprint(w, `{"error":{"code":"forbidden","message":"forbidden"}}`)
								return
							}
							fmt.Fprint(w, tc.parentJSON)
						case op.childPath:
							expected := tc.request
							if expected == "" {
								expected = tc.provider
							}
							if expected == "" && op.name != "DNS record" {
								expected = "team_owner"
							}
							if got := r.URL.Query().Get("teamId"); got != expected {
								t.Errorf("team query=%q, want %q", got, expected)
							}
							fmt.Fprint(w, op.childJSON)
						default:
							t.Errorf("unexpected path %s", r.URL.Path)
							w.WriteHeader(404)
						}
					}))
					defer server.Close()
					c := New("test").WithBaseURL(server.URL).WithTeam(Team{ID: tc.provider})
					got, err := op.read(c, tc.request)
					if tc.wantError {
						if err == nil {
							t.Fatal("expected ownership error")
						}
						if !strings.Contains(err.Error(), "ownership") {
							t.Errorf("missing ownership error context: %v", err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if got != tc.want {
						t.Errorf("team=%q, want %q", got, tc.want)
					}
					expectedCalls := 1
					if tc.request != "" || tc.provider != "" {
						expectedCalls = 0
					}
					if parentCalls != expectedCalls {
						t.Errorf("parent calls=%d, want %d", parentCalls, expectedCalls)
					}
				})
			}
		})
	}
}

func TestDNSRecordPersonalOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/domains/records/rec_1" {
			fmt.Fprint(w, `{"domain":"example.com"}`)
			return
		}
		fmt.Fprint(w, `{"domain":{"teamId":null,"userId":"user_1"}}`)
	}))
	defer server.Close()
	r, err := New("test").WithBaseURL(server.URL).GetDNSRecord(context.Background(), "rec_1", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.TeamID != "" {
		t.Errorf("personal record TeamID=%q, want empty", r.TeamID)
	}
}

func TestKMSIssuerResponseOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"issuer_1","ownerId":"team_owner"}`)
	}))
	defer server.Close()
	r, err := New("test").WithBaseURL(server.URL).GetKMSIssuer(context.Background(), "issuer_1", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.TeamID != "team_owner" {
		t.Errorf("TeamID=%q, want team_owner", r.TeamID)
	}
}
