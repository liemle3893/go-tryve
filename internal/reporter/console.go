package reporter

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// ANSI escape codes used by the Console reporter.
const (
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiReset  = "\033[0m"
)

// Console is a Reporter implementation that writes human-readable, optionally
// colourised output to an io.Writer (usually os.Stdout).
type Console struct {
	w       io.Writer
	verbose bool
	color   bool
}

// NewConsole creates a Console reporter writing to w. verbose enables per-step
// output; color enables ANSI escape codes.
func NewConsole(w io.Writer, verbose, color bool) *Console {
	return &Console{w: w, verbose: verbose, color: color}
}

// NewConsoleFromEnv creates a Console reporter writing to os.Stdout, auto-detecting
// colour support via the NO_COLOR environment variable (https://no-color.org/).
func NewConsoleFromEnv(verbose bool) *Console {
	color := os.Getenv("NO_COLOR") == ""
	return NewConsole(os.Stdout, verbose, color)
}

// styled wraps text in the given ANSI style code followed by a reset, but only
// when colour output is enabled.
func (c *Console) styled(text, style string) string {
	if !c.color {
		return text
	}
	return style + text + ansiReset
}

// OnSuiteStart prints the "Tryve Test Runner" header.
func (c *Console) OnSuiteStart(_ context.Context, _ *tryve.SuiteResult) error {
	fmt.Fprintln(c.w, c.styled("Tryve Test Runner", ansiBold))
	return nil
}

// OnTestStart prints a "RUN {name}" line when verbose output is enabled.
func (c *Console) OnTestStart(_ context.Context, test *tryve.TestDefinition) error {
	if c.verbose {
		fmt.Fprintf(c.w, "%s\n", c.styled("RUN "+test.Name, ansiCyan))
	}
	return nil
}

// OnStepComplete prints step results in verbose mode, including failed assertions.
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

	desc := outcome.Step.Description
	if desc == "" {
		desc = outcome.Step.Action
	}

	fmt.Fprintf(c.w, "  %s %s (%s)\n", marker, desc, outcome.Duration)

	// Print details for any failed assertions.
	for _, a := range outcome.Assertions {
		if !a.Passed {
			fmt.Fprintf(c.w, "      %s: expected %v, got %v\n",
				c.styled("ASSERT FAIL", ansiRed), a.Expected, a.Actual)
			if a.Message != "" {
				fmt.Fprintf(c.w, "        %s\n", a.Message)
			}
		}
	}

	return nil
}

// OnTestComplete prints a PASS/FAIL/SKIP line for the test with its duration.
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
		// Show first failed assertion from the last failed step.
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

// OnSuiteComplete prints a summary line showing pass/fail/skip counts and total duration.
func (c *Console) OnSuiteComplete(_ context.Context, _ *tryve.SuiteResult, result *tryve.SuiteResult) error {
	passed := c.styled(fmt.Sprintf("%d passed", result.Passed), ansiGreen)
	failed := c.styled(fmt.Sprintf("%d failed", result.Failed), ansiRed)
	skipped := c.styled(fmt.Sprintf("%d skipped", result.Skipped), ansiYellow)

	fmt.Fprintf(c.w, "\n%s, %s, %s — %d total (%s)\n",
		passed, failed, skipped, result.Total, result.Duration)
	return nil
}

// Flush is a no-op for the Console reporter; all output is written synchronously.
func (c *Console) Flush() error {
	return nil
}
