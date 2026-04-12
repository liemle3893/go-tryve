package adapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liemle3893/e2e-runner/internal/adapter"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// TestHTTPAdapter_GET verifies that a simple GET request returns status 200.
func TestHTTPAdapter_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	result, err := a.Execute(ctx, "request", map[string]any{
		"method": "GET",
		"url":    "/",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	status, ok := result.Data["status"].(float64)
	if !ok {
		t.Fatalf("expected float64 status, got %T: %v", result.Data["status"], result.Data["status"])
	}
	if status != 200 {
		t.Fatalf("expected status 200, got %v", status)
	}
}

// TestHTTPAdapter_POST_JSON verifies that a POST with a body sends JSON and
// the response body is correctly parsed.
func TestHTTPAdapter_POST_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("server: unmarshal request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]any{"received": payload["name"]}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	result, err := a.Execute(ctx, "request", map[string]any{
		"method": "POST",
		"url":    "/items",
		"body":   map[string]any{"name": "widget"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["status"].(float64) != 201 {
		t.Fatalf("expected status 201, got %v", result.Data["status"])
	}

	body, ok := result.Data["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected map body, got %T: %v", result.Data["body"], result.Data["body"])
	}
	if body["received"] != "widget" {
		t.Fatalf("expected received=widget, got %v", body["received"])
	}
}

// TestHTTPAdapter_Headers verifies that custom headers are forwarded to the server.
func TestHTTPAdapter_Headers(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	_, err := a.Execute(ctx, "request", map[string]any{
		"method": "GET",
		"url":    "/secure",
		"headers": map[string]any{
			"Authorization": "Bearer token123",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotAuth != "Bearer token123" {
		t.Fatalf("expected Authorization header 'Bearer token123', got %q", gotAuth)
	}
}

// TestHTTPAdapter_QueryParams verifies that query parameters are appended to the URL.
func TestHTTPAdapter_QueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	_, err := a.Execute(ctx, "request", map[string]any{
		"method": "GET",
		"url":    "/search",
		"query": map[string]any{
			"q":    "test",
			"page": "1",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotQuery == "" {
		t.Fatal("expected query string, got empty")
	}

	// Verify both params are present; order in the raw query may vary.
	checkParam := func(key, want string) {
		// Re-parse the raw query to avoid import of url package duplicating logic.
		val := ""
		for _, kv := range splitQuery(gotQuery) {
			if kv[0] == key {
				val = kv[1]
				break
			}
		}
		if val != want {
			t.Fatalf("expected query param %s=%s, got %q in %q", key, want, val, gotQuery)
		}
	}
	checkParam("q", "test")
	checkParam("page", "1")
}

// splitQuery is a minimal query string parser for test assertions.
func splitQuery(raw string) [][2]string {
	var pairs [][2]string
	for raw != "" {
		var pair string
		if i := indexByte(raw, '&'); i >= 0 {
			pair, raw = raw[:i], raw[i+1:]
		} else {
			pair, raw = raw, ""
		}
		if pair == "" {
			continue
		}
		var k, v string
		if i := indexByte(pair, '='); i >= 0 {
			k, v = unescape(pair[:i]), unescape(pair[i+1:])
		} else {
			k = unescape(pair)
		}
		pairs = append(pairs, [2]string{k, v})
	}
	return pairs
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func unescape(s string) string {
	// Minimal percent-decode for test use only.
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] == '+' {
			out = append(out, ' ')
			i++
		} else if s[i] == '%' && i+2 < len(s) {
			hi := hexVal(s[i+1])
			lo := hexVal(s[i+2])
			out = append(out, byte(hi<<4|lo))
			i += 3
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// TestHTTPAdapter_NonJSONResponse verifies that a text/plain response is
// returned as a string in the body field.
func TestHTTPAdapter_NonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	result, err := a.Execute(ctx, "request", map[string]any{
		"method": "GET",
		"url":    "/text",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	body, ok := result.Data["body"].(string)
	if !ok {
		t.Fatalf("expected string body, got %T: %v", result.Data["body"], result.Data["body"])
	}
	if body != "hello world" {
		t.Fatalf("expected body 'hello world', got %q", body)
	}
}

// TestHTTPAdapter_Health verifies that the health check succeeds when the
// server responds to a HEAD request.
func TestHTTPAdapter_Health(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD for health check, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	if err := a.Health(ctx); err != nil {
		t.Fatalf("Health: unexpected error: %v", err)
	}
}

// TestHTTPAdapter_InvalidAction verifies that an unsupported action returns
// an ADAPTER_ERROR.
func TestHTTPAdapter_InvalidAction(t *testing.T) {
	a := adapter.NewHTTPAdapter("http://localhost")
	ctx := context.Background()

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	_, err := a.Execute(ctx, "unsupported", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unsupported action, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("expected code ADAPTER_ERROR, got %s", tryveErr.Code)
	}
}
