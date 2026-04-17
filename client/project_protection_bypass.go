package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// protectionBypassLocks serialises protection-bypass mutations per project.
// Bypasses with a generated secret are identified by diffing the project's
// bypass map before and after the generate call, which races under
// Terraform's default parallelism when sibling resources create bypasses
// on the same project. Holding a per-project lock across any mutation
// keeps the diff unambiguous and guards the pre-delete promotion path.
var protectionBypassLocks sync.Map // projectID -> *sync.Mutex

func protectionBypassLock(projectID string) *sync.Mutex {
	v, _ := protectionBypassLocks.LoadOrStore(projectID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

type CreateProtectionBypassRequest struct {
	TeamID    string
	ProjectID string
	Secret    string
	Note      string
}

type generateBypassBody struct {
	Secret string `json:"secret,omitempty"`
	Note   string `json:"note,omitempty"`
}

type updateBypassBody struct {
	Secret   string  `json:"secret"`
	IsEnvVar *bool   `json:"isEnvVar,omitempty"`
	Note     *string `json:"note,omitempty"`
}

type revokeBypassBody struct {
	Secret     string `json:"secret"`
	Regenerate bool   `json:"regenerate"`
}

type protectionBypassResponse struct {
	ProtectionBypass map[string]ProtectionBypass `json:"protectionBypass"`
}

func (c *Client) protectionBypassURL(projectID, teamID string) string {
	url := fmt.Sprintf("%s/v10/projects/%s/protection-bypass", c.baseURL, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	return url
}

// CreateProtectionBypass generates a new automation bypass on a project. If secret is
// empty, the API generates one and we identify it by diffing the project's bypass map
// before and after the call. The returned secret is the key the bypass is stored under
// — the caller should persist it because subsequent update/delete calls are keyed by
// it. isEnvVar is not accepted here because the API sets it automatically (true for the
// first bypass on a project, false thereafter); promote a non-default bypass with a
// follow-up UpdateProtectionBypass call.
func (c *Client) CreateProtectionBypass(ctx context.Context, req CreateProtectionBypassRequest) (secret string, bypass ProtectionBypass, err error) {
	lock := protectionBypassLock(req.ProjectID)
	lock.Lock()
	defer lock.Unlock()

	existing := map[string]struct{}{}
	if req.Secret == "" {
		project, err := c.GetProject(ctx, req.ProjectID, req.TeamID)
		if err != nil {
			return "", ProtectionBypass{}, fmt.Errorf("unable to read project before creating protection bypass: %w", err)
		}
		for k := range project.ProtectionBypass {
			existing[k] = struct{}{}
		}
	}

	payload := string(mustMarshal(struct {
		Generate generateBypassBody `json:"generate"`
	}{
		Generate: generateBypassBody{
			Secret: req.Secret,
			Note:   req.Note,
		},
	}))

	tflog.Info(ctx, "creating protection bypass", map[string]any{
		"url":     c.protectionBypassURL(req.ProjectID, req.TeamID),
		"payload": payload,
	})

	var response protectionBypassResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    c.protectionBypassURL(req.ProjectID, req.TeamID),
		body:   payload,
	}, &response)
	if err != nil {
		return "", ProtectionBypass{}, fmt.Errorf("unable to create protection bypass: %w", err)
	}

	if req.Secret != "" {
		bypass, ok := response.ProtectionBypass[req.Secret]
		if !ok {
			return "", ProtectionBypass{}, fmt.Errorf("protection bypass was not present in API response")
		}
		return req.Secret, bypass, nil
	}

	for k, v := range response.ProtectionBypass {
		if _, was := existing[k]; was {
			continue
		}
		if v.Scope != "automation-bypass" {
			continue
		}
		return k, v, nil
	}
	return "", ProtectionBypass{}, fmt.Errorf("newly generated protection bypass was not present in API response")
}

type UpdateProtectionBypassRequest struct {
	TeamID    string
	ProjectID string
	Secret    string
	IsEnvVar  *bool
	Note      *string
}

// UpdateProtectionBypass updates the note and/or isEnvVar for an existing bypass. When
// promoting a bypass to isEnvVar=true, the API atomically demotes the previous default.
func (c *Client) UpdateProtectionBypass(ctx context.Context, req UpdateProtectionBypassRequest) (ProtectionBypass, error) {
	lock := protectionBypassLock(req.ProjectID)
	lock.Lock()
	defer lock.Unlock()

	payload := string(mustMarshal(struct {
		Update updateBypassBody `json:"update"`
	}{
		Update: updateBypassBody{
			Secret:   req.Secret,
			IsEnvVar: req.IsEnvVar,
			Note:     req.Note,
		},
	}))

	tflog.Info(ctx, "updating protection bypass", map[string]any{
		"url":     c.protectionBypassURL(req.ProjectID, req.TeamID),
		"payload": payload,
	})

	var response protectionBypassResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    c.protectionBypassURL(req.ProjectID, req.TeamID),
		body:   payload,
	}, &response)
	if err != nil {
		return ProtectionBypass{}, fmt.Errorf("unable to update protection bypass: %w", err)
	}

	bypass, ok := response.ProtectionBypass[req.Secret]
	if !ok {
		return ProtectionBypass{}, APIError{
			StatusCode: 404,
			Message:    "Protection bypass not found",
			Code:       "not_found",
		}
	}
	return bypass, nil
}

