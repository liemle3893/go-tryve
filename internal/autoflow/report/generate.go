package report

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// Options bundles every input needed to generate the three reports.
// Empty path fields are skipped (rather than failing) so partial runs
// can still produce a best-effort report.
type Options struct {
	// Required.
	Ticket     string
	Branch     string
	PRURL      string
	TicketDir  string // where the three .md files land; usually .autoflow/ticket/<KEY>/
	StateDir   string // state JSONs live here (TicketDir/state or legacy flat)
	BaseBranch string // for the `git diff --name-only origin/<base>...HEAD` fallback

	// Optional.
	SummaryDir string // .planning/quick/<id>/ or impl_plan_dir
	BriefPath  string // task-brief.md
	PRTitle    string // last commit subject, usually

	// E2E tests from the latest run. When nil, Generate reports "no E2E
	// test data available" — useful for partial runs or skipped steps.
	E2E *E2E
}

// Outputs holds the paths of the files Generate writes.
type Outputs struct {
	PRBody          string
	JiraComment     string
	ExecutionReport string
}

// Generate writes the three markdown reports. Returns the paths and any
// error. Non-fatal problems (missing state files, missing summary) are
// folded into WARN-prefixed lines in the report bodies themselves.
func Generate(opts Options) (*Outputs, error) {
	if opts.Ticket == "" || opts.Branch == "" || opts.TicketDir == "" {
		return nil, fmt.Errorf("ticket, branch, and ticketDir are required")
	}
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}
	if err := os.MkdirAll(opts.TicketDir, 0o755); err != nil {
		return nil, err
	}

	state := resolveStateDir(opts)
	coverageFile := filepath.Join(state, "coverage-review-state.json")
	e2eFixFile := filepath.Join(state, "e2e-fix-state.json")
	reviewFile := filepath.Join(state, "enhance-loop-state.json")

	coverage := ParseLoopSummary(coverageFile)
	e2eLoop := ParseLoopSummary(e2eFixFile)
	review := ParseLoopSummary(reviewFile)

	overall := "PASSED"
	switch {
	case e2eLoop.LastStatus == "FAILED",
		review.LastStatus == "ISSUES_FOUND",
		review.LastStatus == "FAILED":
		overall = "FAILED"
	}

	summaryFile := FindSummaryFile(opts.SummaryDir)
	oneLiner := ExtractOneLiner(summaryFile, opts.BriefPath)
	usage := ExtractUsageSection(summaryFile)

	changes, _ := ParseChangesFromSummary(summaryFile)
	if len(changes) == 0 {
		changes = changesFromGitDiff(opts)
	}

	e2e := opts.E2E
	if e2e == nil {
		empty := E2E{}
		e2e = &empty
	}

	data := map[string]any{
		"Ticket":          opts.Ticket,
		"Branch":          opts.Branch,
		"PRURL":           opts.PRURL,
		"PRNumber":        prNumberFromURL(opts.PRURL),
		"PRTitle":         opts.PRTitle,
		"OneLiner":        oneLiner,
		"UsageNotes":      usage,
		"Changes":         changes,
		"E2E":             e2e,
		"OverallStatus":   overall,
		"CoverageRounds":  coverage.Rounds,
		"E2ERounds":       e2eLoop.Rounds,
		"EnhanceRounds":   review.Rounds,
		"TotalIterations": coverage.Rounds + e2eLoop.Rounds + review.Rounds,
		"CoverageFile":    coverageFile,
		"E2EFixFile":      e2eFixFile,
		"ReviewFile":      reviewFile,
	}

	funcs := template.FuncMap{
		"renderChanges":    renderChangesTable,
		"renderE2E":        renderE2ETable,
		"renderLoopTable":  renderLoopTable,
		"renderLoopDetail": renderLoopDetail,
	}

	out := &Outputs{
		PRBody:          filepath.Join(opts.TicketDir, "PR-BODY.md"),
		JiraComment:     filepath.Join(opts.TicketDir, "JIRA-COMMENT.md"),
		ExecutionReport: filepath.Join(opts.TicketDir, "EXECUTION-REPORT.md"),
	}
	if err := render(out.PRBody, prBodyTmpl, funcs, data); err != nil {
		return nil, err
	}
	if err := render(out.JiraComment, jiraCommentTmpl, funcs, data); err != nil {
		return nil, err
	}
	if err := render(out.ExecutionReport, executionReportTmpl, funcs, data); err != nil {
		return nil, err
	}
	return out, nil
}

