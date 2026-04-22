package jira

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetIssue_HappyPath(t *testing.T) {
	var gotPath, gotQuery string
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"key":"PROJ-1","fields":{"summary":"hi"},"renderedFields":{"description":"<p>x</p>"}}`))
	})
	got, err := c.GetIssue(context.Background(), "PROJ-1", []string{"summary", "status"}, []string{"renderedFields"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "PROJ-1" || got.Fields["summary"] != "hi" {
		t.Errorf("bad issue: %+v", got)
	}
	if gotPath != "/rest/api/3/issue/PROJ-1" {
		t.Errorf("path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=summary%2Cstatus") || !strings.Contains(gotQuery, "expand=renderedFields") {
		t.Errorf("query: %s", gotQuery)
	}
	if got.Rendered["description"] != "<p>x</p>" {
		t.Errorf("renderedFields not parsed: %+v", got.Rendered)
	}
}

func TestGetIssue_InvalidKey(t *testing.T) {
	c := &Client{creds: &Credentials{Host: "x", Email: "e", Token: "t"}, http: http.DefaultClient, base: "https://x"}
	if _, err := c.GetIssue(context.Background(), "../bad", nil, nil); err == nil {
		t.Errorf("expected validation error")
	}
}

func TestGetIssue_404(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errorMessages":["not found"]}`, http.StatusNotFound)
	})
	if _, err := c.GetIssue(context.Background(), "PROJ-999", nil, nil); err == nil {
		t.Errorf("expected 404 error")
	} else if !strings.Contains(err.Error(), "404") {
		t.Errorf("want 404 in err, got: %v", err)
	}
}
