package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// APIError is an error type that exposes additional information about why an API request failed.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int
	RawMessage []byte
}

// Error provides a user friendly error message.
func (e APIError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

// doRequest is a helper function for consistently requesting data from vercel.
// This manages:
// - Setting the default Content-Type for requests with a body
// - Authorization via the Bearer token
// - Converting error responses into an inspectable type
// - Unmarshaling responses
func (c *Client) doRequest(req *http.Request, v interface{}) error {
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	resp, err := c.http().Do(req)
	tflog.Info(context.TODO(), "response", map[string]interface{}{
		"status": resp.StatusCode,
		"body": resp,
	})
	if err != nil {
		return fmt.Errorf("error doing http request: %w", err)
	}

	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	tflog.Info(context.TODO(), "response 1", map[string]interface{}{
		"status": resp.StatusCode,
		"body": string(responseBody),
	})
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	tflog.Info(context.TODO(), "response 2", map[string]interface{}{
		"status": resp.StatusCode,
		"body": string(responseBody),
	})

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
		if err != nil {
			return fmt.Errorf("error unmarshaling response for status code %d: %w", resp.StatusCode, err)
		}
		errorResponse.StatusCode = resp.StatusCode
		errorResponse.RawMessage = responseBody
		return errorResponse
	}

	if v == nil {
		return nil
	}

	err = json.Unmarshal(responseBody, v)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	return nil
}
