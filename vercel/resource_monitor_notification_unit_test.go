package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func monitorWebhookNotificationSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	newMonitorWebhookNotificationResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}
	return response.Schema
}

func monitorSlackNotificationSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	newMonitorSlackNotificationResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}
	return response.Schema
}

func TestMonitorNotificationSchemasAreDestinationSpecific(t *testing.T) {
	webhookSchema := monitorWebhookNotificationSchema(t)
	if !webhookSchema.Attributes["webhook_id"].IsRequired() {
		t.Fatal("webhook_id must be required")
	}
	for _, name := range []string{"slack_channel_id", "slack_installation_id"} {
		if _, ok := webhookSchema.Attributes[name]; ok {
			t.Fatalf("webhook resource unexpectedly has %s", name)
		}
	}

	slackSchema := monitorSlackNotificationSchema(t)
	if !slackSchema.Attributes["slack_channel_id"].IsRequired() {
		t.Fatal("slack_channel_id must be required")
	}
	if !slackSchema.Attributes["slack_installation_id"].IsOptional() || !slackSchema.Attributes["slack_installation_id"].IsComputed() {
		t.Fatal("slack_installation_id must be optional and computed")
	}
	if _, ok := slackSchema.Attributes["webhook_id"]; ok {
		t.Fatal("Slack resource unexpectedly has webhook_id")
	}
}

