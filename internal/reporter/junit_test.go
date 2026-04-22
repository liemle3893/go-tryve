package reporter_test

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liemle3893/autoflow/internal/reporter"
	"github.com/liemle3893/autoflow/internal/core"
)

// parseJUnitSuites reads and decodes the JUnit XML written to path.
func parseJUnitSuites(t *testing.T, path string) junitTestSuites {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read junit output: %v", err)
	}
	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("unmarshal junit XML: %v\ncontent:\n%s", err, data)
	}
	return suites
}

// junitTestSuites mirrors the internal struct for unmarshalling in tests.
type junitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Skipped  int             `xml:"skipped,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name    string        `xml:"name,attr"`
	Time    string        `xml:"time,attr"`
	Failure *junitFailure `xml:"failure"`
	Skipped *junitSkipped `xml:"skipped"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Content string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

// tmpFile creates a temporary file and returns its path, registering cleanup.
func tmpFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "junit-*.xml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

// TestJUnit_PassedTest verifies that a passing test produces the correct XML structure.
func TestJUnit_PassedTest(t *testing.T) {
	path := tmpFile(t)
	j := reporter.NewJUnit(path)

	test := &core.TestDefinition{Name: "login-flow"}
	result := &core.TestResult{
		Test:     test,
		Status:   core.StatusPassed,
		Duration: 250 * time.Millisecond,
	}

	if err := j.OnTestComplete(context.Background(), test, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := j.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	suites := parseJUnitSuites(t, path)

	if suites.Tests != 1 {
		t.Errorf("testsuites tests=%d, want 1", suites.Tests)
	}
	if suites.Failures != 0 {
		t.Errorf("testsuites failures=%d, want 0", suites.Failures)
	}
	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 testsuite, got %d", len(suites.Suites))
	}

	suite := suites.Suites[0]
	if suite.Name != "autoflow" {
		t.Errorf("testsuite name=%q, want %q", suite.Name, "autoflow")
	}
	if suite.Tests != 1 {
		t.Errorf("testsuite tests=%d, want 1", suite.Tests)
	}
	if suite.Failures != 0 {
		t.Errorf("testsuite failures=%d, want 0", suite.Failures)
	}
	if len(suite.Cases) != 1 {
		t.Fatalf("expected 1 testcase, got %d", len(suite.Cases))
	}

	tc := suite.Cases[0]
	if tc.Name != "login-flow" {
		t.Errorf("testcase name=%q, want %q", tc.Name, "login-flow")
	}
	if tc.Failure != nil {
		t.Error("passed test should not have a <failure> element")
	}
	if tc.Skipped != nil {
		t.Error("passed test should not have a <skipped> element")
	}
}

