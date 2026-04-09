package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// ANSI escape codes used by the Console reporter.
const (
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiDim    = "\033[2m"
	ansiReset  = "\033[0m"
)

// Console is a Reporter implementation that writes human-readable, optionally
// colourised output to an io.Writer (usually os.Stdout).
type Console struct {
	w       io.Writer
	verbose bool
	debug   bool
	color   bool
}

// NewConsole creates a Console reporter writing to w.
func NewConsole(w io.Writer, verbose, color bool) *Console {
	return &Console{w: w, verbose: verbose, color: color}
}

// NewConsoleWithDebug creates a Console reporter with debug mode.
func NewConsoleWithDebug(w io.Writer, verbose, debug, color bool) *Console {
	return &Console{w: w, verbose: verbose || debug, debug: debug, color: color}
}

// NewConsoleFromEnv creates a Console reporter writing to os.Stdout.
func NewConsoleFromEnv(verbose bool) *Console {
	color := os.Getenv("NO_COLOR") == ""
	return NewConsole(os.Stdout, verbose, color)
}

// NewConsoleFromEnvWithDebug creates a Console reporter with debug mode.
func NewConsoleFromEnvWithDebug(verbose, debug bool) *Console {
	color := os.Getenv("NO_COLOR") == ""
	return NewConsoleWithDebug(os.Stdout, verbose, debug, color)
}

// styled wraps text in the given ANSI style code.
func (c *Console) styled(text, style string) string {
	if !c.color {
		return text
	}
	return style + text + ansiReset
}

// OnSuiteStart prints the header.
func (c *Console) OnSuiteStart(_ context.Context, _ *tryve.SuiteResult) error {
	fmt.Fprintln(c.w, c.styled("Tryve Test Runner", ansiBold))
	return nil
}

// OnTestStart prints the test name when verbose.
func (c *Console) OnTestStart(_ context.Context, test *tryve.TestDefinition) error {
	if c.verbose {
		fmt.Fprintf(c.w, "\n%s\n", c.styled("RUN "+test.Name, ansiCyan))
	}
	return nil
}

// OnStepComplete prints step results with varying detail levels.
func (c *Console) OnStepComplete(_ context.Context, _ *tryve.StepDefinition, outcome *tryve.StepOutcome) error {
	if !c.verbose {
		return nil
	}

	var marker string
	if outcome.Status == tryve.StatusPassed {
		marker = c.styled("+", ansiGreen)
	} else {
		marker = c.styled("x", ansiRed)
	}

	desc := stepDescription(outcome)
	fmt.Fprintf(c.w, "  %s %s (%s)\n", marker, desc, outcome.Duration)

	// Debug mode: show full request/response data for every step.
	if c.debug && outcome.Result != nil {
		c.printDebugData(outcome)
	}

	// Show errors and failed assertions for failed steps.
	if outcome.Status == tryve.StatusFailed || outcome.Status == tryve.StatusWarned {
		if outcome.Error != nil {
			fmt.Fprintf(c.w, "      %s %v\n", c.styled("ERR", ansiRed), outcome.Error)
		}
		for _, a := range outcome.Assertions {
			if !a.Passed {
				fmt.Fprintf(c.w, "      %s %s %s: expected %v, got %v\n",
					c.styled("ASSERT", ansiRed), a.Path, a.Operator, a.Expected, a.Actual)
			}
		}
	}

	return nil
}

// printDebugData outputs full request/response details for a step.
func (c *Console) printDebugData(outcome *tryve.StepOutcome) {
	step := outcome.Step
	// Use resolved params (post-interpolation) for debug, fall back to step.Params
	params := outcome.ResolvedParams
	if params == nil {
		params = step.Params
	}
	data := outcome.Result.Data
	meta := outcome.Result.Metadata
	dim := ansiDim

	switch step.Adapter {
	case "http":
		c.printHTTPDebug(params, data, meta, dim)
	case "shell":
		c.printShellDebug(params, data, dim)
	case "postgresql":
		c.printDBDebug("pg", params, data, dim)
	case "mongodb":
		c.printDBDebug("mongo", params, data, dim)
	case "redis":
		c.printRedisDebug(params, data, dim)
	case "kafka", "eventhub":
		c.printEventDebug(params, data, dim)
	}
}

