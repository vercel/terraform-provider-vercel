package client

import (
	"encoding/json"
	"errors"
)

// NotFound detects if an error returned by the Vercel API was the result of an entity not existing.
func NotFound(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404
}

func noContent(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 204
}

func conflictingSharedEnv(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 409 && apiErr.Code == "existing_key_and_target"
}

type EnvConflictError struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Key       string   `json:"key"`
	Target    []string `json:"target"`
	GitBranch *string  `json:"gitBranch"`
}

func conflictingEnvVar(e error) (envConflictError EnvConflictError, ok bool, err error) {
	var apiErr APIError
	conflict := e != nil && errors.As(e, &apiErr) && apiErr.StatusCode == 403 && apiErr.Code == "ENV_ALREADY_EXISTS"
	if !conflict {
		return envConflictError, false, err
	}

	var conflictErr struct {
		Error EnvConflictError `json:"error"`
	}
	_ = json.Unmarshal(apiErr.RawMessage, &conflictErr)
	return conflictErr.Error, true, err
}