// TestJUnit_FailedTest verifies that a failed test includes a <failure> element
// with the assertion message and correct counts.
func TestJUnit_FailedTest(t *testing.T) {
	path := tmpFile(t)
	j := reporter.NewJUnit(path)

	stepDef := &core.StepDefinition{ID: "execute-0", Action: "http.request"}
	test := &core.TestDefinition{Name: "checkout-flow"}
	result := &core.TestResult{
		Test:     test,
		Status:   core.StatusFailed,
		Duration: 300 * time.Millisecond,
		Steps: []core.StepOutcome{
			{
				Step:   stepDef,
				Status: core.StatusFailed,
				Assertions: []core.AssertionOutcome{
					{
						Path:    "$.status",
						Passed:  false,
						Message: "assertion failed: expected 200, got 404",
					},
				},
			},
		},
	}

	if err := j.OnTestComplete(context.Background(), test, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := j.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	suites := parseJUnitSuites(t, path)

	if suites.Failures != 1 {
		t.Errorf("testsuites failures=%d, want 1", suites.Failures)
	}

	tc := suites.Suites[0].Cases[0]
	if tc.Name != "checkout-flow" {
		t.Errorf("testcase name=%q, want %q", tc.Name, "checkout-flow")
	}
	if tc.Failure == nil {
		t.Fatal("expected <failure> element, got nil")
	}
	if !strings.Contains(tc.Failure.Message, "assertion failed") {
		t.Errorf("failure message=%q, want it to contain %q", tc.Failure.Message, "assertion failed")
	}
	if !strings.Contains(tc.Failure.Content, "execute-0") {
		t.Errorf("failure content=%q, want it to contain step id %q", tc.Failure.Content, "execute-0")
	}
	if !strings.Contains(tc.Failure.Content, "$.status") {
		t.Errorf("failure content=%q, want it to contain path %q", tc.Failure.Content, "$.status")
	}
}

// TestJUnit_SkippedTest verifies that a skipped test produces a <skipped> element
// with the skip reason and increments the skipped counter.
func TestJUnit_SkippedTest(t *testing.T) {
	path := tmpFile(t)
	j := reporter.NewJUnit(path)

	test := &core.TestDefinition{Name: "payment-flow", SkipReason: "payment gateway offline"}
	result := &core.TestResult{
		Test:   test,
		Status: core.StatusSkipped,
	}

	if err := j.OnTestComplete(context.Background(), test, result); err != nil {
		t.Fatalf("OnTestComplete: %v", err)
	}
	if err := j.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	suites := parseJUnitSuites(t, path)
	suite := suites.Suites[0]

	if suite.Skipped != 1 {
		t.Errorf("testsuite skipped=%d, want 1", suite.Skipped)
	}

	tc := suite.Cases[0]
	if tc.Skipped == nil {
		t.Fatal("expected <skipped> element, got nil")
	}
	if !strings.Contains(tc.Skipped.Message, "payment gateway offline") {
		t.Errorf("skipped message=%q, want it to contain skip reason", tc.Skipped.Message)
	}
	if tc.Failure != nil {
		t.Error("skipped test should not have a <failure> element")
	}
}

// TestJUnit_MultipleMixedTests verifies that counts and structure are correct
// when the suite contains a mix of passed, failed, and skipped tests.
func TestJUnit_MultipleMixedTests(t *testing.T) {
	path := tmpFile(t)
	j := reporter.NewJUnit(path)
	ctx := context.Background()

	tests := []struct {
		def    *core.TestDefinition
		result *core.TestResult
	}{
		{
			def: &core.TestDefinition{Name: "test-pass"},
			result: &core.TestResult{
				Status:   core.StatusPassed,
				Duration: 100 * time.Millisecond,
			},
		},
		{
			def: &core.TestDefinition{Name: "test-fail"},
			result: &core.TestResult{
				Status:   core.StatusFailed,
				Duration: 200 * time.Millisecond,
				Error:    errors.New("unexpected error"),
			},
		},
		{
			def:    &core.TestDefinition{Name: "test-skip", SkipReason: "not ready"},
			result: &core.TestResult{Status: core.StatusSkipped},
		},
	}

	for _, tc := range tests {
		tc.result.Test = tc.def
		if err := j.OnTestComplete(ctx, tc.def, tc.result); err != nil {
			t.Fatalf("OnTestComplete(%s): %v", tc.def.Name, err)
		}
	}

	if err := j.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	suites := parseJUnitSuites(t, path)

	if suites.Tests != 3 {
		t.Errorf("testsuites tests=%d, want 3", suites.Tests)
	}
	if suites.Failures != 1 {
		t.Errorf("testsuites failures=%d, want 1", suites.Failures)
	}

	suite := suites.Suites[0]
	if suite.Tests != 3 {
		t.Errorf("testsuite tests=%d, want 3", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("testsuite failures=%d, want 1", suite.Failures)
	}
	if suite.Skipped != 1 {
		t.Errorf("testsuite skipped=%d, want 1", suite.Skipped)
	}
	if len(suite.Cases) != 3 {
		t.Fatalf("expected 3 testcase elements, got %d", len(suite.Cases))
	}

	// Verify individual case names are present.
	names := map[string]bool{}
	for _, tc := range suite.Cases {
		names[tc.Name] = true
	}
	for _, want := range []string{"test-pass", "test-fail", "test-skip"} {
		if !names[want] {
			t.Errorf("testcase %q not found in output", want)
		}
	}
}

// TestJUnit_XMLHeader verifies that the output file begins with the standard
// XML declaration line.
func TestJUnit_XMLHeader(t *testing.T) {
	path := tmpFile(t)
	j := reporter.NewJUnit(path)

	test := &core.TestDefinition{Name: "minimal"}
	result := &core.TestResult{Test: test, Status: core.StatusPassed}
	_ = j.OnTestComplete(context.Background(), test, result)

	if err := j.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.HasPrefix(string(data), "<?xml version=") {
		t.Errorf("output does not start with XML declaration, got: %.40s", data)
	}
}

// TestJUnit_FlushToInvalidPath verifies that Flush returns an error when the
// output path is not writable.
func TestJUnit_FlushToInvalidPath(t *testing.T) {
	j := reporter.NewJUnit("/nonexistent-dir/junit.xml")
	if err := j.Flush(); err == nil {
		t.Error("expected error when writing to invalid path, got nil")
	}
}

// TestJUnit_ImplementsReporterInterface is a compile-time assertion that *JUnit
// satisfies the Reporter interface.
func TestJUnit_ImplementsReporterInterface(t *testing.T) {
	var _ reporter.Reporter = (*reporter.JUnit)(nil)
}
