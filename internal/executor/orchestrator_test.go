package executor_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/liemle3893/go-tryve/internal/config"
	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

// newOrchestratorConfig returns a minimal LoadedConfig suitable for orchestrator tests.
// Parallel=1 keeps test execution deterministic; Timeout/RetryDelay are non-zero
// to exercise the defaults path without actually waiting.
func newOrchestratorConfig() *config.LoadedConfig {
	return &config.LoadedConfig{
		Defaults: config.DefaultsConfig{
			Parallel:   1,
			Timeout:    30000,
			RetryDelay: 0,
			Retries:    0,
		},
	}
}

// newEchoTest builds a trivial TestDefinition with a shell/exec echo step.
// name is used for both the test Name and the echo payload so it can be
// identified in results.
func newEchoTest(name string, tags ...string) *tryve.TestDefinition {
	return &tryve.TestDefinition{
		Name: name,
		Tags: tags,
		Execute: []tryve.StepDefinition{
			{
				ID:      name + "-step",
				Adapter: "shell",
				Action:  "exec",
				Params:  map[string]any{"command": "echo " + name},
			},
		},
	}
}

// newFailingTest builds a TestDefinition whose execute step always exits non-zero.
func newFailingTest(name string) *tryve.TestDefinition {
	return &tryve.TestDefinition{
		Name: name,
		Execute: []tryve.StepDefinition{
			{
				ID:      name + "-step",
				Adapter: "shell",
				Action:  "exec",
				Params:  map[string]any{"command": "exit 1"},
				// Assert that exitCode == 0 so the step is recorded as failed.
				Assert: map[string]any{
					"path":   "$.exitCode",
					"equals": float64(0),
				},
			},
		},
	}
}

