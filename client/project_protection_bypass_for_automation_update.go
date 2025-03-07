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
	Secret    string
}

type revokeBypassProtectionRequest struct {
	Regenerate bool   `json:"regenerate"`
	Secret     string `json:"secret"`
}

type generateBypassProtectionRequest struct {
	Secret string `json:"secret"`
}

func getUpdateBypassProtectionRequestBody(newValue bool, secret string) string {
	if newValue {
		if secret == "" {
			return "{}"
		}
		return string(mustMarshal(struct {
			Revoke generateBypassProtectionRequest `json:"generate"`
		}{
			Revoke: generateBypassProtectionRequest{
				Secret: secret,
			},
		}))
	}

	return string(mustMarshal(struct {
		Revoke revokeBypassProtectionRequest `json:"revoke"`
	}{
		Revoke: revokeBypassProtectionRequest{
			Regenerate: false,
			Secret:     secret,
		},
	}))
}

func (c *Client) UpdateProtectionBypassForAutomation(ctx context.Context, request UpdateProtectionBypassForAutomationRequest) (s string, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/protection-bypass", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := getUpdateBypassProtectionRequestBody(request.NewValue, request.Secret)
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
