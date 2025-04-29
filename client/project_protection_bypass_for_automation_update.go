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
		return (string(mustMarshal(struct {
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
		})))
	}

	if setSecretWithValue {
		return (string(mustMarshal(struct {
			Generate generateBypassProtectionRequest `json:"generate"`
		}{
			Generate: generateBypassProtectionRequest{
				Secret: newSecret,
			},
		})))
	}

	if revokeOldSecret {
		return (string(mustMarshal(struct {
			Revoke revokeBypassProtectionRequest `json:"revoke"`
		}{
			Revoke: revokeBypassProtectionRequest{
				Regenerate: false,
				Secret:     oldSecret,
			},
		})))
	}

	// the default behaviour creates a new secret with a generated value
	return "{}"
}

func (c *Client) UpdateProtectionBypassForAutomation(ctx context.Context, request UpdateProtectionBypassForAutomationRequest) (s string, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/protection-bypass", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
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

	if len(response.ProtectionBypass) != 1 {
		return s, fmt.Errorf("error adding protection bypass for automation: the response contained an unexpected number of items (%d)", len(response.ProtectionBypass))
	}

	// return the first key from the map
	for key := range response.ProtectionBypass {
		return key, err
	}

	return s, err
}