// resolveStateDir mirrors the bash auto-detection: prefer TicketDir/state,
// fall back to the legacy flat layout when any loop-state JSON is present
// at the ticket root.
func resolveStateDir(opts Options) string {
	if opts.StateDir != "" {
		return opts.StateDir
	}
	subdir := filepath.Join(opts.TicketDir, "state")
	if _, err := os.Stat(subdir); err == nil {
		return subdir
	}
	for _, legacy := range []string{
		"enhance-loop-state.json", "e2e-fix-state.json", "coverage-review-state.json",
	} {
		if _, err := os.Stat(filepath.Join(opts.TicketDir, legacy)); err == nil {
			return opts.TicketDir
		}
	}
	return subdir
}

func render(path, body string, funcs template.FuncMap, data any) error {
	tmpl := template.Must(template.New(filepath.Base(path)).Funcs(funcs).Parse(body))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

// renderChangesTable is the func callers template in for the Changes
// section. Always renders a table shell with a fallback row when the
// slice is empty.
func renderChangesTable(rows []ChangeRow) string {
	var b strings.Builder
	b.WriteString("| File | What Changed |\n")
	b.WriteString("|------|-------------|\n")
	if len(rows) == 0 {
		b.WriteString("| — | No changes detected |\n")
		return b.String()
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "| `%s` | %s |\n", r.Path, r.Action)
	}
	return b.String()
}

// renderE2ETable prints the per-test table + summary line. withDesc
// inserts a Description column (for JIRA-COMMENT / EXECUTION-REPORT).
func renderE2ETable(e E2E, withDesc bool) string {
	var b strings.Builder
	if len(e.Tests) == 0 && e.Warning == "" {
		b.WriteString("No E2E test data available.\n")
		return b.String()
	}
	if len(e.Tests) > 0 {
		if withDesc {
			b.WriteString("| Test | Description | Status | Duration |\n")
			b.WriteString("|------|------------|--------|----------|\n")
			for _, t := range e.Tests {
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", t.ID, t.Desc, t.Status, t.Duration)
			}
		} else {
			b.WriteString("| Test | Status | Duration |\n")
			b.WriteString("|------|--------|----------|\n")
			for _, t := range e.Tests {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", t.ID, t.Status, t.Duration)
			}
		}
		b.WriteByte('\n')
		line := fmt.Sprintf("**Summary:** %d/%d passed", e.Passed, e.Total)
		if e.Duration != "" && e.Duration != "—" {
			line += " in " + e.Duration
		}
		fmt.Fprintln(&b, line)
	}
	if e.Warning != "" {
		fmt.Fprintf(&b, "\n> WARNING: %s\n", e.Warning)
	}
	return b.String()
}

// renderLoopTable is the condensed per-round view used by JIRA-COMMENT.
func renderLoopTable(path, label string) string {
	summary := ParseLoopSummary(path)
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s (%d rounds)\n\n", label, summary.LastStatus, summary.Rounds)
	if !summary.Present || summary.LastStatus == "SKIPPED" {
		b.WriteString("| Round | Status | Issues | Fixes |\n")
		b.WriteString("|-------|--------|--------|-------|\n")
		b.WriteString("| — | SKIPPED | — | — |\n\n")
		return b.String()
	}
	rounds, _ := ReadRounds(path)
	b.WriteString("| Round | Status | Issues | Fixes |\n")
	b.WriteString("|-------|--------|--------|-------|\n")
	for _, r := range rounds {
		issue := firstProblemSummary(r.Problems)
		fix := firstFixSummary(r.Fixes)
		fmt.Fprintf(&b, "| %d | %s | %s | %s |\n", r.Round, r.Status, issue, fix)
	}
	b.WriteByte('\n')
	return b.String()
}

