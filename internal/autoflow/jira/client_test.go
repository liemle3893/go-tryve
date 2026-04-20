package jira

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestServer returns a Jira-like httptest server and a Client pointed at
// it. The handler is responsible for whatever routing the test needs.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	u, _ := url.Parse(srv.URL)
	c := &Client{
		creds: &Credentials{Host: u.Host, Email: "me@x", Token: "tok"},
		http:  srv.Client(),
		base:  srv.URL,
	}
	return srv, c
}

func TestMyself(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			http.NotFound(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "me@x" || pass != "tok" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"accountId":"abc-123"}`))
	})
	id, err := c.Myself(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc-123" {
		t.Errorf("want abc-123, got %q", id)
	}
}

func TestListAttachments(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/PROJ-1") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("fields") != "attachment" {
			http.Error(w, "missing fields", 400)
			return
		}
		_, _ = w.Write([]byte(`{
		  "fields": {
		    "attachment": [
		      {"id":"1","filename":"a.png","mimeType":"image/png","size":1234,"content":"https://jira/download/1"},
		      {"id":"2","filename":"b.txt","mimeType":"text/plain","size":10,"content":"https://jira/download/2"}
		    ]
		  }
		}`))
	})
	atts, err := c.ListAttachments(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 2 || atts[0].Filename != "a.png" {
		t.Errorf("unexpected attachments: %+v", atts)
	}
}

func TestListAttachments_InvalidKey(t *testing.T) {
	c := &Client{creds: &Credentials{Host: "x", Email: "e", Token: "t"}, http: http.DefaultClient, base: "https://x"}
	if _, err := c.ListAttachments(context.Background(), "../bad"); err == nil {
		t.Errorf("expected validation error")
	}
}

func TestUpload(t *testing.T) {
	var seen struct {
		contentType string
		xsrfHeader  string
		filename    string
		bodyContent []byte
	}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/attachments") {
			http.NotFound(w, r)
			return
		}
		seen.contentType = r.Header.Get("Content-Type")
		seen.xsrfHeader = r.Header.Get("X-Atlassian-Token")

		mr, err := r.MultipartReader()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if part.FormName() == "file" {
				seen.filename = part.FileName()
				seen.bodyContent, _ = io.ReadAll(part)
			}
		}
		_, _ = w.Write([]byte(`[{"id":"att-1"}]`))
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	_ = os.WriteFile(path, []byte("hello"), 0o644)

	results, err := c.Upload(context.Background(), "PROJ-1", []string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Error != nil || results[0].ID != "att-1" {
		t.Errorf("bad result: %+v", results[0])
	}
	if !strings.HasPrefix(seen.contentType, "multipart/form-data") {
		t.Errorf("content-type: %q", seen.contentType)
	}
	if seen.xsrfHeader != "no-check" {
		t.Errorf("X-Atlassian-Token header missing or wrong: %q", seen.xsrfHeader)
	}
	if seen.filename != "note.md" {
		t.Errorf("filename: got %q", seen.filename)
	}
	if string(seen.bodyContent) != "hello" {
		t.Errorf("body content: got %q", string(seen.bodyContent))
	}
}

func TestDownload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/issue/PROJ-1", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"fields": map[string]any{
				"attachment": []map[string]any{
					{"id": "1", "filename": "hello.txt", "mimeType": "text/plain", "size": 5, "content": "http://" + r.Host + "/download/1"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/download/1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("world"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	u, _ := url.Parse(srv.URL)
	c := &Client{
		creds: &Credentials{Host: u.Host, Email: "me@x", Token: "tok"},
		http:  srv.Client(),
		base:  srv.URL,
	}

	destDir := t.TempDir()
	results, err := c.Download(context.Background(), "PROJ-1", destDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Error != nil {
		t.Fatalf("bad result: %+v", results)
	}
	data, _ := os.ReadFile(filepath.Join(destDir, "hello.txt"))
	if string(data) != "world" {
		t.Errorf("downloaded content: %q", string(data))
	}
}

// Silence import-only warnings if any helper ever goes unused.
var _ = multipart.Writer{}
