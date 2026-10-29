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

// Transient detects errors returned by the Vercel API that are likely to
// succeed if the same request is retried, such as conflicts caused by
// concurrent changes, rate limits, and server errors. Validation and
// permission failures are never transient.
func Transient(err error) bool {
	var apiErr APIError
	if err == nil || !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == 409 || apiErr.StatusCode == 429 || apiErr.StatusCode >= 500
}

// MemberNotFound detects the "member_not_found" error returned when a team
// member cannot be located yet. This is commonly transient immediately after a
// member is invited, while the membership propagates.
func MemberNotFound(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.Code == "member_not_found"
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
	EnvVarKey string   `json:"envVarKey"`
	Target    []string `json:"target"`
	GitBranch *string  `json:"gitBranch"`
}

func conflictingEnvVar(e error) (envConflictError EnvConflictError, ok bool, err error) {
	var apiErr APIError
	conflict := e != nil && errors.As(e, &apiErr) && apiErr.StatusCode == 400 && apiErr.Code == "ENV_CONFLICT"
	if !conflict {
		return envConflictError, false, err
	}

	var conflictErr struct {
		Error EnvConflictError `json:"error"`
	}
	_ = json.Unmarshal(apiErr.RawMessage, &conflictErr)
	return conflictErr.Error, true, err
}
