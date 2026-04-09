package reporter

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"sync"
	"time"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// reportEntry pairs a test definition with its final result for template rendering.
type reportEntry struct {
	Test   *tryve.TestDefinition
	Result *tryve.TestResult
}

// HTML is a Reporter implementation that accumulates test results during a run
// and writes a self-contained HTML report file on Flush.
type HTML struct {
	mu         sync.Mutex
	outputPath string
	entries    []reportEntry
	suiteStart time.Time
	suiteEnd   time.Time
}

// NewHTML creates an HTML reporter that will write its output to outputPath on Flush.
func NewHTML(outputPath string) *HTML {
	return &HTML{outputPath: outputPath}
}

// OnSuiteStart records the suite start time.
func (h *HTML) OnSuiteStart(_ context.Context, _ *tryve.SuiteResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.suiteStart = time.Now()
	return nil
}

// OnTestStart is a no-op for the HTML reporter; results are captured on completion.
func (h *HTML) OnTestStart(_ context.Context, _ *tryve.TestDefinition) error {
	return nil
}

// OnStepComplete is a no-op for the HTML reporter; step data arrives via OnTestComplete.
func (h *HTML) OnStepComplete(_ context.Context, _ *tryve.StepDefinition, _ *tryve.StepOutcome) error {
	return nil
}

// OnTestComplete appends the completed test and its result to the internal accumulator.
func (h *HTML) OnTestComplete(_ context.Context, test *tryve.TestDefinition, result *tryve.TestResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, reportEntry{Test: test, Result: result})
	return nil
}

// OnSuiteComplete records the suite end time.
func (h *HTML) OnSuiteComplete(_ context.Context, _ *tryve.SuiteResult, _ *tryve.SuiteResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.suiteEnd = time.Now()
	return nil
}

// Flush renders the accumulated results into a self-contained HTML file at outputPath.
// It returns an error if the file cannot be created or the template fails to execute.
func (h *HTML) Flush() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	passed, failed, skipped := h.counts()

	data := struct {
		GeneratedAt string
		Passed      int
		Failed      int
		Skipped     int
		Total       int
		Duration    string
		Entries     []reportEntry
	}{
		GeneratedAt: time.Now().Format(time.RFC1123),
		Passed:      passed,
		Failed:      failed,
		Skipped:     skipped,
		Total:       passed + failed + skipped,
		Duration:    h.suiteDuration(),
		Entries:     h.entries,
	}

	tmpl, err := template.New("report").Funcs(templateFuncs()).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("html reporter: parse template: %w", err)
	}

	f, err := os.Create(h.outputPath)
	if err != nil {
		return fmt.Errorf("html reporter: create output file %q: %w", h.outputPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("html reporter: execute template: %w", err)
	}

	return nil
}

// counts returns the pass/fail/skip breakdown from the accumulated entries.
func (h *HTML) counts() (passed, failed, skipped int) {
	for _, e := range h.entries {
		switch e.Result.Status {
		case tryve.StatusPassed:
			passed++
		case tryve.StatusFailed:
			failed++
		case tryve.StatusSkipped:
			skipped++
		}
	}
	return
}

// suiteDuration returns a human-readable total suite duration.
func (h *HTML) suiteDuration() string {
	if h.suiteEnd.IsZero() || h.suiteStart.IsZero() {
		return "—"
	}
	return h.suiteEnd.Sub(h.suiteStart).Round(time.Millisecond).String()
}

// templateFuncs returns the template.FuncMap used when rendering the report.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// statusClass maps a TestStatus to a CSS class name.
		"statusClass": func(s tryve.TestStatus) string {
			switch s {
			case tryve.StatusPassed:
				return "pass"
			case tryve.StatusFailed:
				return "fail"
			case tryve.StatusSkipped:
				return "skip"
			default:
				return "skip"
			}
		},
		// statusLabel returns an uppercase display label for a TestStatus.
		"statusLabel": func(s tryve.TestStatus) string {
			switch s {
			case tryve.StatusPassed:
				return "PASS"
			case tryve.StatusFailed:
				return "FAIL"
			case tryve.StatusSkipped:
				return "SKIP"
			default:
				return string(s)
			}
		},
		// fmtDuration formats a time.Duration for display.
		"fmtDuration": func(d time.Duration) string {
			return d.Round(time.Millisecond).String()
		},
	}
}

