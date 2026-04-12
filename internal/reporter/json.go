package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// jsonSummary holds the aggregated counts and total duration for the suite.
type jsonSummary struct {
	Total    int    `json:"total"`
	Passed   int    `json:"passed"`
	Failed   int    `json:"failed"`
	Skipped  int    `json:"skipped"`
	Duration string `json:"duration"`
}

// jsonAssertion is the JSON representation of a single assertion outcome.
type jsonAssertion struct {
	Path     string `json:"path"`
	Operator string `json:"operator"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual"`
	Passed   bool   `json:"passed"`
}

// jsonStep is the JSON representation of a single step outcome.
type jsonStep struct {
	ID         string          `json:"id"`
	Adapter    string          `json:"adapter"`
	Action     string          `json:"action"`
	Status     string          `json:"status"`
	Duration   string          `json:"duration"`
	Assertions []jsonAssertion `json:"assertions"`
}

// jsonTest is the JSON representation of a single test result.
type jsonTest struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"`
	Duration string     `json:"duration"`
	Tags     []string   `json:"tags"`
	Priority string     `json:"priority"`
	Steps    []jsonStep `json:"steps"`
	Error    *string    `json:"error"`
}

// jsonReport is the top-level JSON document written by the JSON reporter.
type jsonReport struct {
	Summary jsonSummary `json:"summary"`
	Tests   []jsonTest  `json:"tests"`
}

// JSON is a Reporter implementation that accumulates test results and writes
// a structured JSON report to a file on Flush.
type JSON struct {
	mu         sync.Mutex
	outputPath string
	tests      []jsonTest
	suite      *tryve.SuiteResult
}

// NewJSON creates a JSON reporter that will write its output to outputPath on Flush.
func NewJSON(outputPath string) *JSON {
	return &JSON{
		outputPath: outputPath,
		tests:      []jsonTest{},
	}
}

// OnSuiteStart stores the initial suite reference for later use on completion.
func (j *JSON) OnSuiteStart(_ context.Context, suite *tryve.SuiteResult) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.suite = suite
	return nil
}

// OnTestStart is a no-op; data is captured at completion.
func (j *JSON) OnTestStart(_ context.Context, _ *tryve.TestDefinition) error {
	return nil
}

// OnStepComplete is a no-op; step data is captured as part of OnTestComplete.
func (j *JSON) OnStepComplete(_ context.Context, _ *tryve.StepDefinition, _ *tryve.StepOutcome) error {
	return nil
}

// OnTestComplete converts the completed test result into the JSON representation
// and appends it to the internal accumulator.
func (j *JSON) OnTestComplete(_ context.Context, test *tryve.TestDefinition, result *tryve.TestResult) error {
	jt := convertTestResult(test, result)

	j.mu.Lock()
	defer j.mu.Unlock()
	j.tests = append(j.tests, jt)
	return nil
}

// OnSuiteComplete stores the final suite result so that Flush can write accurate summary counts.
func (j *JSON) OnSuiteComplete(_ context.Context, _ *tryve.SuiteResult, result *tryve.SuiteResult) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.suite = result
	return nil
}

// Flush marshals the accumulated results to JSON and writes them to the output file.
// It uses MarshalIndent for human-readable formatting.
func (j *JSON) Flush() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	report := jsonReport{
		Tests: j.tests,
	}

	if j.suite != nil {
		report.Summary = jsonSummary{
			Total:    j.suite.Total,
			Passed:   j.suite.Passed,
			Failed:   j.suite.Failed,
			Skipped:  j.suite.Skipped,
			Duration: j.suite.Duration.String(),
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("json reporter: marshal failed: %w", err)
	}

	if err := os.WriteFile(j.outputPath, data, 0o644); err != nil {
		return fmt.Errorf("json reporter: write file %q: %w", j.outputPath, err)
	}

	return nil
}

// convertTestResult maps a tryve.TestResult into the serialisable jsonTest structure.
func convertTestResult(test *tryve.TestDefinition, result *tryve.TestResult) jsonTest {
	steps := make([]jsonStep, 0, len(result.Steps))
	for _, s := range result.Steps {
		steps = append(steps, convertStepOutcome(&s))
	}

	tags := test.Tags
	if tags == nil {
		tags = []string{}
	}

	jt := jsonTest{
		Name:     test.Name,
		Status:   string(result.Status),
		Duration: result.Duration.String(),
		Tags:     tags,
		Priority: string(test.Priority),
		Steps:    steps,
		Error:    nil,
	}

	if result.Error != nil {
		msg := result.Error.Error()
		jt.Error = &msg
	}

	return jt
}

// convertStepOutcome maps a tryve.StepOutcome into the serialisable jsonStep structure.
func convertStepOutcome(outcome *tryve.StepOutcome) jsonStep {
	assertions := make([]jsonAssertion, 0, len(outcome.Assertions))
	for _, a := range outcome.Assertions {
		assertions = append(assertions, jsonAssertion{
			Path:     a.Path,
			Operator: a.Operator,
			Expected: a.Expected,
			Actual:   a.Actual,
			Passed:   a.Passed,
		})
	}

	id := ""
	adapter := ""
	action := ""
	if outcome.Step != nil {
		id = outcome.Step.ID
		adapter = outcome.Step.Adapter
		action = outcome.Step.Action
	}

	return jsonStep{
		ID:         id,
		Adapter:    adapter,
		Action:     action,
		Status:     string(outcome.Status),
		Duration:   outcome.Duration.String(),
		Assertions: assertions,
	}
}
