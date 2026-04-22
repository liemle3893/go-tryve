package reporter_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liemle3893/autoflow/internal/reporter"
	"github.com/liemle3893/autoflow/internal/core"
)

// outputPath returns a temp file path for a JSON report within t's temp dir.
func outputPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "report.json")
}

// readReport reads and unmarshals the JSON report file written by the reporter.
func readReport(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readReport: os.ReadFile(%q): %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("readReport: json.Unmarshal: %v", err)
	}
	return out
}

// TestJSONReporter_SummaryCounts verifies that the summary block contains the
// correct total, passed, failed, and skipped counts after OnSuiteComplete.
func TestJSONReporter_SummaryCounts(t *testing.T) {
	path := outputPath(t)
	r := reporter.NewJSON(path)

	suiteResult := &core.SuiteResult{
		Total:    3,
		Passed:   2,
		Failed:   1,
		Skipped:  0,
		Duration: 1500 * time.Millisecond,
	}

	if err := r.OnSuiteComplete(context.Background(), &core.SuiteResult{}, suiteResult); err != nil {
		t.Fatalf("OnSuiteComplete: %v", err)
	}
	if err := r.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	doc := readReport(t, path)

	summary, ok := doc["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'summary' to be a JSON object, got %T", doc["summary"])
	}

	wantFields := map[string]float64{
		"total":   3,
		"passed":  2,
		"failed":  1,
		"skipped": 0,
	}
	for field, want := range wantFields {
		got, ok := summary[field].(float64)
		if !ok {
			t.Errorf("summary.%s: expected float64, got %T", field, summary[field])
			continue
		}
		if got != want {
			t.Errorf("summary.%s: want %.0f, got %.0f", field, want, got)
		}
	}

	if _, ok := summary["duration"]; !ok {
		t.Error("summary.duration: field missing")
	}
}

// TestJSONReporter_TestNames verifies that each accumulated test result appears
// in the "tests" array with the correct name.
func TestJSONReporter_TestNames(t *testing.T) {
	path := outputPath(t)
	r := reporter.NewJSON(path)

	tests := []*core.TestDefinition{
		{Name: "login-flow", Tags: []string{"smoke"}, Priority: core.PriorityP0},
		{Name: "checkout-flow", Tags: []string{"regression"}, Priority: core.PriorityP1},
	}

	for _, td := range tests {
		result := &core.TestResult{
			Test:     td,
			Status:   core.StatusPassed,
			Duration: 250 * time.Millisecond,
		}
		if err := r.OnTestComplete(context.Background(), td, result); err != nil {
			t.Fatalf("OnTestComplete(%s): %v", td.Name, err)
		}
	}

	if err := r.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	doc := readReport(t, path)

	rawTests, ok := doc["tests"].([]any)
	if !ok {
		t.Fatalf("expected 'tests' to be a JSON array, got %T", doc["tests"])
	}
	if len(rawTests) != len(tests) {
		t.Fatalf("expected %d tests, got %d", len(tests), len(rawTests))
	}

	nameSet := map[string]bool{}
	for _, rt := range rawTests {
		obj, ok := rt.(map[string]any)
		if !ok {
			t.Fatalf("test entry: expected object, got %T", rt)
		}
		name, _ := obj["name"].(string)
		nameSet[name] = true
	}

	for _, td := range tests {
		if !nameSet[td.Name] {
			t.Errorf("test %q not found in output", td.Name)
		}
	}
}

// TestJSONReporter_StepDetails verifies that step fields (id, adapter, action,
// status, assertions) are present in the serialised output.
func TestJSONReporter_StepDetails(t *testing.T) {
	path := outputPath(t)
	r := reporter.NewJSON(path)

	step := &core.StepDefinition{
		ID:      "execute-0",
		Adapter: "http",
		Action:  "request",
	}

	td := &core.TestDefinition{
		Name:     "api-health-check",
		Tags:     []string{"smoke"},
		Priority: core.PriorityP0,
	}

	outcome := core.StepOutcome{
		Step:     step,
		Phase:    core.PhaseExecute,
		Status:   core.StatusPassed,
		Duration: 120 * time.Millisecond,
		Assertions: []core.AssertionOutcome{
			{
				Path:     "$.status",
				Operator: "equals",
				Expected: float64(200),
				Actual:   float64(200),
				Passed:   true,
			},
		},
	}

	result := &core.TestResult{
		Test:     td,
		Status:   core.StatusPassed,
		Duration: 250 * time.Millisecond,
		Steps:    []core.StepOutcome{outcome},
	}

	if err := r.OnTestComplete(context.Background(), td, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := r.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	doc := readReport(t, path)

	rawTests := doc["tests"].([]any)
	testObj := rawTests[0].(map[string]any)

	rawSteps, ok := testObj["steps"].([]any)
	if !ok {
		t.Fatalf("expected 'steps' to be a JSON array, got %T", testObj["steps"])
	}
	if len(rawSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rawSteps))
	}

	stepObj := rawSteps[0].(map[string]any)

	checkString := func(field, want string) {
		t.Helper()
		got, _ := stepObj[field].(string)
		if got != want {
			t.Errorf("step.%s: want %q, got %q", field, want, got)
		}
	}
	checkString("id", "execute-0")
	checkString("adapter", "http")
	checkString("action", "request")
	checkString("status", "passed")

	rawAssertions, ok := stepObj["assertions"].([]any)
	if !ok {
		t.Fatalf("expected 'assertions' to be a JSON array, got %T", stepObj["assertions"])
	}
	if len(rawAssertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(rawAssertions))
	}

	a := rawAssertions[0].(map[string]any)
	if a["path"] != "$.status" {
		t.Errorf("assertion.path: want %q, got %q", "$.status", a["path"])
	}
	if a["operator"] != "equals" {
		t.Errorf("assertion.operator: want %q, got %q", "equals", a["operator"])
	}
	if passed, _ := a["passed"].(bool); !passed {
		t.Error("assertion.passed: want true, got false")
	}
}

// TestJSONReporter_ErrorField verifies that a failed test with an error has a
// non-null "error" string in the JSON output.
func TestJSONReporter_ErrorField(t *testing.T) {
	path := outputPath(t)
	r := reporter.NewJSON(path)

	td := &core.TestDefinition{Name: "failing-test", Tags: []string{}}
	result := &core.TestResult{
		Test:     td,
		Status:   core.StatusFailed,
		Duration: 50 * time.Millisecond,
		Error:    errors.New("connection refused"),
	}

	if err := r.OnTestComplete(context.Background(), td, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := r.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	doc := readReport(t, path)
	rawTests := doc["tests"].([]any)
	testObj := rawTests[0].(map[string]any)

	errVal := testObj["error"]
	if errVal == nil {
		t.Fatal("expected test.error to be non-null for a failed test with an error")
	}
	errStr, ok := errVal.(string)
	if !ok {
		t.Fatalf("expected test.error to be a string, got %T", errVal)
	}
	if errStr != "connection refused" {
		t.Errorf("test.error: want %q, got %q", "connection refused", errStr)
	}
}

// TestJSONReporter_FlushCreatesFile verifies that Flush creates the output file
// even when no test events have been recorded.
func TestJSONReporter_FlushCreatesFile(t *testing.T) {
	path := outputPath(t)
	r := reporter.NewJSON(path)

	if err := r.Flush(); err != nil {
		t.Fatalf("Flush on empty reporter: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected output file to exist at %q after Flush", path)
	}
}
