package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type PatchProtectionBypassForAutomationRequest struct {
	TeamID    string
	ProjectID string
	Generate  *GenerateProtectionBypassRequest `json:"generate,omitempty"`
	Revoke    *RevokeProtectionBypassRequest   `json:"revoke,omitempty"`
	Update    *UpdateProtectionBypassRequest   `json:"update,omitempty"`
}

type patchProtectionBypassForAutomationBody struct {
	Generate *GenerateProtectionBypassRequest `json:"generate,omitempty"`
	Revoke   *RevokeProtectionBypassRequest   `json:"revoke,omitempty"`
	Update   *UpdateProtectionBypassRequest   `json:"update,omitempty"`
}

type RevokeProtectionBypassRequest struct {
	Regenerate bool   `json:"regenerate"`
	Secret     string `json:"secret"`
}

type GenerateProtectionBypassRequest struct {
	Secret string  `json:"secret,omitempty"`
	Note   *string `json:"note,omitempty"`
}

type UpdateProtectionBypassRequest struct {
	Secret   string  `json:"secret"`
	IsEnvVar *bool   `json:"isEnvVar,omitempty"`
	Note     *string `json:"note,omitempty"`
}

func (c *Client) PatchProtectionBypassForAutomation(ctx context.Context, request PatchProtectionBypassForAutomationRequest) (map[string]ProtectionBypass, error) {
	url := fmt.Sprintf("%s/v10/projects/%s/protection-bypass", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	payload := string(mustMarshal(patchProtectionBypassForAutomationBody{
		Generate: request.Generate,
		Revoke:   request.Revoke,
		Update:   request.Update,
	}))
	tflog.Info(ctx, "patching protection bypass", map[string]any{
		"url":     url,
		"payload": payload,
	})

	response := struct {
		ProtectionBypass map[string]ProtectionBypass `json:"protectionBypass"`
	}{}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &response)
	if err != nil {
		return nil, fmt.Errorf("unable to update protection bypass for automation: %w", err)
	}

	if response.ProtectionBypass == nil {
		return map[string]ProtectionBypass{}, nil
	}

	return response.ProtectionBypass, nil
}

type UpdateProtectionBypassForAutomationRequest struct {
	TeamID    string
	ProjectID string
	NewValue  bool
	NewSecret string
	OldSecret string
}

func (c *Client) UpdateProtectionBypassForAutomation(ctx context.Context, request UpdateProtectionBypassForAutomationRequest) (string, error) {
	var patch PatchProtectionBypassForAutomationRequest
	patch.TeamID = request.TeamID
	patch.ProjectID = request.ProjectID

	switch {
	case request.NewValue && request.NewSecret != "" && request.OldSecret != "":
		patch.Generate = &GenerateProtectionBypassRequest{
			Secret: request.NewSecret,
		}
		patch.Revoke = &RevokeProtectionBypassRequest{
			Regenerate: true,
			Secret:     request.OldSecret,
		}
	case request.NewValue && request.NewSecret != "":
		patch.Generate = &GenerateProtectionBypassRequest{
			Secret: request.NewSecret,
		}
	case request.NewValue:
		// Leaving the patch body empty triggers a generated secret.
	case request.OldSecret != "":
		patch.Revoke = &RevokeProtectionBypassRequest{
			Regenerate: false,
			Secret:     request.OldSecret,
		}
	default:
		return "", nil
	}

	protectionBypass, err := c.PatchProtectionBypassForAutomation(ctx, patch)
	if err != nil {
		return "", err
	}

	if !request.NewValue {
		return "", nil
	}

	if request.NewSecret != "" {
		return request.NewSecret, nil
	}

	for key, bypass := range protectionBypass {
		if bypass.Scope == "automation-bypass" {
			return key, nil
		}
	}

	return "", fmt.Errorf("error adding protection bypass for automation: the response did not contain an automation bypass secret")
}
