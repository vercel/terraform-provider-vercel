package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAlertRuleNotifications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.EscapedPath(); got != "/alerts/v3/alert-rules/ar_custom%2Fvalue/notifications" {
			t.Fatalf("path = %s, want escaped custom alert rule path", got)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"notifications": [
				{"type":"slack","configId":"icfg_123","channelId":"C123","channelName":"alerts","slackWorkspaceId":"T123","id":"all","name":"All Projects","projectScope":{"type":"all"}},
				{"type":"webhook","webhook":{"id":"hook_123","url":"https://example.com/webhook","events":["alerts.triggered"],"ownerId":"team_123","createdAt":1,"updatedAt":2,"createdFrom":"account"}},
				{"type":"webhook","webhook":{"id":"hook_missing"}}
			]
		}`)
	}))
	t.Cleanup(server.Close)

	notifications, err := New("TOKEN").WithBaseURL(server.URL).GetAlertRuleNotifications(context.Background(), "ar_custom/value", "team_123")
	if err != nil {
		t.Fatalf("GetAlertRuleNotifications() error = %v", err)
	}
	if len(notifications) != 3 {
		t.Fatalf("len(notifications) = %d, want 3", len(notifications))
	}
	if notifications[0].Type != AlertRuleNotificationTypeSlack || notifications[0].ConfigID != "icfg_123" || notifications[0].ChannelID != "C123" {
		t.Fatalf("Slack notification = %#v", notifications[0])
	}
	if notifications[1].Type != AlertRuleNotificationTypeWebhook || notifications[1].Webhook.ID != "hook_123" {
		t.Fatalf("hydrated webhook notification = %#v", notifications[1])
	}
	if notifications[2].Type != AlertRuleNotificationTypeWebhook || notifications[2].Webhook.ID != "hook_missing" {
		t.Fatalf("ID-only webhook notification = %#v", notifications[2])
	}
}

func TestMutateAlertRuleNotification(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		target   AlertRuleNotificationTarget
		wantBody map[string]string
		mutate   func(*Client, context.Context, AlertRuleNotificationRequest) error
	}{
		{
			name:     "link Slack channel",
			method:   http.MethodPost,
			target:   AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeSlack, ConfigID: "icfg_123", ChannelID: "C123"},
			wantBody: map[string]string{"type": "slack", "configId": "icfg_123", "channelId": "C123"},
			mutate: func(c *Client, ctx context.Context, request AlertRuleNotificationRequest) error {
				return c.LinkAlertRuleNotification(ctx, request)
			},
		},
		{
			name:     "unlink webhook",
			method:   http.MethodDelete,
			target:   AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeWebhook, WebhookID: "hook_123"},
			wantBody: map[string]string{"type": "webhook", "webhookId": "hook_123"},
			mutate: func(c *Client, ctx context.Context, request AlertRuleNotificationRequest) error {
				return c.UnlinkAlertRuleNotification(ctx, request)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("method = %s, want %s", r.Method, tt.method)
				}
				if r.URL.Path != "/alerts/v3/alert-rules/ar_123/notifications/links" {
					t.Fatalf("path = %s, want notification links path", r.URL.Path)
				}
				if got := r.URL.Query().Get("teamId"); got != "team_123" {
					t.Fatalf("teamId = %q, want team_123", got)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				if len(body) != len(tt.wantBody) {
					t.Fatalf("body = %#v, want %#v", body, tt.wantBody)
				}
				for key, value := range tt.wantBody {
					if body[key] != value {
						t.Fatalf("body[%q] = %q, want %q", key, body[key], value)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"success":true}`)
			}))
			t.Cleanup(server.Close)

			request := AlertRuleNotificationRequest{
				TeamID:                      "team_123",
				AlertRuleID:                 "ar_123",
				AlertRuleNotificationTarget: tt.target,
			}
			if err := tt.mutate(New("TOKEN").WithBaseURL(server.URL), context.Background(), request); err != nil {
				t.Fatalf("mutation error = %v", err)
			}
		})
	}
}
