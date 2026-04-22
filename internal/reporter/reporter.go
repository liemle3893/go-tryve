// Package reporter defines the Reporter interface and supporting types for
// delivering test lifecycle events to one or more output sinks.
package reporter

import (
	"context"

	"github.com/liemle3893/autoflow/internal/core"
)

// Reporter is the single interface every output sink must satisfy. Each method
// corresponds to a lifecycle event; implementations must be safe for concurrent
// use but should not block the test runner — heavy I/O should be buffered.
type Reporter interface {
	// OnSuiteStart is called once before any tests execute.
	OnSuiteStart(ctx context.Context, suite *core.SuiteResult) error
	// OnTestStart is called immediately before a test begins.
	OnTestStart(ctx context.Context, test *core.TestDefinition) error
	// OnStepComplete is called after each step finishes, regardless of outcome.
	OnStepComplete(ctx context.Context, step *core.StepDefinition, outcome *core.StepOutcome) error
	// OnTestComplete is called after a test finishes with its final result.
	OnTestComplete(ctx context.Context, test *core.TestDefinition, result *core.TestResult) error
	// OnSuiteComplete is called once after all tests have finished.
	OnSuiteComplete(ctx context.Context, suite *core.SuiteResult, result *core.SuiteResult) error
	// Flush writes any buffered output and should be called before the process exits.
	Flush() error
}

// Multi fans out every lifecycle event to all contained reporters. Errors from
// individual reporters are silently ignored so that one failing reporter cannot
// block the rest of the pipeline.
type Multi struct {
	reporters []Reporter
}

// NewMulti creates a Multi that dispatches events to each of the provided reporters.
func NewMulti(reporters ...Reporter) *Multi {
	return &Multi{reporters: reporters}
}

// OnSuiteStart dispatches the event to all reporters.
func (m *Multi) OnSuiteStart(ctx context.Context, suite *core.SuiteResult) error {
	for _, r := range m.reporters {
		_ = r.OnSuiteStart(ctx, suite)
	}
	return nil
}

// OnTestStart dispatches the event to all reporters.
func (m *Multi) OnTestStart(ctx context.Context, test *core.TestDefinition) error {
	for _, r := range m.reporters {
		_ = r.OnTestStart(ctx, test)
	}
	return nil
}

// OnStepComplete dispatches the event to all reporters.
func (m *Multi) OnStepComplete(ctx context.Context, step *core.StepDefinition, outcome *core.StepOutcome) error {
	for _, r := range m.reporters {
		_ = r.OnStepComplete(ctx, step, outcome)
	}
	return nil
}

// OnTestComplete dispatches the event to all reporters.
func (m *Multi) OnTestComplete(ctx context.Context, test *core.TestDefinition, result *core.TestResult) error {
	for _, r := range m.reporters {
		_ = r.OnTestComplete(ctx, test, result)
	}
	return nil
}

// OnSuiteComplete dispatches the event to all reporters.
func (m *Multi) OnSuiteComplete(ctx context.Context, suite *core.SuiteResult, result *core.SuiteResult) error {
	for _, r := range m.reporters {
		_ = r.OnSuiteComplete(ctx, suite, result)
	}
	return nil
}

// Flush flushes all reporters, returning the first non-nil error encountered.
func (m *Multi) Flush() error {
	var firstErr error
	for _, r := range m.reporters {
		if err := r.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
