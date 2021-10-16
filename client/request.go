package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c *Client) doRequest(req *http.Request, v interface{}) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("error doing http request: %w", err)
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 status code returned: %d - %s", resp.StatusCode, responseBody)
	}

	err = json.Unmarshal(responseBody, v)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	return nil
}
