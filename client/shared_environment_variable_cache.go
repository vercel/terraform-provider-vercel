package client

import (
	"context"
	"slices"
	"time"
)

// Bound staleness when another client changes a variable during a long refresh.
const sharedEnvCacheTTL = 30 * time.Second

type sharedEnvKey struct {
	teamID string
	envID  string
}

type sharedEnvEntry struct {
	gate    chan struct{}
	value   SharedEnvironmentVariableResponse
	expires time.Time
}

// Serialize reads and writes for each variable so concurrent cache misses share
// a successful fetch and an older read cannot repopulate the cache after a write.
func (c *Client) lockSharedEnv(ctx context.Context, teamID, envID string) (*sharedEnvEntry, error) {
	key := sharedEnvKey{teamID: c.TeamID(teamID), envID: envID}
	c.sharedEnvMu.Lock()
	if c.sharedEnvs == nil {
		c.sharedEnvs = make(map[sharedEnvKey]*sharedEnvEntry)
	}
	entry := c.sharedEnvs[key]
	if entry == nil {
		entry = &sharedEnvEntry{gate: make(chan struct{}, 1)}
		c.sharedEnvs[key] = entry
	}
	c.sharedEnvMu.Unlock()

	select {
	case entry.gate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			entry.unlock()
			return nil, err
		}
		return entry, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (e *sharedEnvEntry) unlock() {
	<-e.gate
}

func (e *sharedEnvEntry) unlockAfterWrite() {
	// Even a failed request can have reached the server before the connection
	// failed, so discard cached state regardless of the response.
	e.value = SharedEnvironmentVariableResponse{}
	e.expires = time.Time{}
	e.unlock()
}

func (e SharedEnvironmentVariableResponse) clone() SharedEnvironmentVariableResponse {
	e.ProjectIDs = slices.Clone(e.ProjectIDs)
	e.Target = slices.Clone(e.Target)
	return e
}
