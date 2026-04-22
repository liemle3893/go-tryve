package jira

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestSearchJQL_FirstPage(t *testing.T) {
	var gotQuery string
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			http.NotFound(w, r)
			return
		}
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"issues":[{"key":"PROJ-1"},{"key":"PROJ-2"}],"nextPageToken":"tok-2","isLast":false}`))
	})
	page, err := c.SearchJQL(context.Background(), "project = PROJ", []string{"summary"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Issues) != 2 || page.NextPageToken != "tok-2" || page.IsLast {
		t.Errorf("bad page: %+v", page)
	}
	if !strings.Contains(gotQuery, "jql=project") || !strings.Contains(gotQuery, "fields=summary") {
		t.Errorf("query: %s", gotQuery)
	}
	if strings.Contains(gotQuery, "nextPageToken=") {
		t.Errorf("first page should not carry token: %s", gotQuery)
	}
}

func TestSearchJQL_Pagination(t *testing.T) {
	var gotQuery string
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"issues":[{"key":"PROJ-3"}],"isLast":true}`))
	})
	page, err := c.SearchJQL(context.Background(), "project = PROJ", nil, "tok-2")
	if err != nil {
		t.Fatal(err)
	}
	if !page.IsLast || page.NextPageToken != "" {
		t.Errorf("last page expected, got: %+v", page)
	}
	if !strings.Contains(gotQuery, "nextPageToken=tok-2") {
		t.Errorf("page token missing: %s", gotQuery)
	}
}

func TestSearchJQL_EmptyJQL(t *testing.T) {
	c := &Client{creds: &Credentials{Host: "x", Email: "e", Token: "t"}, http: http.DefaultClient, base: "https://x"}
	if _, err := c.SearchJQL(context.Background(), "   ", nil, ""); err == nil {
		t.Errorf("expected error for empty jql")
	}
}

func TestSearchJQL_5xx(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	if _, err := c.SearchJQL(context.Background(), "project = X", nil, ""); err == nil {
		t.Errorf("expected 5xx error")
	}
}
