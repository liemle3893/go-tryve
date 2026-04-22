package runner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/autoflow/pkg/runner"
)

// newTestServer starts a minimal httptest.Server with a single route:
//
//	GET /ping → {"ok":true} 200
//
// The server is automatically closed when t finishes.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// writeConfig writes a minimal e2e.config.yaml pointing at serverURL to dir and
// returns the absolute file path.
func writeConfig(t *testing.T, dir, serverURL string) string {
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
		t.Fatalf("writeConfig: %v", err)
	}
	return path
}

// writePingTest writes a simple test YAML that sends GET /ping and asserts status 200.
func writePingTest(t *testing.T, dir string) string {
	t.Helper()

	content := `name: "Ping test"
tags: [smoke]
priority: P1

execute:
  - adapter: http
    action: request
    url: /ping
    method: GET
    assert:
      status: 200
      json:
        - path: "$.ok"
          equals: true
`
	path := filepath.Join(dir, "ping.test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writePingTest: %v", err)
	}
	return path
}

// writeInvalidTest writes a YAML file that is missing the required execute phase.
func writeInvalidTest(t *testing.T, dir string) string {
	t.Helper()

	content := `name: "Missing execute"
tags: [bad]
priority: P2
`
	path := filepath.Join(dir, "invalid.test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeInvalidTest: %v", err)
	}
	return path
}

// writeBrokenYAML writes a file with unparseable YAML content.
func writeBrokenYAML(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "broken.test.yaml")
	if err := os.WriteFile(path, []byte(":\t bad yaml {{{{"), 0o644); err != nil {
		t.Fatalf("writeBrokenYAML: %v", err)
	}
	return path
}

// TestRunTests_Pass runs a single HTTP test against a live httptest.Server and
// verifies that the SuiteResult reports exactly one passing test.
func TestRunTests_Pass(t *testing.T) {
	srv := newTestServer(t)
	tmpDir := t.TempDir()

	cfgPath := writeConfig(t, tmpDir, srv.URL)
	writePingTest(t, tmpDir)

	opts := runner.Options{
		ConfigPath:  cfgPath,
		Environment: "test",
		TestDir:     tmpDir,
		Retries:     -1, // use config default
	}

	result, err := runner.RunTests(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunTests returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RunTests returned nil SuiteResult")
	}
	if result.Passed != 1 {
		t.Errorf("expected passed=1, got %d (failed=%d, skipped=%d)",
			result.Passed, result.Failed, result.Skipped)
		printSuiteDetails(t, result)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
		printSuiteDetails(t, result)
	}
}