type DeleteProtectionBypassRequest struct {
	TeamID    string
	ProjectID string
	Secret    string
	// PromoteReplacementIfDefault asks the client to atomically promote another
	// automation-bypass on the project to the env-var default before revoking
	// this one. The API invariant requires exactly one env-var default when any
	// bypass exists. Safe to set true for non-default bypasses too — it's a no-op
	// when a sibling already holds the slot.
	PromoteReplacementIfDefault bool
}

// DeleteProtectionBypass revokes a bypass without regenerating a replacement.
// When PromoteReplacementIfDefault is true and the target bypass is the current
// env-var default, another sibling bypass is atomically promoted under the same
// lock so parallel deletes of multiple bypasses on the same project can't race
// each other into a "not found" state.
func (c *Client) DeleteProtectionBypass(ctx context.Context, req DeleteProtectionBypassRequest) error {
	lock := protectionBypassLock(req.ProjectID)
	lock.Lock()
	defer lock.Unlock()

	if req.PromoteReplacementIfDefault {
		project, err := c.GetProject(ctx, req.ProjectID, req.TeamID)
		if err != nil && !NotFound(err) {
			return fmt.Errorf("unable to read project to locate replacement bypass: %w", err)
		}
		if err == nil {
			current, ok := project.ProtectionBypass[req.Secret]
			isDefault := ok && current.IsEnvVar != nil && *current.IsEnvVar
			if isDefault {
				for secret, b := range project.ProtectionBypass {
					if secret == req.Secret || b.Scope != "automation-bypass" {
						continue
					}
					isEnvVar := true
					payload := string(mustMarshal(struct {
						Update updateBypassBody `json:"update"`
					}{
						Update: updateBypassBody{Secret: secret, IsEnvVar: &isEnvVar},
					}))
					if err := c.doRequest(clientRequest{
						ctx:    ctx,
						method: "PATCH",
						url:    c.protectionBypassURL(req.ProjectID, req.TeamID),
						body:   payload,
					}, nil); err != nil {
						return fmt.Errorf("unable to promote replacement bypass: %w", err)
					}
					break
				}
			}
		}
	}

	payload := string(mustMarshal(struct {
		Revoke revokeBypassBody `json:"revoke"`
	}{
		Revoke: revokeBypassBody{
			Secret:     req.Secret,
			Regenerate: false,
		},
	}))

	tflog.Info(ctx, "deleting protection bypass", map[string]any{
		"url":     c.protectionBypassURL(req.ProjectID, req.TeamID),
		"payload": payload,
	})

	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    c.protectionBypassURL(req.ProjectID, req.TeamID),
		body:   payload,
	}, nil)
	if err != nil {
		return fmt.Errorf("unable to delete protection bypass: %w", err)
	}
	return nil
}

// GetProtectionBypass fetches the bypass with the given secret from the project. Returns
// a 404 APIError if the bypass does not exist on the project.
func (c *Client) GetProtectionBypass(ctx context.Context, projectID, teamID, secret string) (ProtectionBypass, error) {
	project, err := c.GetProject(ctx, projectID, teamID)
	if err != nil {
		return ProtectionBypass{}, err
	}
	bypass, ok := project.ProtectionBypass[secret]
	if !ok {
		return ProtectionBypass{}, APIError{
			StatusCode: 404,
			Message:    "Protection bypass not found",
			Code:       "not_found",
		}
	}
	return bypass, nil
}