func (c *Console) printHTTPDebug(params map[string]any, data, meta map[string]any, dim string) {
	// Request — use metadata for resolved URL, fall back to params
	method, _ := meta["method"].(string)
	url, _ := meta["url"].(string)
	if method == "" {
		method, _ = params["method"].(string)
		if method == "" {
			method = "GET"
		}
	}
	if url == "" {
		url, _ = params["url"].(string)
	}
	fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("→ %s %s", method, url), dim))

	// Response headers (from actual response)
	if headers, ok := data["headers"].(map[string]any); ok && len(headers) > 0 {
		for k, v := range headers {
			fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("  %s: %v", k, v), dim))
		}
	}

	// Response
	status, _ := data["status"].(float64)
	statusText, _ := data["statusText"].(string)
	fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("← %d %s", int(status), statusText), dim))

	// Response headers
	if headers, ok := data["headers"].(map[string]any); ok {
		for k, v := range headers {
			fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("  %s: %v", k, v), dim))
		}
	}

	// Response body
	if body := data["body"]; body != nil {
		bodyStr := formatJSON(body)
		for _, line := range limitLines(bodyStr, 30) {
			fmt.Fprintf(c.w, "      %s\n", c.styled("  "+line, dim))
		}
	}
}

func (c *Console) printShellDebug(params, data map[string]any, dim string) {
	if stdout, ok := data["stdout"].(string); ok && stdout != "" {
		fmt.Fprintf(c.w, "      %s\n", c.styled("stdout:", dim))
		for _, line := range limitLines(stdout, 20) {
			fmt.Fprintf(c.w, "      %s\n", c.styled("  "+line, dim))
		}
	}
	if stderr, ok := data["stderr"].(string); ok && stderr != "" {
		fmt.Fprintf(c.w, "      %s\n", c.styled("stderr:", dim))
		for _, line := range limitLines(stderr, 10) {
			fmt.Fprintf(c.w, "      %s\n", c.styled("  "+line, dim))
		}
	}
	if code, ok := data["exitCode"].(float64); ok && code != 0 {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("exit code: %d", int(code)), dim))
	}
}

func (c *Console) printDBDebug(prefix string, params, data map[string]any, dim string) {
	// Show the query/sql
	if sql, ok := params["sql"].(string); ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(prefix+" query:", dim))
		for _, line := range limitLines(sql, 10) {
			fmt.Fprintf(c.w, "      %s\n", c.styled("  "+line, dim))
		}
	}
	// Show params
	if p := params["params"]; p != nil {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("  params: %v", p), dim))
	}
	// Show result
	if rows, ok := data["rows"].([]any); ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("← %d row(s)", len(rows)), dim))
		for i, row := range rows {
			if i >= 5 {
				fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("  ... and %d more", len(rows)-5), dim))
				break
			}
			fmt.Fprintf(c.w, "      %s\n", c.styled("  "+formatJSON(row), dim))
		}
	}
	if row, ok := data["row"]; ok && row != nil {
		fmt.Fprintf(c.w, "      %s\n", c.styled("← "+formatJSON(row), dim))
	}
	if count, ok := data["rowsAffected"]; ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("← %v row(s) affected", count), dim))
	}
}

func (c *Console) printRedisDebug(params, data map[string]any, dim string) {
	if key, ok := params["key"].(string); ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("key: %s", key), dim))
	}
	if val, ok := data["value"]; ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("← %v", val), dim))
	}
}

func (c *Console) printEventDebug(params, data map[string]any, dim string) {
	if topic, ok := params["topic"].(string); ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("topic: %s", topic), dim))
	}
	if events, ok := data["events"].([]any); ok {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("← %d event(s)", len(events)), dim))
	}
	if val := data["value"]; val != nil {
		fmt.Fprintf(c.w, "      %s\n", c.styled(fmt.Sprintf("← %v", val), dim))
	}
}

