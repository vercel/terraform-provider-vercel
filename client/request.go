package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

/*
 * version is the tagged version of this repository. It is overriden at build time by ldflags.
 * please see the .goreleaser.yml file for more information.
 */
var version = "dev"

// APIError is an error type that exposes additional information about why an API request failed.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int
	RawMessage []byte
	retryAfter int
}

// Error provides a user friendly error message.
func (e APIError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

type clientRequest struct {
	ctx              context.Context
	method           string
	url              string
	body             string
	errorOnNoContent bool
}

func (cr *clientRequest) toHTTPRequest() (*http.Request, error) {
	r, err := http.NewRequestWithContext(
		cr.ctx,
		cr.method,
		cr.url,
		strings.NewReader(cr.body),
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", fmt.Sprintf("terraform-provider-vercel/%s", version))
	if cr.body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r, nil
}

// doRequest is a helper function for consistently requesting data from vercel.
// This manages:
// - Setting the default Content-Type for requests with a body
// - Setting the User-Agent
// - Authorization via the Bearer token
// - Converting error responses into an inspectable type
// - Unmarshaling responses
// - Parsing a Retry-After header in the case of rate limits being hit
// - In the case of a rate-limit being hit, trying again aftera period of time
func (c *Client) doRequest(req clientRequest, v any) error {
	r, err := req.toHTTPRequest()
	if err != nil {
		return err
	}
	err = c._doRequest(r, v, req.errorOnNoContent)
	for retries := 0; retries < 3; retries++ {
		var apiErr APIError
		if errors.As(err, &apiErr) && // we received an api error
			apiErr.StatusCode == 429 && // and it was a rate limit
			apiErr.retryAfter > 0 && // and there was a retry time
			apiErr.retryAfter < 5*60 { // and the retry time is less than 5 minutes
			tflog.Error(req.ctx, "Rate limit was hit", map[string]any{
				"error":      apiErr,
				"retryAfter": apiErr.retryAfter,
			})
			time.Sleep(time.Duration(apiErr.retryAfter) * time.Second)
			r, err = req.toHTTPRequest()
			if err != nil {
				return err
			}
			err = c._doRequest(r, v, req.errorOnNoContent)
			if err != nil {
				continue
			}
			return nil
		} else {
			break
		}
	}

	return err
}

func (c *Client) _doRequest(req *http.Request, v any, errorOnNoContent bool) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("error doing http request: %w", err)
	}

	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		var errorResponse APIError
		if string(responseBody) == "" {
			errorResponse.StatusCode = resp.StatusCode
			return errorResponse
		}
		err = json.Unmarshal(responseBody, &struct {
			Error *APIError `json:"error"`
		}{
			Error: &errorResponse,
		})
		if errorResponse.Code == "" && errorResponse.Message == "" {
			return fmt.Errorf("error performing API request: %d %s", resp.StatusCode, string(responseBody))
		}
		if err != nil {
			return fmt.Errorf("error unmarshaling response for status code %d: %w: %s", resp.StatusCode, err, string(responseBody))
		}
		errorResponse.StatusCode = resp.StatusCode
		errorResponse.RawMessage = responseBody
		errorResponse.retryAfter = 1000 // set a sensible default for retrying. This is in milliseconds.
		if resp.StatusCode == 429 {
			retryAfterRaw := resp.Header.Get("Retry-After")
			if retryAfterRaw != "" {
				retryAfter, err := strconv.Atoi(retryAfterRaw)
				if err == nil && retryAfter > 0 {
					errorResponse.retryAfter = retryAfter
				}
			}
		}
		return errorResponse
	}

	if v == nil {
		return nil
	}

	if errorOnNoContent && resp.StatusCode == 204 {
		return APIError{
			StatusCode: 204,
			Code:       "no_content",
			Message:    "No content",
		}
	}

	err = json.Unmarshal(responseBody, v)
	if err != nil {
		return fmt.Errorf("error unmarshaling response %s: %w", responseBody, err)
	}

	return nil
}

// doRequestWithResponse is similar to doRequest but returns the raw response body as a string
func (c *Client) doRequestWithResponse(req clientRequest) (string, error) {
	r, err := req.toHTTPRequest()
	if err != nil {
		return "", err
	}

	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	resp, err := c.http().Do(r)
	if err != nil {
		return "", fmt.Errorf("error doing http request: %w", err)
	}

	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		var errorResponse APIError
		if string(responseBody) == "" {
			errorResponse.StatusCode = resp.StatusCode
			return string(responseBody), errorResponse
		}
		err = json.Unmarshal(responseBody, &struct {
			Error *APIError `json:"error"`
		}{
			Error: &errorResponse,
		})
		if errorResponse.Code == "" && errorResponse.Message == "" {
			return string(responseBody), fmt.Errorf("error performing API request: %d %s", resp.StatusCode, string(responseBody))
		}
		if err != nil {
			return string(responseBody), fmt.Errorf("error unmarshaling response for status code %d: %w: %s", resp.StatusCode, err, string(responseBody))
		}
		errorResponse.StatusCode = resp.StatusCode
		errorResponse.RawMessage = responseBody
		errorResponse.retryAfter = 1000 // set a sensible default for retrying. This is in milliseconds.
		if resp.StatusCode == 429 {
			retryAfterRaw := resp.Header.Get("Retry-After")
			if retryAfterRaw != "" {
				retryAfter, err := strconv.Atoi(retryAfterRaw)
				if err == nil && retryAfter > 0 {
					errorResponse.retryAfter = retryAfter
				}
			}
		}
		return string(responseBody), errorResponse
	}

	return string(responseBody), nil
}