func TestMonitorWebhookNotificationCreate(t *testing.T) {
	var body map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/alerts/v3/monitors/ar_123/notifications/links" {
			t.Fatalf("request = %s %s, want notification link POST", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"success":true,"notification":{"type":"webhook","webhookId":"hook_123","minimumSeverityLevel":"critical"}}`)
	}))
	t.Cleanup(server.Close)

	resourceSchema := monitorWebhookNotificationSchema(t)
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(context.Background(), MonitorWebhookNotification{
		ID:                   types.StringUnknown(),
		TeamID:               types.StringNull(),
		MonitorID:            types.StringValue("ar_123"),
		WebhookID:            types.StringValue("hook_123"),
		MinimumSeverityLevel: types.StringValue("critical"),
	}); diags.HasError() {
		t.Fatalf("Plan.Set() diagnostics = %v", diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&monitorWebhookNotificationResource{
		client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	}).Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}

	var state MonitorWebhookNotification
	if diags := response.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("State.Get() diagnostics = %v", diags)
	}
	if got := state.ID.ValueString(); got != "ar_123/hook_123" {
		t.Fatalf("id = %q, want ar_123/hook_123", got)
	}
	if got := state.TeamID.ValueString(); got != "team_123" {
		t.Fatalf("team_id = %q, want team_123", got)
	}
	if body["type"] != "webhook" || body["webhookId"] != "hook_123" || body["minimumSeverityLevel"] != "critical" {
		t.Fatalf("body = %#v", body)
	}
	if got := state.MinimumSeverityLevel.ValueString(); got != "critical" {
		t.Fatalf("minimum_severity_level = %q, want critical", got)
	}
}

func TestMonitorSlackNotificationCreateResolvesInstallation(t *testing.T) {
	var body map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = fmt.Fprint(w, `{"success":true,"notification":{"type":"slack","configId":"icfg_123","channelId":"C123"}}`)
	}))
	t.Cleanup(server.Close)

	resourceSchema := monitorSlackNotificationSchema(t)
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(context.Background(), MonitorSlackNotification{
		ID:                  types.StringUnknown(),
		TeamID:              types.StringNull(),
		MonitorID:           types.StringValue("ar_123"),
		SlackChannelID:      types.StringValue("C123"),
		SlackInstallationID: types.StringUnknown(),
	}); diags.HasError() {
		t.Fatalf("Plan.Set() diagnostics = %v", diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&monitorSlackNotificationResource{
		client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	}).Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}

	var state MonitorSlackNotification
	if diags := response.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("State.Get() diagnostics = %v", diags)
	}
	if got := state.ID.ValueString(); got != "ar_123/icfg_123/C123" {
		t.Fatalf("id = %q, want ar_123/icfg_123/C123", got)
	}
	if got := state.SlackInstallationID.ValueString(); got != "icfg_123" {
		t.Fatalf("slack_installation_id = %q, want icfg_123", got)
	}
	if _, ok := body["configId"]; ok {
		t.Fatalf("request unexpectedly included configId: %#v", body)
	}
	if _, ok := body["minimumSeverityLevel"]; ok {
		t.Fatalf("request unexpectedly included minimumSeverityLevel: %#v", body)
	}
	if !state.MinimumSeverityLevel.IsNull() {
		t.Fatalf("minimum_severity_level = %s, want null", state.MinimumSeverityLevel)
	}
}

func TestMonitorNotificationCreateClearsAdoptedMinimumSeverityLevel(t *testing.T) {
	tests := []struct {
		name         string
		slack        bool
		linkResponse string
		wantPatch    string
	}{
		{
			name:         "Slack",
			slack:        true,
			linkResponse: `{"success":true,"notification":{"type":"slack","configId":"icfg_123","channelId":"C123","minimumSeverityLevel":"critical"}}`,
			wantPatch:    `{"type":"slack","configId":"icfg_123","channelId":"C123","minimumSeverityLevel":null}`,
		},
		{
			name:         "webhook",
			linkResponse: `{"success":true,"notification":{"type":"webhook","webhookId":"hook_123","minimumSeverityLevel":"critical"}}`,
			wantPatch:    `{"type":"webhook","webhookId":"hook_123","minimumSeverityLevel":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var methods []string
			var patchBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/alerts/v3/monitors/ar_123/notifications/links" {
					t.Fatalf("path = %s, want notification links", r.URL.Path)
				}
				methods = append(methods, r.Method)
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read request: %v", err)
				}
				switch r.Method {
				case http.MethodPost:
					_, _ = fmt.Fprint(w, tt.linkResponse)
				case http.MethodPatch:
					patchBody = string(raw)
					_, _ = fmt.Fprint(w, `{"success":true,"notification":{}}`)
				default:
					t.Fatalf("unexpected method %s", r.Method)
				}
			}))
			t.Cleanup(server.Close)
			apiClient := client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"})

			var resourceSchema schema.Schema
			var plannedValue any
			if tt.slack {
				resourceSchema = monitorSlackNotificationSchema(t)
				plannedValue = MonitorSlackNotification{
					ID:                   types.StringUnknown(),
					TeamID:               types.StringNull(),
					MonitorID:            types.StringValue("ar_123"),
					SlackChannelID:       types.StringValue("C123"),
					SlackInstallationID:  types.StringUnknown(),
					MinimumSeverityLevel: types.StringNull(),
				}
			} else {
				resourceSchema = monitorWebhookNotificationSchema(t)
				plannedValue = MonitorWebhookNotification{
					ID:                   types.StringUnknown(),
					TeamID:               types.StringNull(),
					MonitorID:            types.StringValue("ar_123"),
					WebhookID:            types.StringValue("hook_123"),
					MinimumSeverityLevel: types.StringNull(),
				}
			}
			plan := tfsdk.Plan{Schema: resourceSchema}
			if diags := plan.Set(context.Background(), plannedValue); diags.HasError() {
				t.Fatalf("Plan.Set() diagnostics = %v", diags)
			}
			request := resource.CreateRequest{Plan: plan}
			response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
			if tt.slack {
				(&monitorSlackNotificationResource{client: apiClient}).Create(context.Background(), request, &response)
			} else {
				(&monitorWebhookNotificationResource{client: apiClient}).Create(context.Background(), request, &response)
			}
			if response.Diagnostics.HasError() {
				t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
			}
			if len(methods) != 2 || methods[0] != http.MethodPost || methods[1] != http.MethodPatch {
				t.Fatalf("methods = %v, want [POST PATCH]", methods)
			}
			if patchBody != tt.wantPatch {
				t.Fatalf("PATCH body = %s, want %s", patchBody, tt.wantPatch)
			}

			var got types.String
			if diags := response.State.GetAttribute(context.Background(), path.Root("minimum_severity_level"), &got); diags.HasError() {
				t.Fatalf("State.GetAttribute() diagnostics = %v", diags)
			}
			if !got.IsNull() {
				t.Fatalf("minimum_severity_level = %s, want null", got)
			}
		})
	}
}

