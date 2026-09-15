package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func alertRuleNotificationSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	newAlertRuleNotificationResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}
	return response.Schema
}

func TestAlertRuleNotificationSchema(t *testing.T) {
	if _, ok := newAlertRuleNotificationResource().(resource.ResourceWithModifyPlan); !ok {
		t.Fatal("alert rule notification must implement ResourceWithModifyPlan to hide fields for the other destination")
	}

	resourceSchema := alertRuleNotificationSchema(t)
	if !resourceSchema.Attributes["alert_rule_id"].IsRequired() {
		t.Fatal("alert_rule_id must be required")
	}
	for _, name := range []string{"webhook_id", "slack_channel_id", "slack_installation_id"} {
		if !resourceSchema.Attributes[name].IsOptional() {
			t.Fatalf("%s must be optional", name)
		}
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		if len(attribute.PlanModifiers) == 0 {
			t.Fatalf("%s must require replacement", name)
		}
	}
	if !resourceSchema.Attributes["slack_installation_id"].IsComputed() {
		t.Fatal("slack_installation_id must be computed")
	}
	if len((&alertRuleNotificationResource{}).ConfigValidators(context.Background())) == 0 {
		t.Fatal("resource must validate the mutually exclusive destinations")
	}
}

func TestAlertRuleNotificationModifyPlanHidesOtherDestinationFields(t *testing.T) {
	tests := []struct {
		name   string
		config AlertRuleNotification
		plan   AlertRuleNotification
		check  func(*testing.T, AlertRuleNotification)
	}{
		{
			name: "webhook replacement hides Slack installation",
			config: AlertRuleNotification{
				ID:                  types.StringNull(),
				TeamID:              types.StringNull(),
				AlertRuleID:         types.StringValue("ar_123"),
				WebhookID:           types.StringUnknown(),
				SlackChannelID:      types.StringNull(),
				SlackInstallationID: types.StringNull(),
			},
			plan: AlertRuleNotification{
				ID:                  types.StringUnknown(),
				TeamID:              types.StringUnknown(),
				AlertRuleID:         types.StringValue("ar_123"),
				WebhookID:           types.StringUnknown(),
				SlackChannelID:      types.StringNull(),
				SlackInstallationID: types.StringUnknown(),
			},
			check: func(t *testing.T, modified AlertRuleNotification) {
				t.Helper()
				if !modified.SlackInstallationID.IsNull() {
					t.Fatalf("slack_installation_id = %v, want null", modified.SlackInstallationID)
				}
			},
		},
		{
			name: "Slack replacement hides webhook",
			config: AlertRuleNotification{
				ID:                  types.StringNull(),
				TeamID:              types.StringNull(),
				AlertRuleID:         types.StringValue("ar_123"),
				WebhookID:           types.StringNull(),
				SlackChannelID:      types.StringUnknown(),
				SlackInstallationID: types.StringNull(),
			},
			plan: AlertRuleNotification{
				ID:                  types.StringUnknown(),
				TeamID:              types.StringUnknown(),
				AlertRuleID:         types.StringValue("ar_123"),
				WebhookID:           types.StringUnknown(),
				SlackChannelID:      types.StringUnknown(),
				SlackInstallationID: types.StringUnknown(),
			},
			check: func(t *testing.T, modified AlertRuleNotification) {
				t.Helper()
				if !modified.WebhookID.IsNull() {
					t.Fatalf("webhook_id = %v, want null", modified.WebhookID)
				}
				if !modified.SlackInstallationID.IsUnknown() {
					t.Fatalf("slack_installation_id = %v, want unknown for API resolution", modified.SlackInstallationID)
				}
			},
		},
	}

	ctx := context.Background()
	resourceSchema := alertRuleNotificationSchema(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPlan := tfsdk.Plan{Schema: resourceSchema}
			if diags := configPlan.Set(ctx, tt.config); diags.HasError() {
				t.Fatalf("config Plan.Set() diagnostics = %v", diags)
			}
			plan := tfsdk.Plan{Schema: resourceSchema}
			if diags := plan.Set(ctx, tt.plan); diags.HasError() {
				t.Fatalf("Plan.Set() diagnostics = %v", diags)
			}

			response := &resource.ModifyPlanResponse{Plan: plan}
			(&alertRuleNotificationResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{
				Config: tfsdk.Config{Raw: configPlan.Raw, Schema: resourceSchema},
				Plan:   plan,
			}, response)
			if response.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan() diagnostics = %v", response.Diagnostics)
			}

			var modified AlertRuleNotification
			if diags := response.Plan.Get(ctx, &modified); diags.HasError() {
				t.Fatalf("modified Plan.Get() diagnostics = %v", diags)
			}
			tt.check(t, modified)
		})
	}
}

