// Package integration contains end-to-end tests that verify the full test runner
// pipeline: config loading → test discovery → parsing → execution with HTTP
// adapter → variable interpolation → capture → assertions → reporting.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/autoflow/internal/adapter"
	"github.com/liemle3893/autoflow/internal/config"
	"github.com/liemle3893/autoflow/internal/executor"
	"github.com/liemle3893/autoflow/internal/loader"
	"github.com/liemle3893/autoflow/internal/reporter"
	"github.com/liemle3893/autoflow/internal/core"
)

// newIntegrationServer starts a local HTTP server with the routes required by
// the integration test suite and returns it along with a teardown function.
//
// Routes:
//   - GET  /api/health         → {"status":"ok"} 200
//   - POST /api/users          → {"id":42,"name":<from body>} 201
//   - GET  /api/users/42       → {"id":42,"name":"test-user"} 200
//   - anything else            → 404
func newIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// GET /api/health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
	})

	// POST /api/users
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		name, _ := body["name"].(string)
		resp, _ := json.Marshal(map[string]any{
			"id":   42,
			"name": name,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(resp)
	})

	// GET /api/users/42
	mux.HandleFunc("/api/users/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"id":42,"name":"test-user"}`)
	})

	// Default → 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// writeConfigFile creates a temporary e2e.config.yaml pointing at serverURL
// and returns its absolute path.
func writeConfigFile(t *testing.T, dir, serverURL string) string {
	t.Helper()

	content := fmt.Sprintf(`version: "1.0"
environments:
  test:
    baseUrl: "%s"
defaults:
  timeout: 10000
  parallel: 1
`, serverURL)

	path := filepath.Join(dir, "e2e.config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}

// writeTestFile creates a temporary YAML test file that exercises the full HTTP
// flow: health check → create user (with capture) → get user by captured ID.
func writeTestFile(t *testing.T, dir string) string {
	t.Helper()

	content := `name: "Integration: User CRUD"
tags: [integration, api]
priority: P0

variables:
  userName: "test-user"

execute:
  - adapter: http
    action: request
    url: /api/health
    method: GET
    assert:
      status: 200
      json:
        - path: "$.status"
          equals: "ok"

  - adapter: http
    action: request
    url: /api/users
    method: POST
    body:
      name: "{{userName}}"
    capture:
      userId: "$.id"
    assert:
      status: 201
      json:
        - path: "$.name"
          equals: "test-user"

  - adapter: http
    action: request
    url: "/api/users/{{captured.userId}}"
    method: GET
    assert:
      status: 200
      json:
        - path: "$.id"
          equals: 42
`

	path := filepath.Join(dir, "user-crud.test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return path
}

// printFailureDetails logs step and assertion details when the integration test
// fails, making it easier to diagnose assertion mismatches without re-running
// the whole test under a debugger.
func printFailureDetails(t *testing.T, result *core.SuiteResult) {
	t.Helper()

	for i := range result.Tests {
		tr := &result.Tests[i]
		if tr.Status != core.StatusFailed {
			continue
		}
		testName := "<unknown>"
		if tr.Test != nil {
			testName = tr.Test.Name
		}
		t.Logf("FAILED test: %q  error: %v", testName, tr.Error)

		for j := range tr.Steps {
			so := &tr.Steps[j]
			stepID := "<no-id>"
			if so.Step != nil {
				stepID = so.Step.ID
			}
			t.Logf("  step[%d] id=%q  phase=%s  status=%s  err=%v",
				j, stepID, so.Phase, so.Status, so.Error)

			for k, ao := range so.Assertions {
				t.Logf("    assertion[%d] path=%q  op=%q  expected=%v  actual=%v  passed=%v  msg=%q",
					k, ao.Path, ao.Operator, ao.Expected, ao.Actual, ao.Passed, ao.Message)
			}
		}
	}
}

// TestIntegration_FullHTTPFlow is the primary integration test. It wires every
// component together — config loading, test discovery, parsing, validation,
// adapter registry, orchestration, and reporting — and asserts that one test
// passes with zero failures.
func TestIntegration_FullHTTPFlow(t *testing.T) {
	// 1. Start test HTTP server.
	srv := newIntegrationServer(t)

	// 2. Create a temporary directory for config and test files.
	tmpDir := t.TempDir()

	// 3. Write config and test files.
	cfgPath := writeConfigFile(t, tmpDir, srv.URL)
	writeTestFile(t, tmpDir)

	// 4. Load configuration.
	cfg, err := config.Load(cfgPath, "test")
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	// 5. Discover test files in the temp directory.
	paths, err := loader.Discover(tmpDir)
	if err != nil {
		t.Fatalf("loader.Discover failed: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no test files discovered in temp directory")
	}

	// 6. Parse all discovered test files.
	tests := make([]*core.TestDefinition, 0, len(paths))
	for _, p := range paths {
		td, parseErr := loader.ParseFile(p)
		if parseErr != nil {
			t.Fatalf("loader.ParseFile(%q) failed: %v", p, parseErr)
		}
		tests = append(tests, td)
	}

	// 7. Validate each test definition; fail fast if there are errors.
	for _, td := range tests {
		if errs := loader.Validate(td); len(errs) > 0 {
			for _, ve := range errs {
				t.Errorf("validation error for %q: %v", td.Name, ve)
			}
			t.FailNow()
		}
	}

	// 8. Build adapter registry with HTTP adapter (base URL from config) and shell adapter.
	reg := adapter.NewRegistry()
	reg.Register("http", adapter.NewHTTPAdapter(cfg.Environment.BaseURL))
	reg.Register("shell", adapter.NewShellAdapter(nil))

	// 9. Create a no-op reporter using reporter.NewMulti with no sinks.
	rep := reporter.NewMulti()

	// 10. Create orchestrator and execute the tests.
	orch := executor.NewOrchestrator(reg, rep, cfg)
	result := orch.Run(context.Background(), tests)

	// 11. Print failure details before asserting so they appear in test output.
	if result.Failed > 0 {
		printFailureDetails(t, result)
	}

	// 12. Assert the expected pass/fail counts.
	if result.Passed != 1 {
		t.Errorf("expected passed=1, got %d", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
	}
}
