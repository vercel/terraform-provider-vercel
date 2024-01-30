package client

import "errors"

// NotFound detects if an error returned by the Vercel API was the result of an entity not existing.
func NotFound(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 404
}

// NotFound detects if an error returned by the Vercel API was the result of a sensitive env var not being able to be decrypted
func NotDecryptable(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.Code == "not_decryptable"
}
