package reporter_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liemle3893/autoflow/internal/reporter"
	"github.com/liemle3893/autoflow/internal/core"
)

// makeStep is a helper that builds a minimal StepDefinition for use in tests.
func makeStep(action, description string) *core.StepDefinition {
	return &core.StepDefinition{
		ID:          action + "-id",
		Action:      action,
		Description: description,
	}
}

// TestHTMLReporter_FileCreated verifies that Flush creates a file at the given path.
func TestHTMLReporter_FileCreated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	h := reporter.NewHTML(outPath)
	if err := h.Flush(); err != nil {
		t.Fatalf("Flush returned unexpected error: %v", err)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("expected output file to exist at %q, but it does not", outPath)
	}
}

// TestHTMLReporter_ContainsTitle verifies that the rendered file contains the
// expected report title.
func TestHTMLReporter_ContainsTitle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	h := reporter.NewHTML(outPath)
	if err := h.Flush(); err != nil {
		t.Fatalf("Flush returned unexpected error: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Autoflow Test Report") {
		t.Errorf("expected output to contain %q", "Autoflow Test Report")
	}
}

// TestHTMLReporter_ContainsTestNames verifies that reported test names appear in the output.
func TestHTMLReporter_ContainsTestNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	h := reporter.NewHTML(outPath)

	ctx := context.Background()
	testA := &core.TestDefinition{Name: "login-flow", Description: "Tests the login path"}
	testB := &core.TestDefinition{Name: "checkout-flow", Description: "Tests the checkout path"}

	resultA := &core.TestResult{
		Test:     testA,
		Status:   core.StatusPassed,
		Duration: 120 * time.Millisecond,
	}
	resultB := &core.TestResult{
		Test:     testB,
		Status:   core.StatusFailed,
		Duration: 300 * time.Millisecond,
	}

	if err := h.OnTestComplete(ctx, testA, resultA); err != nil {
		t.Fatalf("OnTestComplete(testA): %v", err)
	}
	if err := h.OnTestComplete(ctx, testB, resultB); err != nil {
		t.Fatalf("OnTestComplete(testB): %v", err)
	}
	if err := h.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	out := string(content)

	for _, name := range []string{"login-flow", "checkout-flow"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected output to contain test name %q", name)
		}
	}
}

// TestHTMLReporter_ContainsPassedAndFailed verifies that the summary section
// renders both "passed" and "failed" labels in the output.
func TestHTMLReporter_ContainsPassedAndFailed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	h := reporter.NewHTML(outPath)

	ctx := context.Background()

	testPass := &core.TestDefinition{Name: "healthy-check"}
	testFail := &core.TestDefinition{Name: "broken-service"}

	if err := h.OnTestComplete(ctx, testPass, &core.TestResult{
		Test: testPass, Status: core.StatusPassed, Duration: 50 * time.Millisecond,
	}); err != nil {
		t.Fatalf("OnTestComplete(pass): %v", err)
	}
	if err := h.OnTestComplete(ctx, testFail, &core.TestResult{
		Test: testFail, Status: core.StatusFailed, Duration: 80 * time.Millisecond,
	}); err != nil {
		t.Fatalf("OnTestComplete(fail): %v", err)
	}
	if err := h.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	out := string(content)

	if !strings.Contains(out, "passed") {
		t.Errorf("expected output to contain %q", "passed")
	}
	if !strings.Contains(out, "failed") {
		t.Errorf("expected output to contain %q", "failed")
	}
}

// TestHTMLReporter_StepsAndAssertions verifies that step details and failed
// assertion information are rendered in the output.
func TestHTMLReporter_StepsAndAssertions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	h := reporter.NewHTML(outPath)
	ctx := context.Background()

	step := makeStep("http.request", "Call the API")
	test := &core.TestDefinition{Name: "api-test"}
	result := &core.TestResult{
		Test:     test,
		Status:   core.StatusFailed,
		Duration: 200 * time.Millisecond,
		Steps: []core.StepOutcome{
			{
				Step:     step,
				Phase:    core.PhaseVerify,
				Status:   core.StatusFailed,
				Duration: 150 * time.Millisecond,
				Assertions: []core.AssertionOutcome{
					{
						Path:     "$.status",
						Operator: "equals",
						Expected: 200,
						Actual:   500,
						Passed:   false,
						Message:  "status code mismatch",
					},
				},
			},
		},
	}

	if err := h.OnTestComplete(ctx, test, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := h.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	out := string(content)

	for _, want := range []string{
		"api-test",
		"Call the API",
		"$.status",
		"equals",
		"status code mismatch",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

// TestHTMLReporter_SkippedTest verifies that a skipped test renders the "skipped" label.
func TestHTMLReporter_SkippedTest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	h := reporter.NewHTML(outPath)
	ctx := context.Background()

	test := &core.TestDefinition{Name: "optional-feature", Skip: true, SkipReason: "not ready"}
	result := &core.TestResult{
		Test:     test,
		Status:   core.StatusSkipped,
		Duration: 0,
	}

	if err := h.OnTestComplete(ctx, test, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := h.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	out := string(content)

	if !strings.Contains(out, "skipped") {
		t.Errorf("expected output to contain %q", "skipped")
	}
	if !strings.Contains(out, "optional-feature") {
		t.Errorf("expected output to contain test name %q", "optional-feature")
	}
}

// TestHTMLReporter_ImplementsReporterInterface is a compile-time check that
// *HTML satisfies the Reporter interface.
func TestHTMLReporter_ImplementsReporterInterface(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var _ reporter.Reporter = reporter.NewHTML(filepath.Join(dir, "report.html"))
}

// TestHTMLReporter_InvalidPath verifies that Flush returns an error when the
// output path is not writable.
func TestHTMLReporter_InvalidPath(t *testing.T) {
	t.Parallel()

	h := reporter.NewHTML("/nonexistent-directory/report.html")
	if err := h.Flush(); err == nil {
		t.Error("expected Flush to return an error for an unwritable path, but got nil")
	}
}

// TestHTMLReporter_LifecycleNoError verifies that all lifecycle methods return
// nil without panicking.
func TestHTMLReporter_LifecycleNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h := reporter.NewHTML(filepath.Join(dir, "report.html"))
	ctx := context.Background()

	suite := &core.SuiteResult{}
	test := &core.TestDefinition{Name: "smoke-test"}
	step := makeStep("shell.exec", "")
	outcome := &core.StepOutcome{Step: step, Status: core.StatusPassed, Duration: 10 * time.Millisecond}

	if err := h.OnSuiteStart(ctx, suite); err != nil {
		t.Errorf("OnSuiteStart: %v", err)
	}
	if err := h.OnTestStart(ctx, test); err != nil {
		t.Errorf("OnTestStart: %v", err)
	}
	if err := h.OnStepComplete(ctx, step, outcome); err != nil {
		t.Errorf("OnStepComplete: %v", err)
	}
	if err := h.OnTestComplete(ctx, test, &core.TestResult{
		Test: test, Status: core.StatusPassed, Duration: 10 * time.Millisecond,
	}); err != nil {
		t.Errorf("OnTestComplete: %v", err)
	}
	if err := h.OnSuiteComplete(ctx, suite, suite); err != nil {
		t.Errorf("OnSuiteComplete: %v", err)
	}
	if err := h.Flush(); err != nil {
		t.Errorf("Flush: %v", err)
	}
}
