package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type UpdateProtectionBypassForAutomationRequest struct {
	TeamID    string
	ProjectID string
	NewValue  bool
	NewSecret string
	OldSecret string
}

type revokeBypassProtectionRequest struct {
	Regenerate bool   `json:"regenerate"`
	Secret     string `json:"secret"`
}

type generateBypassProtectionRequest struct {
	Secret string `json:"secret"`
}

func getUpdateBypassProtectionRequestBody(newValue bool, newSecret string, oldSecret string) string {
	setSecretWithValue := newValue && newSecret != ""
	revokeOldSecret := oldSecret != ""

	if setSecretWithValue && revokeOldSecret {
		return string(mustMarshal(struct {
			Generate generateBypassProtectionRequest `json:"generate"`
			Revoke   revokeBypassProtectionRequest   `json:"revoke"`
		}{
			Generate: generateBypassProtectionRequest{
				Secret: newSecret,
			},
			Revoke: revokeBypassProtectionRequest{
				Regenerate: true,
				Secret:     oldSecret,
			},
		}))
	}

	if setSecretWithValue {
		return string(mustMarshal(struct {
			Generate generateBypassProtectionRequest `json:"generate"`
		}{
			Generate: generateBypassProtectionRequest{
				Secret: newSecret,
			},
		}))
	}

	if revokeOldSecret {
		return string(mustMarshal(struct {
			Revoke revokeBypassProtectionRequest `json:"revoke"`
		}{
			Revoke: revokeBypassProtectionRequest{
				Regenerate: false,
				Secret:     oldSecret,
			},
		}))
	}

	// the default behaviour creates a new secret with a generated value
	return "{}"
}

func (c *Client) UpdateProtectionBypassForAutomation(ctx context.Context, request UpdateProtectionBypassForAutomationRequest) (s string, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/protection-bypass", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	payload := getUpdateBypassProtectionRequestBody(request.NewValue, request.NewSecret, request.OldSecret)
	tflog.Info(ctx, "updating protection bypass", map[string]any{
		"url":      url,
		"payload":  payload,
		"newValue": request.NewValue,
	})
	response := struct {
		ProtectionBypass map[string]ProtectionBypass `json:"protectionBypass"`
	}{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &response)

	if err != nil {
		return s, fmt.Errorf("unable to update protection bypass for automation: %w", err)
	}

	if !request.NewValue {
		return
	}

	// When the caller supplied an explicit new secret we already know what was
	// set — no need to inspect the response, which can contain entries for
	// other bypass scopes (e.g. shareable-link) or both the old and new
	// automation-bypass during a rotation.
	if request.NewSecret != "" {
		return request.NewSecret, nil
	}

	// The API generated a secret for us. Filter to the automation-bypass scope
	// since the project may have other bypass scopes (e.g. shareable-link) in
	// the same response map.
	var automationKey string
	var automationCount int
	for key, bypass := range response.ProtectionBypass {
		if bypass.Scope == "automation-bypass" {
			automationKey = key
			automationCount++
		}
	}

	if automationCount != 1 {
		return s, fmt.Errorf("error adding protection bypass for automation: the response contained an unexpected number of automation-bypass items (%d)", automationCount)
	}

	return automationKey, nil
}