func TestAlertRuleNotificationTarget(t *testing.T) {
	tests := []struct {
		name    string
		model   AlertRuleNotification
		want    client.AlertRuleNotificationTarget
		wantErr bool
	}{
		{
			name: "webhook",
			model: AlertRuleNotification{
				WebhookID:           types.StringValue("hook_123"),
				SlackChannelID:      types.StringNull(),
				SlackInstallationID: types.StringNull(),
			},
			want: client.AlertRuleNotificationTarget{Type: client.AlertRuleNotificationTypeWebhook, WebhookID: "hook_123"},
		},
		{
			name: "Slack",
			model: AlertRuleNotification{
				WebhookID:           types.StringNull(),
				SlackChannelID:      types.StringValue("C123"),
				SlackInstallationID: types.StringValue("icfg_123"),
			},
			want: client.AlertRuleNotificationTarget{Type: client.AlertRuleNotificationTypeSlack, ConfigID: "icfg_123", ChannelID: "C123"},
		},
		{
			name: "Slack with inferred installation",
			model: AlertRuleNotification{
				WebhookID:           types.StringNull(),
				SlackChannelID:      types.StringValue("C123"),
				SlackInstallationID: types.StringNull(),
			},
			want: client.AlertRuleNotificationTarget{Type: client.AlertRuleNotificationTypeSlack, ChannelID: "C123"},
		},
		{
			name: "Slack installation without channel",
			model: AlertRuleNotification{
				WebhookID:           types.StringNull(),
				SlackChannelID:      types.StringNull(),
				SlackInstallationID: types.StringValue("icfg_123"),
			},
			wantErr: true,
		},
		{
			name: "multiple destinations",
			model: AlertRuleNotification{
				WebhookID:           types.StringValue("hook_123"),
				SlackChannelID:      types.StringValue("C123"),
				SlackInstallationID: types.StringValue("icfg_123"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.model.target()
			if (err != nil) != tt.wantErr {
				t.Fatalf("target() error = %v, wantErr %t", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("target() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAlertRuleNotificationExistsMatchesDestinationIdentity(t *testing.T) {
	notifications := []client.AlertRuleNotification{
		{Type: client.AlertRuleNotificationTypeSlack, ConfigID: "icfg_other", ChannelID: "C123"},
		{Type: client.AlertRuleNotificationTypeSlack, ConfigID: "icfg_123", ChannelID: "C123"},
		{Type: client.AlertRuleNotificationTypeWebhook, Webhook: client.AlertRuleNotificationWebhook{ID: "hook_123"}},
	}

	if !alertRuleNotificationExists(client.AlertRuleNotificationTarget{
		Type: client.AlertRuleNotificationTypeSlack, ConfigID: "icfg_123", ChannelID: "C123",
	}, notifications) {
		t.Fatal("matching Slack destination was not found")
	}
	if alertRuleNotificationExists(client.AlertRuleNotificationTarget{
		Type: client.AlertRuleNotificationTypeSlack, ConfigID: "icfg_missing", ChannelID: "C123",
	}, notifications) {
		t.Fatal("Slack destination with a different installation was matched")
	}
	if !alertRuleNotificationExists(client.AlertRuleNotificationTarget{
		Type: client.AlertRuleNotificationTypeWebhook, WebhookID: "hook_123",
	}, notifications) {
		t.Fatal("matching webhook destination was not found")
	}
}

func TestAlertRuleNotificationCreate(t *testing.T) {
	var body map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/alerts/v3/alert-rules/ar_123/notifications/links" {
			t.Fatalf("request = %s %s, want notification link POST", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"success":true,"notification":{"type":"webhook","webhookId":"hook_123"}}`)
	}))
	t.Cleanup(server.Close)

	resourceSchema := alertRuleNotificationSchema(t)
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(context.Background(), AlertRuleNotification{
		ID:                  types.StringUnknown(),
		TeamID:              types.StringNull(),
		AlertRuleID:         types.StringValue("ar_123"),
		WebhookID:           types.StringValue("hook_123"),
		SlackChannelID:      types.StringNull(),
		SlackInstallationID: types.StringNull(),
	}); diags.HasError() {
		t.Fatalf("Plan.Set() diagnostics = %v", diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&alertRuleNotificationResource{
		client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	}).Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}

	var state AlertRuleNotification
	if diags := response.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("State.Get() diagnostics = %v", diags)
	}
	if got := state.ID.ValueString(); got != "ar_123/webhook/hook_123" {
		t.Fatalf("id = %q, want ar_123/webhook/hook_123", got)
	}
	if got := state.TeamID.ValueString(); got != "team_123" {
		t.Fatalf("team_id = %q, want team_123", got)
	}
	if body["type"] != "webhook" || body["webhookId"] != "hook_123" {
		t.Fatalf("body = %#v", body)
	}
}

func TestAlertRuleNotificationCreateResolvesSlackInstallation(t *testing.T) {
	var body map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"success":true,"notification":{"type":"slack","configId":"icfg_123","channelId":"C123"}}`)
	}))
	t.Cleanup(server.Close)

	resourceSchema := alertRuleNotificationSchema(t)
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(context.Background(), AlertRuleNotification{
		ID:                  types.StringUnknown(),
		TeamID:              types.StringNull(),
		AlertRuleID:         types.StringValue("ar_123"),
		WebhookID:           types.StringNull(),
		SlackChannelID:      types.StringValue("C123"),
		SlackInstallationID: types.StringUnknown(),
	}); diags.HasError() {
		t.Fatalf("Plan.Set() diagnostics = %v", diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&alertRuleNotificationResource{
		client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	}).Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}

	var state AlertRuleNotification
	if diags := response.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("State.Get() diagnostics = %v", diags)
	}
	if got := state.ID.ValueString(); got != "ar_123/slack/icfg_123/C123" {
		t.Fatalf("id = %q, want ar_123/slack/icfg_123/C123", got)
	}
	if got := state.SlackInstallationID.ValueString(); got != "icfg_123" {
		t.Fatalf("slack_installation_id = %q, want icfg_123", got)
	}
	if _, ok := body["configId"]; ok {
		t.Fatalf("request unexpectedly included configId: %#v", body)
	}
}

func TestAlertRuleNotificationRead(t *testing.T) {
	tests := []struct {
		name             string
		status           int
		response         string
		expectRemoved    bool
		expectDiagnostic bool
	}{
		{
			name:     "linked webhook remains in state",
			status:   http.StatusOK,
			response: `{"notifications":[{"type":"webhook","webhook":{"id":"hook_123"}}]}`,
		},
		{
			name:          "missing link removes state",
			status:        http.StatusOK,
			response:      `{"notifications":[]}`,
			expectRemoved: true,
		},
		{
			name:          "missing rule removes state",
			status:        http.StatusNotFound,
			response:      `{"error":{"code":"not_found","message":"Alert rule not found."}}`,
			expectRemoved: true,
		},
		{
			name:             "transient failure preserves state",
			status:           http.StatusBadGateway,
			response:         `{"error":{"code":"bad_gateway","message":"Unable to read Slack subscriptions."}}`,
			expectDiagnostic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.response)
			}))
			t.Cleanup(server.Close)

			resourceSchema := alertRuleNotificationSchema(t)
			prior := AlertRuleNotification{
				ID:                  types.StringValue("ar_123/webhook/hook_123"),
				TeamID:              types.StringValue("team_123"),
				AlertRuleID:         types.StringValue("ar_123"),
				WebhookID:           types.StringValue("hook_123"),
				SlackChannelID:      types.StringNull(),
				SlackInstallationID: types.StringNull(),
			}
			state := tfsdk.State{Schema: resourceSchema}
			if diags := state.Set(context.Background(), prior); diags.HasError() {
				t.Fatalf("State.Set() diagnostics = %v", diags)
			}
			response := resource.ReadResponse{State: state}
			(&alertRuleNotificationResource{
				client: client.New("TOKEN").WithBaseURL(server.URL),
			}).Read(context.Background(), resource.ReadRequest{State: state}, &response)

			if response.Diagnostics.HasError() != tt.expectDiagnostic {
				t.Fatalf("diagnostics error = %t, want %t: %v", response.Diagnostics.HasError(), tt.expectDiagnostic, response.Diagnostics)
			}
			if response.State.Raw.IsNull() != tt.expectRemoved {
				t.Fatalf("state removed = %t, want %t", response.State.Raw.IsNull(), tt.expectRemoved)
			}
		})
	}
}

