package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var providerOpenAPI = &openAPIEnums{
	url:    "https://openapi.vercel.sh/",
	client: &http.Client{Timeout: 30 * time.Second},
}

type openAPIEnums struct {
	url    string
	client *http.Client
	mu     sync.Mutex
	values map[string][]string
}

type openAPISchema struct {
	Items *openAPISchema `json:"items"`
	Enum  []*string      `json:"enum"`
}

type openAPIOperation struct {
	RequestBody struct {
		Content map[string]struct {
			Schema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
}

func (s *openAPIEnums) load(ctx context.Context) (map[string][]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values != nil {
		return s.values, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating OpenAPI request: %w", err)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Vercel OpenAPI schema: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching Vercel OpenAPI schema: unexpected status code %d", res.StatusCode)
	}

	var document struct {
		Paths map[string]map[string]openAPIOperation `json:"paths"`
	}
	if err := json.NewDecoder(res.Body).Decode(&document); err != nil {
		return nil, fmt.Errorf("decoding Vercel OpenAPI schema: %w", err)
	}

	frameworkJSON := document.Paths["/v9/projects/{idOrName}"]["patch"].RequestBody.Content["application/json"].Schema.Properties["framework"]
	eventsJSON := document.Paths["/v1/webhooks"]["post"].RequestBody.Content["application/json"].Schema.Properties["events"]
	var framework, events openAPISchema
	if err := json.Unmarshal(frameworkJSON, &framework); err != nil {
		return nil, fmt.Errorf("decoding OpenAPI framework schema: %w", err)
	}
	if err := json.Unmarshal(eventsJSON, &events); err != nil {
		return nil, fmt.Errorf("decoding OpenAPI webhook event schema: %w", err)
	}
	if events.Items == nil {
		return nil, fmt.Errorf("vercel OpenAPI schema is missing webhook event items")
	}

	values := make(map[string][]string)
	for name, schema := range map[string]openAPISchema{"framework": framework, "webhook event": *events.Items} {
		for _, value := range schema.Enum {
			if value != nil {
				values[name] = append(values[name], *value)
			}
		}
		if len(values[name]) == 0 {
			return nil, fmt.Errorf("vercel OpenAPI schema has no %s enum values", name)
		}
	}

	// Cache only successful fetches, so a transient failure does not poison the
	// provider process. Both validators share the same immutable enum lists.
	s.values = values
	return s.values, nil
}
