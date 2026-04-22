package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// TransitionTo names the target status of a transition.
type TransitionTo struct {
	Name string `json:"name"`
}

// Transition is one option returned by GET /issue/{key}/transitions.
type Transition struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	To   TransitionTo `json:"to"`
}

// GetTransitions lists the transitions available on the issue for the
// current user. Transition IDs are instance-specific, so callers should
// resolve by name via FindTransitionByName.
func (c *Client) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	if err := state.ValidateTicketKey(key); err != nil {
		return nil, err
	}
	path := "/rest/api/3/issue/" + key + "/transitions"
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	body, err := c.do(req, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var out struct {
		Transitions []Transition `json:"transitions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse transitions for %s: %w", key, err)
	}
	return out.Transitions, nil
}

// DoTransition posts a transition by id to /issue/{key}/transitions. Jira
// returns 204 No Content on success.
func (c *Client) DoTransition(ctx context.Context, key, transitionID string) error {
	if err := state.ValidateTicketKey(key); err != nil {
		return err
	}
	if strings.TrimSpace(transitionID) == "" {
		return fmt.Errorf("DoTransition: transitionID is required")
	}
	payload, err := json.Marshal(map[string]any{
		"transition": map[string]string{"id": transitionID},
	})
	if err != nil {
		return err
	}
	path := "/rest/api/3/issue/" + key + "/transitions"
	req, err := c.newRequest(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if _, err := c.do(req, http.StatusNoContent); err != nil {
		return err
	}
	return nil
}

// FindTransitionByName returns the first transition whose Name matches
// `name` case-insensitively. On no match, the error message lists every
// available name so callers can surface it to the user.
func FindTransitionByName(ts []Transition, name string) (*Transition, error) {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return nil, fmt.Errorf("FindTransitionByName: name is required")
	}
	for i := range ts {
		if strings.ToLower(ts[i].Name) == want {
			return &ts[i], nil
		}
	}
	names := make([]string, 0, len(ts))
	for _, t := range ts {
		names = append(names, t.Name)
	}
	return nil, fmt.Errorf("no transition named %q (available: %s)", name, strings.Join(names, ", "))
}