// TestOrchestrator_RunAll verifies that an orchestrator running two passing
// shell tests reports total=2, passed=2, failed=0, skipped=0.
func TestOrchestrator_RunAll(t *testing.T) {
	reg := newTestRegistry("")
	var buf bytes.Buffer
	rep := reporter.NewConsole(&buf, false, false)
	cfg := newOrchestratorConfig()

	orch := executor.NewOrchestrator(reg, rep, cfg)

	tests := []*tryve.TestDefinition{
		newEchoTest("test-a"),
		newEchoTest("test-b"),
	}

	result := orch.Run(context.Background(), tests)

	if result == nil {
		t.Fatal("expected non-nil SuiteResult")
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
	if result.Passed != 2 {
		t.Errorf("expected passed=2, got %d", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
	}
	if result.Skipped != 0 {
		t.Errorf("expected skipped=0, got %d", result.Skipped)
	}
}

// TestOrchestrator_BailOnFailure verifies that when bail=true a failing test
// prevents the subsequent test from running (it is recorded as skipped).
// With parallel=1 the failing test is guaranteed to run before the second.
func TestOrchestrator_BailOnFailure(t *testing.T) {
	reg := newTestRegistry("")
	var buf bytes.Buffer
	rep := reporter.NewConsole(&buf, false, false)
	cfg := newOrchestratorConfig() // parallel=1 ensures ordering

	orch := executor.NewOrchestrator(reg, rep, cfg)
	orch.SetBail(true)

	tests := []*tryve.TestDefinition{
		newFailingTest("fail-first"),
		newEchoTest("should-be-skipped"),
	}

	result := orch.Run(context.Background(), tests)

	if result == nil {
		t.Fatal("expected non-nil SuiteResult")
	}
	// Only the first test actually ran (and failed).  The second was skipped due to bail.
	// Total must equal the number of tests regardless of bail.
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
	if result.Failed < 1 {
		t.Errorf("expected at least 1 failed test, got %d", result.Failed)
	}
	if result.Skipped < 1 {
		t.Errorf("expected at least 1 skipped test (bailed), got %d", result.Skipped)
	}
}

// TestOrchestrator_FilterByTag verifies that FilterTests retains only tests
// whose tag list intersects the requested tags.
func TestOrchestrator_FilterByTag(t *testing.T) {
	allTests := []*tryve.TestDefinition{
		newEchoTest("smoke-a", "smoke", "regression"),
		newEchoTest("regression-only", "regression"),
		newEchoTest("smoke-b", "smoke"),
		newEchoTest("untagged"),
	}

	filtered := executor.FilterTests(allTests, executor.FilterOptions{
		Tags: []string{"smoke"},
	})

	if len(filtered) != 2 {
		t.Errorf("expected 2 smoke-tagged tests, got %d", len(filtered))
	}
	for _, td := range filtered {
		found := false
		for _, tag := range td.Tags {
			if tag == "smoke" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("test %q missing 'smoke' tag after filter", td.Name)
		}
	}
}

// TestOrchestrator_FilterByGrep verifies that FilterTests retains only tests
// whose name matches the given pattern (regex or substring).
func TestOrchestrator_FilterByGrep(t *testing.T) {
	allTests := []*tryve.TestDefinition{
		newEchoTest("API Login"),
		newEchoTest("API Logout"),
		newEchoTest("UI Dashboard"),
		newEchoTest("API Create User"),
	}

	filtered := executor.FilterTests(allTests, executor.FilterOptions{
		Grep: "API",
	})

	if len(filtered) != 3 {
		t.Errorf("expected 3 API tests, got %d", len(filtered))
	}
	for _, td := range filtered {
		if len(td.Name) < 3 || td.Name[:3] != "API" {
			t.Errorf("test %q should not be included in API grep results", td.Name)
		}
	}
}

// TestOrchestrator_FilterByPriority verifies that FilterTests retains only
// tests whose Priority matches the requested value exactly.
func TestOrchestrator_FilterByPriority(t *testing.T) {
	allTests := []*tryve.TestDefinition{
		{Name: "critical", Priority: tryve.PriorityP0},
		{Name: "high", Priority: tryve.PriorityP1},
		{Name: "medium", Priority: tryve.PriorityP2},
		{Name: "low", Priority: tryve.PriorityP3},
		{Name: "no-priority"},
	}

	filtered := executor.FilterTests(allTests, executor.FilterOptions{
		Priority: "P0",
	})

	if len(filtered) != 1 {
		t.Errorf("expected 1 P0 test, got %d", len(filtered))
	}
	if filtered[0].Name != "critical" {
		t.Errorf("expected 'critical', got %q", filtered[0].Name)
	}
}

// TestOrchestrator_DependencyOrdering verifies that a test with a `depends`
// entry runs after the named dependency has completed.  With parallel=1 and
// sequential scheduling the dependent test must come after its prerequisite.
func TestOrchestrator_DependencyOrdering(t *testing.T) {
	reg := newTestRegistry("")
	var buf bytes.Buffer
	rep := reporter.NewConsole(&buf, false, false)
	cfg := newOrchestratorConfig()

	orch := executor.NewOrchestrator(reg, rep, cfg)

	// Intentionally pass tests in reverse dependency order to confirm topo sort.
	dependent := &tryve.TestDefinition{
		Name:    "dependent",
		Depends: []string{"prerequisite"},
		Execute: []tryve.StepDefinition{
			{
				ID:      "dep-step",
				Adapter: "shell",
				Action:  "exec",
				Params:  map[string]any{"command": "echo dependent"},
			},
		},
	}
	prerequisite := newEchoTest("prerequisite")

	result := orch.Run(context.Background(), []*tryve.TestDefinition{dependent, prerequisite})

	if result == nil {
		t.Fatal("expected non-nil SuiteResult")
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
	if result.Passed != 2 {
		t.Errorf("expected both tests to pass, got passed=%d failed=%d skipped=%d",
			result.Passed, result.Failed, result.Skipped)
	}
}

// TestOrchestrator_SkipOnDependencyFailure verifies that a test is skipped
// when one of its dependencies failed.
func TestOrchestrator_SkipOnDependencyFailure(t *testing.T) {
	reg := newTestRegistry("")
	var buf bytes.Buffer
	rep := reporter.NewConsole(&buf, false, false)
	cfg := newOrchestratorConfig()

	orch := executor.NewOrchestrator(reg, rep, cfg)

	failing := newFailingTest("prereq-fail")
	dependent := &tryve.TestDefinition{
		Name:    "should-skip",
		Depends: []string{"prereq-fail"},
		Execute: []tryve.StepDefinition{
			{
				ID:      "dep-step",
				Adapter: "shell",
				Action:  "exec",
				Params:  map[string]any{"command": "echo should-not-run"},
			},
		},
	}

	result := orch.Run(context.Background(), []*tryve.TestDefinition{failing, dependent})

	if result == nil {
		t.Fatal("expected non-nil SuiteResult")
	}
	if result.Skipped < 1 {
		t.Errorf("expected dependent test to be skipped due to failed dependency, skipped=%d", result.Skipped)
	}
}