func TestMonitorNotificationUpdateMinimumSeverityLevel(t *testing.T) {
	tests := []struct {
		name     string
		slack    bool
		prior    types.String
		planned  types.String
		wantBody string
	}{
		{
			name:     "Slack becomes Critical-only",
			slack:    true,
			prior:    types.StringNull(),
			planned:  types.StringValue("critical"),
			wantBody: `{"type":"slack","configId":"icfg_123","channelId":"C123","minimumSeverityLevel":"critical"}`,
		},
		{
			name:     "webhook clears Critical-only",
			prior:    types.StringValue("critical"),
			planned:  types.StringNull(),
			wantBody: `{"type":"webhook","webhookId":"hook_123","minimumSeverityLevel":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch || r.URL.Path != "/alerts/v3/monitors/ar_123/notifications/links" {
					t.Fatalf("request = %s %s, want notification link PATCH", r.Method, r.URL.Path)
				}
				if got := r.URL.Query().Get("teamId"); got != "team_123" {
					t.Fatalf("teamId = %q, want team_123", got)
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read request: %v", err)
				}
				body = string(raw)
				if tt.slack {
					_, _ = fmt.Fprint(w, `{"success":true,"notification":{"type":"slack","configId":"icfg_123","channelId":"C123","minimumSeverityLevel":"critical"}}`)
					return
				}
				_, _ = fmt.Fprint(w, `{"success":true,"notification":{"type":"webhook","webhookId":"hook_123"}}`)
			}))
			t.Cleanup(server.Close)
			apiClient := client.New("TOKEN").WithBaseURL(server.URL)

			var resourceSchema schema.Schema
			var priorValue, plannedValue any
			if tt.slack {
				resourceSchema = monitorSlackNotificationSchema(t)
				notification := MonitorSlackNotification{
					ID:                   types.StringValue("ar_123/icfg_123/C123"),
					TeamID:               types.StringValue("team_123"),
					MonitorID:            types.StringValue("ar_123"),
					SlackChannelID:       types.StringValue("C123"),
					SlackInstallationID:  types.StringValue("icfg_123"),
					MinimumSeverityLevel: tt.prior,
				}
				priorValue = notification
				notification.MinimumSeverityLevel = tt.planned
				plannedValue = notification
			} else {
				resourceSchema = monitorWebhookNotificationSchema(t)
				notification := MonitorWebhookNotification{
					ID:                   types.StringValue("ar_123/hook_123"),
					TeamID:               types.StringValue("team_123"),
					MonitorID:            types.StringValue("ar_123"),
					WebhookID:            types.StringValue("hook_123"),
					MinimumSeverityLevel: tt.prior,
				}
				priorValue = notification
				notification.MinimumSeverityLevel = tt.planned
				plannedValue = notification
			}

			state := tfsdk.State{Schema: resourceSchema}
			if diags := state.Set(context.Background(), priorValue); diags.HasError() {
				t.Fatalf("State.Set() diagnostics = %v", diags)
			}
			plan := tfsdk.Plan{Schema: resourceSchema}
			if diags := plan.Set(context.Background(), plannedValue); diags.HasError() {
				t.Fatalf("Plan.Set() diagnostics = %v", diags)
			}
			request := resource.UpdateRequest{Plan: plan, State: state}
			response := resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
			if tt.slack {
				(&monitorSlackNotificationResource{client: apiClient}).Update(context.Background(), request, &response)
			} else {
				(&monitorWebhookNotificationResource{client: apiClient}).Update(context.Background(), request, &response)
			}
			if response.Diagnostics.HasError() {
				t.Fatalf("Update() diagnostics = %v", response.Diagnostics)
			}
			if body != tt.wantBody {
				t.Fatalf("body = %s, want %s", body, tt.wantBody)
			}

			var got types.String
			if diags := response.State.GetAttribute(context.Background(), path.Root("minimum_severity_level"), &got); diags.HasError() {
				t.Fatalf("State.GetAttribute() diagnostics = %v", diags)
			}
			if !got.Equal(tt.planned) {
				t.Fatalf("minimum_severity_level = %s, want %s", got, tt.planned)
			}
		})
	}
}

func TestMonitorWebhookNotificationRead(t *testing.T) {
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
			response:      `{"error":{"code":"not_found","message":"Monitor not found."}}`,
			expectRemoved: true,
		},
		{
			name:             "transient failure preserves state",
			status:           http.StatusBadGateway,
			response:         `{"error":{"code":"bad_gateway","message":"Unable to read notifications."}}`,
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

			resourceSchema := monitorWebhookNotificationSchema(t)
			prior := MonitorWebhookNotification{
				ID:        types.StringValue("ar_123/hook_123"),
				TeamID:    types.StringValue("team_123"),
				MonitorID: types.StringValue("ar_123"),
				WebhookID: types.StringValue("hook_123"),
			}
			state := tfsdk.State{Schema: resourceSchema}
			if diags := state.Set(context.Background(), prior); diags.HasError() {
				t.Fatalf("State.Set() diagnostics = %v", diags)
			}
			response := resource.ReadResponse{State: state}
			(&monitorWebhookNotificationResource{
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

func TestMonitorNotificationImportState(t *testing.T) {
	tests := []struct {
		name           string
		importID       string
		configuredTeam client.Team
		response       string
		resourceSchema func(*testing.T) schema.Schema
		checkState     func(*testing.T, tfsdk.State)
		slack          bool
	}{
		{
			name:           "webhook with explicit team",
			importID:       "team_123/ar_123/hook_123",
			response:       `{"notifications":[{"type":"webhook","webhook":{"id":"hook_123"}}]}`,
			resourceSchema: monitorWebhookNotificationSchema,
			checkState: func(t *testing.T, state tfsdk.State) {
				var notification MonitorWebhookNotification
				if diags := state.Get(context.Background(), &notification); diags.HasError() {
					t.Fatalf("State.Get() diagnostics = %v", diags)
				}
				if got := notification.ID.ValueString(); got != "ar_123/hook_123" {
					t.Fatalf("id = %q, want ar_123/hook_123", got)
				}
				if got := notification.TeamID.ValueString(); got != "team_123" {
					t.Fatalf("team_id = %q, want team_123", got)
				}
			},
		},
		{
			name:           "Slack with provider team",
			importID:       "ar_default/icfg_123/C123",
			configuredTeam: client.Team{ID: "team_456"},
			response:       `{"notifications":[{"type":"slack","configId":"icfg_123","channelId":"C123","minimumSeverityLevel":"critical"}]}`,
			resourceSchema: monitorSlackNotificationSchema,
			slack:          true,
			checkState: func(t *testing.T, state tfsdk.State) {
				var notification MonitorSlackNotification
				if diags := state.Get(context.Background(), &notification); diags.HasError() {
					t.Fatalf("State.Get() diagnostics = %v", diags)
				}
				if got := notification.ID.ValueString(); got != "ar_default/icfg_123/C123" {
					t.Fatalf("id = %q, want ar_default/icfg_123/C123", got)
				}
				if got := notification.TeamID.ValueString(); got != "team_456" {
					t.Fatalf("team_id = %q, want team_456", got)
				}
				if got := notification.MinimumSeverityLevel.ValueString(); got != "critical" {
					t.Fatalf("minimum_severity_level = %q, want critical", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, tt.response)
			}))
			t.Cleanup(server.Close)

			resourceSchema := tt.resourceSchema(t)
			response := resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
			if tt.slack {
				(&monitorSlackNotificationResource{
					client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(tt.configuredTeam),
				}).ImportState(context.Background(), resource.ImportStateRequest{ID: tt.importID}, &response)
			} else {
				(&monitorWebhookNotificationResource{
					client: client.New("TOKEN").WithBaseURL(server.URL),
				}).ImportState(context.Background(), resource.ImportStateRequest{ID: tt.importID}, &response)
			}
			if response.Diagnostics.HasError() {
				t.Fatalf("ImportState() diagnostics = %v", response.Diagnostics)
			}
			tt.checkState(t, response.State)
		})
	}
}

func TestMonitorNotificationImportStateRejectsMalformedIDs(t *testing.T) {
	tests := []struct {
		name       string
		resource   resource.ResourceWithImportState
		resourceID string
	}{
		{name: "empty webhook", resource: &monitorWebhookNotificationResource{}, resourceID: ""},
		{name: "webhook has too many parts", resource: &monitorWebhookNotificationResource{}, resourceID: "team/ar/hook/extra"},
		{name: "Slack missing channel", resource: &monitorSlackNotificationResource{}, resourceID: "ar/icfg"},
		{name: "Slack has too many parts", resource: &monitorSlackNotificationResource{}, resourceID: "team/ar/icfg/C123/extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &resource.ImportStateResponse{}
			tt.resource.ImportState(context.Background(), resource.ImportStateRequest{ID: tt.resourceID}, response)
			if !response.Diagnostics.HasError() {
				t.Fatal("ImportState() returned no diagnostics")
			}
		})
	}
}
