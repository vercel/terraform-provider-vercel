package client

import (
	"context"
	"encoding/json"
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

func getUpdateBypassProtectionRequestBody(newValue bool, secret string) string {
	if newValue {
		return "{}"
	}

	bytes, err := json.Marshal(struct {
		Revoke revokeBypassProtectionRequest `json:"revoke"`
	}{
		Revoke: revokeBypassProtectionRequest{
			Regenerate: false,
			Secret:     secret,
		},
	})
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func (c *Client) UpdateProtectionBypassForAutomation(ctx context.Context, request UpdateProtectionBypassForAutomationRequest) (s string, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/protection-bypass", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := getUpdateBypassProtectionRequestBody(request.NewValue, request.Secret)
	tflog.Error(ctx, "creating project domain", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	response := struct {
		ProtectionBypass map[string]ProtectionBypass `json:"protectionBypass"`
	}{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)

	if err != nil {
		return s, fmt.Errorf("unable to add protection bypass for automation: %w", err)
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
