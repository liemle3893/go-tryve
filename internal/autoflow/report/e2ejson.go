package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// TestRow is one line of the E2E results table.
type TestRow struct {
	ID       string
	Desc     string // optional — filled from the test file when available
	Status   string // PASSED | FAILED | SKIPPED
	Duration string // human form — "1.23s" or "—"
}

// E2E summarises a run for the report section.
type E2E struct {
	Tests    []TestRow
	Passed   int
	Failed   int
	Skipped  int
	Total    int
	Duration string
	Warning  string // non-empty when a sanity check flagged the run
}

// FromSuite converts a structured tryve SuiteResult into the report shape.
// Replaces the regex-based parser in generate-report.sh — no more grepping
// "Test NAME: passed (NNNms)" out of console logs.
func FromSuite(s *tryve.SuiteResult) E2E {
	if s == nil {
		return E2E{}
	}
	out := E2E{
		Passed:   s.Passed,
		Failed:   s.Failed,
		Skipped:  s.Skipped,
		Total:    s.Total,
		Duration: humanDuration(s.Duration),
	}
	for _, t := range s.Tests {
		if t.Test == nil {
			continue
		}
		row := TestRow{
			ID:       t.Test.Name,
			Status:   strings.ToUpper(string(t.Status)),
			Duration: humanDuration(t.Duration),
		}
		if d := strings.TrimSpace(t.Test.Description); d != "" {
			if len(d) > 80 {
				d = d[:77] + "..."
			}
			row.Desc = d
		}
		out.Tests = append(out.Tests, row)
	}
	if len(out.Tests) == 0 && out.Total == 0 {
		out.Warning = "No tests found matching the specified criteria — 0 tests ran"
	}
	return out
}

// humanDuration prints "N.NNs" for >=1s and "Nms" otherwise. Empty
// duration yields "—" so the table still aligns.
func humanDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d >= time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}
