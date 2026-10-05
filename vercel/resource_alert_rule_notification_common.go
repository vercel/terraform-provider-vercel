package vercel

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func alertRuleNotificationCommonAttributes(idDescription string) map[string]schema.Attribute {
	identityPlanModifiers := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	identityValidators := []validator.String{
		stringvalidator.LengthBetween(1, 256),
		validateStringIsTrimmed(),
	}

	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: idDescription,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
		},
		"team_id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The ID of the team that owns the alert rule and notification destination. Required if a default team is not configured in the provider.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
				stringplanmodifier.UseNonNullStateForUnknown(),
			},
		},
		"alert_rule_id": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The ID of the alert rule. Use `ar_default` to manage a notification destination for the team's default rule.",
			PlanModifiers:       identityPlanModifiers,
			Validators:          identityValidators,
		},
	}
}

func configureAlertRuleNotificationResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*client.Client, bool) {
	if req.ProviderData == nil {
		return nil, false
	}

	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil, false
	}
	return configuredClient, true
}

func optionalStringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func alertRuleNotificationExists(target client.AlertRuleNotificationTarget, notifications []client.AlertRuleNotification) bool {
	for _, notification := range notifications {
		switch target.Type {
		case client.AlertRuleNotificationTypeWebhook:
			if notification.Type == target.Type && notification.Webhook.ID == target.WebhookID {
				return true
			}
		case client.AlertRuleNotificationTypeSlack:
			if notification.Type == target.Type && notification.ConfigID == target.ConfigID && notification.ChannelID == target.ChannelID {
				return true
			}
		}
	}
	return false
}

func validAlertRuleNotificationImportParts(importID string, allowedLengths ...int) ([]string, bool) {
	parts := strings.Split(importID, "/")
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
	}
	for _, allowedLength := range allowedLengths {
		if len(parts) == allowedLength {
			return parts, true
		}
	}
	return nil, false
}
