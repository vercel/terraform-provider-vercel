package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMonitorNotifications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.EscapedPath(); got != "/alerts/v3/monitors/ar_custom%2Fvalue/notifications" {
			t.Fatalf("path = %s, want escaped monitor path", got)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"notifications": [
				{"type":"slack","configId":"icfg_123","channelId":"C123"},
				{"type":"webhook","webhook":{"id":"hook_123"},"minimumSeverityLevel":"critical"}
			]
		}`)
	}))
	t.Cleanup(server.Close)

	notifications, err := New("TOKEN").WithBaseURL(server.URL).GetMonitorNotifications(context.Background(), "ar_custom/value", "team_123")
	if err != nil {
		t.Fatalf("GetMonitorNotifications() error = %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("len(notifications) = %d, want 2", len(notifications))
	}
	if notifications[0].Type != AlertRuleNotificationTypeSlack || notifications[0].ConfigID != "icfg_123" || notifications[0].ChannelID != "C123" {
		t.Fatalf("Slack notification = %#v", notifications[0])
	}
	if notifications[1].Type != AlertRuleNotificationTypeWebhook || notifications[1].Webhook.ID != "hook_123" || notifications[1].MinimumSeverityLevel == nil || *notifications[1].MinimumSeverityLevel != "critical" {
		t.Fatalf("webhook notification = %#v", notifications[1])
	}
}

func TestMutateMonitorNotification(t *testing.T) {
	critical := "critical"
	tests := []struct {
		name     string
		method   string
		target   AlertRuleNotificationTarget
		wantBody map[string]any
		response string
		want     AlertRuleNotificationTarget
		mutate   func(*Client, context.Context, MonitorNotificationRequest) (AlertRuleNotificationTarget, error)
	}{
		{
			name:     "link Slack channel",
			method:   http.MethodPost,
			target:   AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeSlack, ChannelID: "C123"},
			wantBody: map[string]any{"type": "slack", "channelId": "C123"},
			response: `{"success":true,"notification":{"type":"slack","configId":"icfg_123","channelId":"C123"}}`,
			want:     AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeSlack, ConfigID: "icfg_123", ChannelID: "C123"},
			mutate: func(c *Client, ctx context.Context, request MonitorNotificationRequest) (AlertRuleNotificationTarget, error) {
				return c.LinkMonitorNotification(ctx, request)
			},
		},
		{
			name:     "update webhook severity",
			method:   http.MethodPatch,
			target:   AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeWebhook, WebhookID: "hook_123", MinimumSeverityLevel: &critical},
			wantBody: map[string]any{"type": "webhook", "webhookId": "hook_123", "minimumSeverityLevel": "critical"},
			response: `{"success":true,"notification":{"type":"webhook","webhookId":"hook_123","minimumSeverityLevel":"critical"}}`,
			want:     AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeWebhook, WebhookID: "hook_123", MinimumSeverityLevel: &critical},
			mutate: func(c *Client, ctx context.Context, request MonitorNotificationRequest) (AlertRuleNotificationTarget, error) {
				return c.UpdateMonitorNotification(ctx, request)
			},
		},
		{
			name:     "unlink webhook",
			method:   http.MethodDelete,
			target:   AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeWebhook, WebhookID: "hook_123", MinimumSeverityLevel: &critical},
			wantBody: map[string]any{"type": "webhook", "webhookId": "hook_123"},
			response: `{"success":true,"notification":{"type":"webhook","webhookId":"hook_123"}}`,
			want:     AlertRuleNotificationTarget{Type: AlertRuleNotificationTypeWebhook, WebhookID: "hook_123"},
			mutate: func(c *Client, ctx context.Context, request MonitorNotificationRequest) (AlertRuleNotificationTarget, error) {
				return c.UnlinkMonitorNotification(ctx, request)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("method = %s, want %s", r.Method, tt.method)
				}
				if r.URL.Path != "/alerts/v3/monitors/ar_123/notifications/links" {
					t.Fatalf("path = %s, want monitor notification links path", r.URL.Path)
				}
				if got := r.URL.Query().Get("teamId"); got != "team_123" {
					t.Fatalf("teamId = %q, want team_123", got)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				if len(body) != len(tt.wantBody) {
					t.Fatalf("body = %#v, want %#v", body, tt.wantBody)
				}
				for key, value := range tt.wantBody {
					if body[key] != value {
						t.Fatalf("body[%q] = %v, want %v", key, body[key], value)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, tt.response)
			}))
			t.Cleanup(server.Close)

			got, err := tt.mutate(New("TOKEN").WithBaseURL(server.URL), context.Background(), MonitorNotificationRequest{
				TeamID:                      "team_123",
				MonitorID:                   "ar_123",
				AlertRuleNotificationTarget: tt.target,
			})
			if err != nil {
				t.Fatalf("mutation error = %v", err)
			}
			if !optionalStringsEqualForTest(got.MinimumSeverityLevel, tt.want.MinimumSeverityLevel) {
				t.Fatalf("mutation target = %#v, want %#v", got, tt.want)
			}
			got.MinimumSeverityLevel, tt.want.MinimumSeverityLevel = nil, nil
			if got != tt.want {
				t.Fatalf("mutation target = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func optionalStringsEqualForTest(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
