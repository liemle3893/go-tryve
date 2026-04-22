package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// Issue is the subset of a Jira issue payload the autoflow skills rely on.
// Fields and Rendered are kept as free-form maps so callers/JSON consumers
// can read any field without client-side schema coupling.
type Issue struct {
	Key      string         `json:"key"`
	Fields   map[string]any `json:"fields,omitempty"`
	Rendered map[string]any `json:"renderedFields,omitempty"`
}

// GetIssue fetches /rest/api/3/issue/{key} with optional fields and expand
// selectors. Pass nil/empty slices to omit the corresponding query params.
func (c *Client) GetIssue(ctx context.Context, key string, fields, expand []string) (*Issue, error) {
	if err := state.ValidateTicketKey(key); err != nil {
		return nil, err
	}

	q := url.Values{}
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	if len(expand) > 0 {
		q.Set("expand", strings.Join(expand, ","))
	}
	path := "/rest/api/3/issue/" + key
	if qs := q.Encode(); qs != "" {
		path += "?" + qs
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	body, err := c.do(req, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var out Issue
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse issue %s: %w", key, err)
	}
	return &out, nil
}
