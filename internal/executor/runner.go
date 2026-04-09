package executor

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/liemle3893/e2e-runner/internal/adapter"
	"github.com/liemle3893/e2e-runner/internal/reporter"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// phaseEntry groups a phase identifier with its ordered step list.
type phaseEntry struct {
	phase tryve.TestPhase
	steps []tryve.StepDefinition
}

// RunTest executes a single test through all lifecycle phases (setup, execute,
// verify, teardown) and returns an aggregated TestResult.
//
// Parameters:
//   - ctx            – parent context; a child deadline is created when td.Timeout > 0.
//   - td             – the parsed test definition to execute.
//   - registry       – adapter registry used to resolve step adapters.
//   - rep            – reporter that receives lifecycle events.
//   - defaultRetries – retry count used when td.Retries is not set (0 = no retries).
//   - defaultRetryDelay – base retry back-off delay in milliseconds.
//
// If td.Skip is true the function returns immediately with StatusSkipped without
// calling any reporter methods beyond OnTestStart/OnTestComplete.
func RunTest(
	ctx context.Context,
	td *tryve.TestDefinition,
	registry *adapter.Registry,
	rep reporter.Reporter,
	defaultRetries int,
	defaultRetryDelay int,
	baseURL string,
	configVars map[string]any,
) *tryve.TestResult {
	// 1. Early-return for skipped tests.
	if td.Skip {
		result := &tryve.TestResult{
			Test:   td,
			Status: tryve.StatusSkipped,
		}
		return result
	}

	// 2. Apply per-test timeout when configured.
	runCtx := ctx
	if td.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(td.Timeout)*time.Millisecond)
		defer cancel()
	}

	// 3. Notify reporter that the test is starting.
	_ = rep.OnTestStart(runCtx, td)

	start := time.Now()

	// 4. Build the interpolation context seeded with config + test variables.
	interpCtx := tryve.NewInterpolationContext()
	interpCtx.BaseURL = baseURL

	// Populate environment variables from the process.
	for _, entry := range os.Environ() {
		if k, v, ok := strings.Cut(entry, "="); ok {
			interpCtx.Env[k] = v
		}
	}

	// Config-level variables (lower priority than test-level).
	for k, v := range configVars {
		interpCtx.Variables[k] = v
	}

	// Test-level variables override config variables.
	for k, v := range td.Variables {
		interpCtx.Variables[k] = v
	}

	// 5. Resolve retry settings.
	maxRetries := defaultRetries
	if td.Retries > 0 {
		maxRetries = td.Retries
	}
	baseDelay := time.Duration(defaultRetryDelay) * time.Millisecond

	// 6. Execute phases in canonical order.
	phases := []phaseEntry{
		{tryve.PhaseSetup, td.Setup},
		{tryve.PhaseExecute, td.Execute},
		{tryve.PhaseVerify, td.Verify},
		{tryve.PhaseTeardown, td.Teardown},
	}

	var (
		steps  []tryve.StepOutcome
		failed bool
		runErr error
	)

	for _, pe := range phases {
		if len(pe.steps) == 0 {
			continue
		}

		// Skip non-teardown phases when a previous phase has already failed.
		if failed && pe.phase != tryve.PhaseTeardown {
			continue
		}

		for i := range pe.steps {
			step := &pe.steps[i]

			outcome, _ := ExecuteStepWithRetry(runCtx, step, registry, interpCtx, maxRetries, baseDelay)

			// Stamp the phase on the outcome so callers can inspect it.
			outcome.Phase = pe.phase

			steps = append(steps, *outcome)
			_ = rep.OnStepComplete(runCtx, step, outcome)

			if outcome.Status == tryve.StatusFailed {
				// Record the first failure error.
				if runErr == nil {
					runErr = outcome.Error
				}
				failed = true

				// In teardown, continue executing remaining steps despite failure.
				if pe.phase == tryve.PhaseTeardown {
					continue
				}
				// In any other phase, stop processing further steps in this phase.
				break
			}
		}
	}

	// 7. Determine final status.
	status := tryve.StatusPassed
	if failed {
		status = tryve.StatusFailed
	}

	result := &tryve.TestResult{
		Test:     td,
		Status:   status,
		Duration: time.Since(start),
		Steps:    steps,
		Error:    runErr,
	}

	// 8. Notify reporter of completion.
	_ = rep.OnTestComplete(runCtx, td, result)

	return result
}