// htmlTemplate is the self-contained Go template used to render the HTML report.
// All styles are inlined; no external assets are referenced.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Tryve Test Report</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: #f5f7fa;
      color: #1a1a2e;
      font-size: 14px;
      line-height: 1.6;
    }
    header {
      background: #1a1a2e;
      color: #fff;
      padding: 20px 32px;
    }
    header h1 { font-size: 22px; font-weight: 700; letter-spacing: 0.5px; }
    header .meta { font-size: 12px; color: #a0aec0; margin-top: 4px; }
    .container { max-width: 960px; margin: 24px auto; padding: 0 16px; }
    .summary {
      display: flex;
      gap: 12px;
      margin-bottom: 24px;
      flex-wrap: wrap;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 16px;
      border-radius: 6px;
      font-weight: 600;
      font-size: 15px;
    }
    .badge.pass  { background: #d4edda; color: #155724; }
    .badge.fail  { background: #f8d7da; color: #721c24; }
    .badge.skip  { background: #fff3cd; color: #856404; }
    .badge.total { background: #e9ecef; color: #343a40; }
    .badge .count { font-size: 22px; font-weight: 800; }
    .tests { list-style: none; display: flex; flex-direction: column; gap: 10px; }
    details {
      background: #fff;
      border: 1px solid #dee2e6;
      border-radius: 8px;
      overflow: hidden;
    }
    details[open] { box-shadow: 0 2px 8px rgba(0,0,0,.08); }
    summary {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 12px 16px;
      cursor: pointer;
      list-style: none;
      user-select: none;
    }
    summary::-webkit-details-marker { display: none; }
    summary .arrow {
      width: 18px;
      text-align: center;
      transition: transform 0.2s;
      color: #6c757d;
      font-size: 12px;
    }
    details[open] summary .arrow { transform: rotate(90deg); }
    .status-badge {
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      min-width: 40px;
      text-align: center;
    }
    .status-badge.pass  { background: #d4edda; color: #155724; }
    .status-badge.fail  { background: #f8d7da; color: #721c24; }
    .status-badge.skip  { background: #fff3cd; color: #856404; }
    .test-name { flex: 1; font-weight: 600; }
    .test-duration { color: #6c757d; font-size: 12px; }
    .details-body { padding: 0 16px 16px; border-top: 1px solid #e9ecef; }
    .steps { list-style: none; margin-top: 12px; display: flex; flex-direction: column; gap: 6px; }
    .step {
      padding: 8px 12px;
      border-radius: 6px;
      background: #f8f9fa;
      border-left: 4px solid #dee2e6;
    }
    .step.pass { border-left-color: #28a745; }
    .step.fail { border-left-color: #dc3545; }
    .step.skip { border-left-color: #ffc107; }
    .step-header { display: flex; align-items: center; gap: 8px; }
    .step-action { font-family: monospace; font-size: 12px; background: #e9ecef; padding: 1px 5px; border-radius: 3px; }
    .step-desc { flex: 1; }
    .step-duration { color: #6c757d; font-size: 11px; }
    .assertions { margin-top: 8px; }
    .assertion-fail {
      background: #fff5f5;
      border: 1px solid #f5c6cb;
      border-radius: 4px;
      padding: 6px 10px;
      margin-top: 4px;
      font-size: 12px;
    }
    .assertion-fail .label { font-weight: 700; color: #721c24; }
    .assertion-fail table { border-collapse: collapse; width: 100%; margin-top: 4px; }
    .assertion-fail td { padding: 2px 6px; vertical-align: top; }
    .assertion-fail td:first-child { color: #6c757d; width: 80px; font-weight: 600; }
    .assertion-fail code { font-family: monospace; font-size: 11px; word-break: break-all; }
    footer {
      text-align: center;
      color: #adb5bd;
      font-size: 12px;
      padding: 24px 16px;
    }
  </style>
</head>
<body>
  <header>
    <h1>Tryve Test Report</h1>
    <div class="meta">Generated: {{.GeneratedAt}} &nbsp;|&nbsp; Duration: {{.Duration}}</div>
  </header>

  <div class="container">
    <div class="summary">
      <div class="badge total"><span class="count">{{.Total}}</span> total</div>
      <div class="badge pass"><span class="count">{{.Passed}}</span> passed</div>
      <div class="badge fail"><span class="count">{{.Failed}}</span> failed</div>
      <div class="badge skip"><span class="count">{{.Skipped}}</span> skipped</div>
    </div>

    <ul class="tests">
      {{range .Entries}}
      <li>
        <details {{if eq .Result.Status "failed"}}open{{end}}>
          <summary>
            <span class="arrow">&#9654;</span>
            <span class="status-badge {{statusClass .Result.Status}}">{{statusLabel .Result.Status}}</span>
            <span class="test-name">{{.Test.Name}}</span>
            <span class="test-duration">{{fmtDuration .Result.Duration}}</span>
          </summary>
          <div class="details-body">
            {{if .Test.Description}}<p style="color:#6c757d;margin-bottom:8px;">{{.Test.Description}}</p>{{end}}
            <ul class="steps">
              {{range .Result.Steps}}
              <li class="step {{statusClass .Status}}">
                <div class="step-header">
                  <span class="status-badge {{statusClass .Status}}">{{statusLabel .Status}}</span>
                  {{if .Step.Description}}
                  <span class="step-desc">{{.Step.Description}}</span>
                  {{else}}
                  <span class="step-desc"><span class="step-action">{{.Step.Action}}</span></span>
                  {{end}}
                  <span class="step-duration">{{fmtDuration .Duration}}</span>
                </div>
                {{$failedAssertions := .Assertions}}
                {{range $failedAssertions}}
                {{if not .Passed}}
                <div class="assertion-fail">
                  <div class="label">Assertion failed</div>
                  <table>
                    <tr><td>path</td><td><code>{{.Path}}</code></td></tr>
                    <tr><td>operator</td><td><code>{{.Operator}}</code></td></tr>
                    <tr><td>expected</td><td><code>{{.Expected}}</code></td></tr>
                    <tr><td>actual</td><td><code>{{.Actual}}</code></td></tr>
                    {{if .Message}}<tr><td>message</td><td>{{.Message}}</td></tr>{{end}}
                  </table>
                </div>
                {{end}}
                {{end}}
              </li>
              {{end}}
            </ul>
          </div>
        </details>
      </li>
      {{end}}
    </ul>
  </div>

  <footer>Tryve Test Report</footer>
</body>
</html>`