func TestAlertRuleNotificationImportState(t *testing.T) {
	tests := []struct {
		name           string
		importID       string
		configuredTeam client.Team
		response       string
		wantID         string
		wantTeamID     string
	}{
		{
			name:       "webhook with explicit team",
			importID:   "team_123/ar_123/webhook/hook_123",
			response:   `{"notifications":[{"type":"webhook","webhook":{"id":"hook_123"}}]}`,
			wantID:     "ar_123/webhook/hook_123",
			wantTeamID: "team_123",
		},
		{
			name:           "Slack with provider team",
			importID:       "ar_default/slack/icfg_123/C123",
			configuredTeam: client.Team{ID: "team_456"},
			response:       `{"notifications":[{"type":"slack","configId":"icfg_123","channelId":"C123"}]}`,
			wantID:         "ar_default/slack/icfg_123/C123",
			wantTeamID:     "team_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("teamId"); got != tt.wantTeamID {
					t.Fatalf("teamId = %q, want %q", got, tt.wantTeamID)
				}
				_, _ = fmt.Fprint(w, tt.response)
			}))
			t.Cleanup(server.Close)

			resourceSchema := alertRuleNotificationSchema(t)
			response := resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
			(&alertRuleNotificationResource{
				client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(tt.configuredTeam),
			}).ImportState(context.Background(), resource.ImportStateRequest{ID: tt.importID}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("ImportState() diagnostics = %v", response.Diagnostics)
			}

			var state AlertRuleNotification
			if diags := response.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("State.Get() diagnostics = %v", diags)
			}
			if got := state.ID.ValueString(); got != tt.wantID {
				t.Fatalf("id = %q, want %q", got, tt.wantID)
			}
			if got := state.TeamID.ValueString(); got != tt.wantTeamID {
				t.Fatalf("team_id = %q, want %q", got, tt.wantTeamID)
			}
		})
	}
}

func TestAlertRuleNotificationImportStateRejectsMalformedIDs(t *testing.T) {
	for _, importID := range []string{
		"",
		"ar_123/webhook",
		"ar_123/unknown/destination",
		"team_123/ar_123/slack/icfg_123",
		"team_123/ar_123/slack/icfg_123/C123/extra",
	} {
		t.Run(importID, func(t *testing.T) {
			response := &resource.ImportStateResponse{}
			(&alertRuleNotificationResource{}).ImportState(
				context.Background(),
				resource.ImportStateRequest{ID: importID},
				response,
			)
			if !response.Diagnostics.HasError() {
				t.Fatal("ImportState() returned no diagnostics")
			}
		})
	}
}
