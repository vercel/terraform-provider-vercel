package client

import (
	"context"
	"fmt"
)

type AttackChallengeMode struct {
	ProjectID string `json:"projectId"`
	TeamID    string `json:"-"`
	Enabled   bool   `json:"attackModeEnabled"`
}

func (c *Client) GetAttackChallengeMode(ctx context.Context, projectID, teamID string) (a AttackChallengeMode, err error) {
	project, err := c.GetProject(ctx, projectID, teamID)
	if err != nil {
		return a, err
	}
	var enabled bool
	if project.Security != nil {
		enabled = project.Security.AttackModeEnabled
	}
	return AttackChallengeMode{
		ProjectID: projectID,
		TeamID:    teamID,
		Enabled:   enabled,
	}, err
}

func (c *Client) UpdateAttackChallengeMode(ctx context.Context, request AttackChallengeMode) (a AttackChallengeMode, err error) {
	url := fmt.Sprintf("%s/security/attack-mode", c.baseURL)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	var res struct {
		AttackModeEnabled bool `json:"attackModeEnabled"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &res)
	if err != nil {
		return a, err
	}
	return AttackChallengeMode{
		ProjectID: request.ProjectID,
		TeamID:    request.TeamID,
		Enabled:   res.AttackModeEnabled,
	}, err
}