// TestRunTests_DryRun verifies that DryRun=true returns a result with the correct
// test count but does not execute tests (Failed==0, Passed==0).
func TestRunTests_DryRun(t *testing.T) {
	srv := newTestServer(t)
	tmpDir := t.TempDir()

	cfgPath := writeConfig(t, tmpDir, srv.URL)
	writePingTest(t, tmpDir)

	opts := runner.Options{
		ConfigPath:  cfgPath,
		Environment: "test",
		TestDir:     tmpDir,
		DryRun:      true,
		Retries:     -1,
	}

	result, err := runner.RunTests(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunTests (DryRun) returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RunTests (DryRun) returned nil SuiteResult")
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if result.Passed != 0 || result.Failed != 0 {
		t.Errorf("DryRun should not execute tests; passed=%d failed=%d",
			result.Passed, result.Failed)
	}
}

// TestValidateTests_ValidFile verifies that a well-formed test file produces a
// ValidationResult with Valid==true and no errors.
func TestValidateTests_ValidFile(t *testing.T) {
	srv := newTestServer(t)
	tmpDir := t.TempDir()

	cfgPath := writeConfig(t, tmpDir, srv.URL)
	_ = cfgPath // not needed for validate, but kept for completeness

	writePingTest(t, tmpDir)

	opts := runner.Options{
		TestDir: tmpDir,
	}

	results, err := runner.ValidateTests(opts)
	if err != nil {
		t.Fatalf("ValidateTests returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid=true, got false; errors: %v", results[0].Errors)
	}
	if len(results[0].Errors) != 0 {
		t.Errorf("expected no errors, got: %v", results[0].Errors)
	}
}

// TestValidateTests_InvalidFile verifies that a file with a missing execute phase
// produces a ValidationResult with Valid==false and at least one error.
func TestValidateTests_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	writeInvalidTest(t, tmpDir)

	opts := runner.Options{
		TestDir: tmpDir,
	}

	results, err := runner.ValidateTests(opts)
	if err != nil {
		t.Fatalf("ValidateTests returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("expected valid=false for a file with no execute steps")
	}
	if len(results[0].Errors) == 0 {
		t.Error("expected at least one validation error")
	}
}

// TestValidateTests_BrokenYAML verifies that a file with invalid YAML produces a
// ValidationResult with Valid==false and a parse error message.
func TestValidateTests_BrokenYAML(t *testing.T) {
	tmpDir := t.TempDir()
	writeBrokenYAML(t, tmpDir)

	opts := runner.Options{
		TestDir: tmpDir,
	}

	results, err := runner.ValidateTests(opts)
	if err != nil {
		t.Fatalf("ValidateTests returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("expected valid=false for unparseable YAML")
	}
	if len(results[0].Errors) == 0 {
		t.Error("expected a parse error message")
	}
}

// TestValidateTests_MixedFiles verifies that a directory containing both valid and
// invalid test files produces one ValidationResult per file, each correctly
// categorised.
func TestValidateTests_MixedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writePingTest(t, tmpDir)
	writeInvalidTest(t, tmpDir)

	opts := runner.Options{
		TestDir: tmpDir,
	}

	results, err := runner.ValidateTests(opts)
	if err != nil {
		t.Fatalf("ValidateTests returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	validCount := 0
	invalidCount := 0
	for _, r := range results {
		if r.Valid {
			validCount++
		} else {
			invalidCount++
		}
	}

	if validCount != 1 {
		t.Errorf("expected 1 valid result, got %d", validCount)
	}
	if invalidCount != 1 {
		t.Errorf("expected 1 invalid result, got %d", invalidCount)
	}
}

// TestListTests_All verifies that ListTests returns every valid test when no
// filter options are set.
func TestListTests_All(t *testing.T) {
	tmpDir := t.TempDir()
	writePingTest(t, tmpDir)

	opts := runner.Options{
		TestDir: tmpDir,
	}

	tests, err := runner.ListTests(opts)
	if err != nil {
		t.Fatalf("ListTests returned unexpected error: %v", err)
	}
	if len(tests) != 1 {
		t.Errorf("expected 1 test, got %d", len(tests))
	}
	if tests[0].Name != "Ping test" {
		t.Errorf("expected name %q, got %q", "Ping test", tests[0].Name)
	}
}

// TestListTests_FilterByTag verifies that ListTests respects the Tags filter and
// returns only tests whose tag list intersects the requested tags.
func TestListTests_FilterByTag(t *testing.T) {
	tmpDir := t.TempDir()
	writePingTest(t, tmpDir) // tagged [smoke]

	// Write a second test with a different tag.
	content := `name: "Regression test"
tags: [regression]
priority: P2

execute:
  - adapter: shell
    action: exec
    command: "echo hello"
`
	path := filepath.Join(tmpDir, "regression.test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write regression test: %v", err)
	}

	smokeOpts := runner.Options{
		TestDir: tmpDir,
		Tags:    []string{"smoke"},
	}

	tests, err := runner.ListTests(smokeOpts)
	if err != nil {
		t.Fatalf("ListTests returned unexpected error: %v", err)
	}
	if len(tests) != 1 {
		t.Errorf("expected 1 smoke-tagged test, got %d", len(tests))
	}
	if len(tests) > 0 && tests[0].Name != "Ping test" {
		t.Errorf("expected %q, got %q", "Ping test", tests[0].Name)
	}
}

// TestListTests_FilterByGrep verifies that ListTests filters by name substring.
func TestListTests_FilterByGrep(t *testing.T) {
	tmpDir := t.TempDir()
	writePingTest(t, tmpDir) // "Ping test"

	// Write a second test with a distinct name.
	content := `name: "Auth test"
tags: [auth]
priority: P0

execute:
  - adapter: shell
    action: exec
    command: "echo auth"
`
	path := filepath.Join(tmpDir, "auth.test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write auth test: %v", err)
	}

	opts := runner.Options{
		TestDir: tmpDir,
		Grep:    "Ping",
	}

	tests, err := runner.ListTests(opts)
	if err != nil {
		t.Fatalf("ListTests returned unexpected error: %v", err)
	}
	if len(tests) != 1 {
		t.Errorf("expected 1 test matching grep=Ping, got %d", len(tests))
	}
}

// TestListTests_SkipsInvalidFiles verifies that files that fail parse or
// validation are silently excluded from the results.
func TestListTests_SkipsInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writePingTest(t, tmpDir)    // valid
	writeInvalidTest(t, tmpDir) // invalid — missing execute steps

	opts := runner.Options{
		TestDir: tmpDir,
	}

	tests, err := runner.ListTests(opts)
	if err != nil {
		t.Fatalf("ListTests returned unexpected error: %v", err)
	}
	// Only the valid ping test should appear.
	if len(tests) != 1 {
		t.Errorf("expected 1 test (invalid file silently skipped), got %d", len(tests))
	}
}

// printSuiteDetails marshals the suite result to JSON and logs it to aid
// diagnosis when a RunTests assertion fails.
func printSuiteDetails(t *testing.T, result any) {
	t.Helper()
	b, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("suite result:\n%s", b)
}
