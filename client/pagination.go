package client

import (
	"fmt"
	"net/url"
	"strconv"
)

const defaultPaginationLimit = 100

// PageInfo is the pagination metadata returned by Vercel list endpoints.
type PageInfo struct {
	Count int    `json:"count"`
	Next  *int64 `json:"next"`
	Prev  *int64 `json:"prev"`
}

func paginationLimit(limit int) int {
	if limit > 0 {
		return limit
	}
	return defaultPaginationLimit
}

func paginationQuery(values url.Values, limit int, until, since *int64) url.Values {
	query := url.Values{}
	for key, vals := range values {
		query[key] = append([]string(nil), vals...)
	}

	query.Set("limit", strconv.Itoa(paginationLimit(limit)))
	if until != nil {
		query.Set("until", strconv.FormatInt(*until, 10))
	}
	if since != nil {
		query.Set("since", strconv.FormatInt(*since, 10))
	}
	return query
}

func urlWithQuery(baseURL string, query url.Values) string {
	if encoded := query.Encode(); encoded != "" {
		return fmt.Sprintf("%s?%s", baseURL, encoded)
	}
	return baseURL
}

func collectPages[T any](fetch func(until *int64) ([]T, PageInfo, error)) ([]T, error) {
	var all []T
	var until *int64

	for {
		items, pagination, err := fetch(until)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		if pagination.Next == nil {
			return all, nil
		}
		if until != nil && *pagination.Next == *until {
			return nil, fmt.Errorf("pagination cursor did not advance")
		}
		until = pagination.Next
	}
}
