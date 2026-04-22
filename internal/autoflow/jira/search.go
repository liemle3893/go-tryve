package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// SearchPage is the subset of the /rest/api/3/search/jql response autoflow
// uses. Jira returns `isLast=true` plus an absent `nextPageToken` on the
// last page; we preserve both so callers can drive pagination either way.
type SearchPage struct {
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	IsLast        bool    `json:"isLast"`
}

// SearchJQL runs a JQL query against the new /search/jql endpoint (the
// legacy /search endpoint is deprecated). `pageToken` is the opaque token
// returned by a previous page; pass "" for the first page.
func (c *Client) SearchJQL(ctx context.Context, jql string, fields []string, pageToken string) (*SearchPage, error) {
	if strings.TrimSpace(jql) == "" {
		return nil, fmt.Errorf("SearchJQL: jql is required")
	}
	q := url.Values{}
	q.Set("jql", jql)
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	if pageToken != "" {
		q.Set("nextPageToken", pageToken)
	}
	path := "/rest/api/3/search/jql?" + q.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	body, err := c.do(req, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var out SearchPage
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	return &out, nil
}