// renderLoopDetail is the round-by-round view used by EXECUTION-REPORT.
// One subsection per round with bulleted problems and fixes.
func renderLoopDetail(path, label string) string {
	summary := ParseLoopSummary(path)
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s (%d rounds)\n\n", label, summary.LastStatus, summary.Rounds)
	if !summary.Present || summary.LastStatus == "SKIPPED" {
		b.WriteString("SKIPPED — no state file found.\n\n")
		return b.String()
	}
	rounds, _ := ReadRounds(path)
	for _, r := range rounds {
		fmt.Fprintf(&b, "#### Round %d — %s\n", r.Round, r.Status)
		if len(r.Problems) > 0 {
			b.WriteString("**Problems:**\n")
			for _, p := range r.Problems {
				b.WriteString("- ")
				b.WriteString(formatProblem(p))
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
		if len(r.Fixes) > 0 {
			b.WriteString("**Fixes:**\n")
			for _, f := range r.Fixes {
				b.WriteString("- ")
				b.WriteString(formatFix(f))
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
		if len(r.Problems) == 0 && len(r.Fixes) == 0 {
			b.WriteString("No issues found.\n\n")
		}
	}
	return b.String()
}

func firstProblemSummary(problems []GenericEntry) string {
	if len(problems) == 0 {
		return "—"
	}
	text := problems[0].Description
	if text == "" && problems[0].AC != "" {
		text = "AC #" + problems[0].AC
	}
	if text == "" {
		text = "issue found"
	}
	if len(problems) > 1 {
		text = fmt.Sprintf("%s (+%d more)", text, len(problems)-1)
	}
	if len(text) > 80 {
		text = text[:77] + "..."
	}
	return text
}

func firstFixSummary(fixes []GenericEntry) string {
	if len(fixes) == 0 {
		return "—"
	}
	text := fixes[0].Action
	if text == "" && fixes[0].File != "" {
		text = fixes[0].File
	}
	if text == "" {
		text = "fixed"
	}
	if len(fixes) > 1 {
		text = fmt.Sprintf("%s (+%d more)", text, len(fixes)-1)
	}
	if len(text) > 80 {
		text = text[:77] + "..."
	}
	return text
}

func formatProblem(p GenericEntry) string {
	var parts []string
	if p.Severity != "" && p.Category != "" {
		parts = append(parts, fmt.Sprintf("[%s/%s]", p.Severity, p.Category))
	}
	if p.AC != "" {
		parts = append(parts, "AC #"+p.AC+":")
	}
	loc := ""
	if p.File != "" {
		loc = "`" + p.File
		if lineStr := genericLine(p.Line); lineStr != "" {
			loc += ":" + lineStr
		}
		loc += "` — "
	}
	prefix := strings.Join(parts, " ")
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += " "
	}
	return prefix + loc + p.Description
}

func formatFix(f GenericEntry) string {
	out := "`" + f.File + "` — " + f.Action
	if f.Commit != "" {
		out += " (" + f.Commit + ")"
	}
	return out
}

func genericLine(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%d", int(x))
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return ""
	}
}

// changesFromGitDiff falls back on `git diff --name-only origin/<base>...HEAD`
// when SUMMARY.md lacks created:/modified: sections. Best-effort — errors
// produce an empty slice (the PR body then renders "No changes detected").
func changesFromGitDiff(opts Options) []ChangeRow {
	cmd := exec.Command("git", "diff", "--name-only",
		fmt.Sprintf("origin/%s...HEAD", opts.BaseBranch))
	cmd.Dir = filepath.Dir(opts.TicketDir)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var rows []ChangeRow
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rows = append(rows, ChangeRow{Path: line, Action: "Modified"})
	}
	return rows
}

func prNumberFromURL(url string) string {
	if url == "" {
		return ""
	}
	// Pull request URLs end in /NNN; extract the trailing digit run.
	i := len(url) - 1
	for i >= 0 && url[i] >= '0' && url[i] <= '9' {
		i--
	}
	if i == len(url)-1 {
		return ""
	}
	return url[i+1:]
}