// OnTestComplete prints PASS/FAIL/SKIP with duration.
func (c *Console) OnTestComplete(_ context.Context, test *tryve.TestDefinition, result *tryve.TestResult) error {
	var label string
	switch result.Status {
	case tryve.StatusPassed:
		label = c.styled("PASS", ansiGreen)
	case tryve.StatusFailed:
		label = c.styled("FAIL", ansiRed)
	case tryve.StatusSkipped:
		label = c.styled("SKIP", ansiYellow)
	default:
		label = string(result.Status)
	}

	fmt.Fprintf(c.w, "%s %s (%s)\n", label, test.Name, result.Duration)

	// Always show failure reason, even without --verbose.
	if result.Status == tryve.StatusFailed && !c.verbose {
		if result.Error != nil {
			fmt.Fprintf(c.w, "     %s %v\n", c.styled("→", ansiRed), result.Error)
		}
		for i := len(result.Steps) - 1; i >= 0; i-- {
			step := result.Steps[i]
			if step.Status != tryve.StatusFailed {
				continue
			}
			for _, a := range step.Assertions {
				if !a.Passed {
					fmt.Fprintf(c.w, "     %s [%s] %s %s %v, got %v\n",
						c.styled("→", ansiRed), step.Step.ID, a.Path, a.Operator, a.Expected, a.Actual)
					break
				}
			}
			break
		}
	}

	return nil
}

// OnSuiteComplete prints the summary.
func (c *Console) OnSuiteComplete(_ context.Context, _ *tryve.SuiteResult, result *tryve.SuiteResult) error {
	passed := c.styled(fmt.Sprintf("%d passed", result.Passed), ansiGreen)
	failed := c.styled(fmt.Sprintf("%d failed", result.Failed), ansiRed)
	skipped := c.styled(fmt.Sprintf("%d skipped", result.Skipped), ansiYellow)

	fmt.Fprintf(c.w, "\n%s, %s, %s — %d total (%s)\n",
		passed, failed, skipped, result.Total, result.Duration)
	return nil
}

// Flush is a no-op for the Console reporter.
func (c *Console) Flush() error {
	return nil
}

// stepDescription builds a human-readable label for a step.
func stepDescription(outcome *tryve.StepOutcome) string {
	step := outcome.Step
	if step.Description != "" {
		return step.Description
	}

	prefix := step.Adapter + "." + step.Action

	switch step.Adapter {
	case "http":
		method, _ := step.Params["method"].(string)
		url, _ := step.Params["url"].(string)
		if method == "" {
			method = "GET"
		}
		if url != "" {
			return fmt.Sprintf("%s %s", method, url)
		}
	case "shell":
		if cmd, ok := step.Params["command"].(string); ok {
			if len(cmd) > 60 {
				cmd = cmd[:57] + "..."
			}
			return fmt.Sprintf("$ %s", cmd)
		}
	case "postgresql":
		if sql, ok := step.Params["sql"].(string); ok {
			if len(sql) > 60 {
				sql = sql[:57] + "..."
			}
			return fmt.Sprintf("pg: %s", sql)
		}
	case "mongodb":
		coll, _ := step.Params["collection"].(string)
		if coll != "" {
			return fmt.Sprintf("mongo.%s(%s)", step.Action, coll)
		}
	case "redis":
		key, _ := step.Params["key"].(string)
		if key != "" {
			return fmt.Sprintf("redis.%s %s", step.Action, key)
		}
	case "kafka", "eventhub":
		topic, _ := step.Params["topic"].(string)
		if topic != "" {
			return fmt.Sprintf("%s.%s(%s)", step.Adapter, step.Action, topic)
		}
	}

	return prefix
}

// formatJSON pretty-prints a value as JSON. Falls back to %v on failure.
func formatJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// limitLines splits text into lines and caps at maxLines.
func limitLines(s string, maxLines int) []string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], fmt.Sprintf("... (%d more lines)", len(lines)-maxLines))
	}
	return lines
}
