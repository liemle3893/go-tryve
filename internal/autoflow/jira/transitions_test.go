package jira

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetTransitions(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"transitions":[
		  {"id":"11","name":"Start Dev","to":{"name":"In Development"}},
		  {"id":"21","name":"Dev Done","to":{"name":"In Code Review"}}
		]}`))
	})
	ts, err := c.GetTransitions(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 || ts[0].ID != "11" || ts[1].To.Name != "In Code Review" {
		t.Errorf("unexpected transitions: %+v", ts)
	}
}

func TestGetTransitions_InvalidKey(t *testing.T) {
	c := &Client{creds: &Credentials{Host: "x", Email: "e", Token: "t"}, http: http.DefaultClient, base: "https://x"}
	if _, err := c.GetTransitions(context.Background(), "bad"); err == nil {
		t.Errorf("expected validation error")
	}
}

func TestDoTransition(t *testing.T) {
	var seen struct {
		method string
		ct     string
		body   map[string]any
	}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		seen.method = r.Method
		seen.ct = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &seen.body)
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DoTransition(context.Background(), "PROJ-1", "11"); err != nil {
		t.Fatal(err)
	}
	if seen.method != http.MethodPost || !strings.HasPrefix(seen.ct, "application/json") {
		t.Errorf("unexpected request: %+v", seen)
	}
	tr, _ := seen.body["transition"].(map[string]any)
	if tr == nil || tr["id"] != "11" {
		t.Errorf("unexpected body: %+v", seen.body)
	}
}

func TestDoTransition_EmptyID(t *testing.T) {
	c := &Client{creds: &Credentials{Host: "x", Email: "e", Token: "t"}, http: http.DefaultClient, base: "https://x"}
	if err := c.DoTransition(context.Background(), "PROJ-1", "  "); err == nil {
		t.Errorf("expected error for empty id")
	}
}

func TestFindTransitionByName(t *testing.T) {
	ts := []Transition{
		{ID: "11", Name: "Start Dev"},
		{ID: "21", Name: "Dev Done"},
	}
	got, err := FindTransitionByName(ts, "start dev")
	if err != nil || got.ID != "11" {
		t.Fatalf("case-insensitive match failed: %v, got %+v", err, got)
	}
	got, err = FindTransitionByName(ts, "DEV DONE")
	if err != nil || got.ID != "21" {
		t.Fatalf("uppercase lookup failed: %v, got %+v", err, got)
	}
}

func TestFindTransitionByName_NoMatch(t *testing.T) {
	ts := []Transition{{ID: "11", Name: "Start Dev"}}
	_, err := FindTransitionByName(ts, "Deploy")
	if err == nil {
		t.Fatal("expected no-match error")
	}
	// Error must surface available names so the CLI can display them.
	if !strings.Contains(err.Error(), "Start Dev") {
		t.Errorf("error must list available names, got: %v", err)
	}
}

func TestFindTransitionByName_Empty(t *testing.T) {
	if _, err := FindTransitionByName(nil, "  "); err == nil {
		t.Errorf("expected error for empty name")
	}
}
