package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeStateFile(t *testing.T, path string, body any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(body)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_MinimalInputs(t *testing.T) {
	ticketDir := t.TempDir()
	opts := Options{
		Ticket:     "PROJ-1",
		Branch:     "jira-iss/proj-1-foo",
		PRURL:      "https://github.com/org/repo/pull/42",
		TicketDir:  ticketDir,
		BaseBranch: "main",
	}
	out, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	// All three files must exist.
	for _, path := range []string{out.PRBody, out.JiraComment, out.ExecutionReport} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing output: %s: %v", path, err)
		}
	}
	// PR body does not include the ticket key (parity with
	// generate-report.sh PR-BODY section).
	prBody, _ := os.ReadFile(out.PRBody)
	if !strings.Contains(string(prBody), "See PR for details") {
		t.Errorf("PR body missing default one-liner, got:\n%s", prBody)
	}
	// Jira comment + execution report must identify the ticket.
	for _, path := range []string{out.JiraComment, out.ExecutionReport} {
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "PROJ-1") {
			t.Errorf("%s did not embed ticket key", path)
		}
	}
}

func TestGenerate_WithLoops(t *testing.T) {
	ticketDir := t.TempDir()
	stateDir := filepath.Join(ticketDir, "state")

	writeStateFile(t, filepath.Join(stateDir, "coverage-review-state.json"), map[string]any{
		"loop": "coverage-review", "ticket": "PROJ-1", "max_rounds": 3,
		"rounds": []map[string]any{
			{"round": 1, "timestamp": "t", "status": "GAPS_FOUND", "problems": []map[string]any{
				{"description": "missing AC for logout"},
			}, "fixes": []map[string]any{
				{"action": "added test", "file": "tests/e2e/x.test.yaml"},
			}},
			{"round": 2, "timestamp": "t", "status": "PASS"},
		},
	})

	writeStateFile(t, filepath.Join(stateDir, "e2e-fix-state.json"), map[string]any{
		"loop": "e2e-fix", "ticket": "PROJ-1", "max_rounds": 5,
		"rounds": []map[string]any{
			{"round": 1, "timestamp": "t", "status": "PASSED"},
		},
	})

	e2e := &E2E{
		Tests: []TestRow{
			{ID: "TC-PROJ-1-001", Desc: "signup happy path", Status: "PASSED", Duration: "1.10s"},
		},
		Passed: 1, Total: 1, Duration: "1.10s",
	}

	out, err := Generate(Options{
		Ticket:     "PROJ-1",
		Branch:     "jira-iss/proj-1-foo",
		PRURL:      "https://github.com/org/repo/pull/42",
		TicketDir:  ticketDir,
		StateDir:   stateDir,
		BaseBranch: "main",
		E2E:        e2e,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := os.ReadFile(out.JiraComment)
	s := string(body)
	if !strings.Contains(s, "AC Coverage") {
		t.Errorf("missing AC Coverage heading")
	}
	if !strings.Contains(s, "TC-PROJ-1-001") {
		t.Errorf("missing test row")
	}
	if !strings.Contains(s, "2 rounds") {
		t.Errorf("missing round count")
	}
}

func TestResolveStateDir(t *testing.T) {
	// New layout — TicketDir/state exists.
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "state"), 0o755)
	if got := resolveStateDir(Options{TicketDir: dir}); got != filepath.Join(dir, "state") {
		t.Errorf("new layout: got %q", got)
	}

	// Legacy flat layout — state file sits at the root.
	dir2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir2, "coverage-review-state.json"), []byte(`{}`), 0o644)
	if got := resolveStateDir(Options{TicketDir: dir2}); got != dir2 {
		t.Errorf("legacy layout: got %q, want %q", got, dir2)
	}

	// Neither — defaults to subdir.
	dir3 := t.TempDir()
	if got := resolveStateDir(Options{TicketDir: dir3}); got != filepath.Join(dir3, "state") {
		t.Errorf("empty dir default: got %q", got)
	}
}

func TestParseLoopSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.json")
	_ = os.WriteFile(path, []byte(`{"rounds":[{"status":"PASSED"},{"status":"FAILED"}]}`), 0o644)
	s := ParseLoopSummary(path)
	if s.Rounds != 2 {
		t.Errorf("want 2 rounds, got %d", s.Rounds)
	}
	if s.LastStatus != "FAILED" {
		t.Errorf("want FAILED, got %q", s.LastStatus)
	}
	// Missing file.
	s = ParseLoopSummary(filepath.Join(t.TempDir(), "missing.json"))
	if s.LastStatus != "SKIPPED" {
		t.Errorf("want SKIPPED, got %q", s.LastStatus)
	}
}

func TestPRNumberFromURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/org/repo/pull/42":     "42",
		"https://github.com/org/repo/pull/1234/":  "",
		"https://github.com/org/repo/pull/1234#x": "",
		"":                                        "",
	}
	for in, want := range cases {
		if got := prNumberFromURL(in); got != want {
			t.Errorf("prNumberFromURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParseChangesFromSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	body := `# Title

changes:
  created:
    - internal/foo/new.go
    - internal/foo/new_test.go
  modified:
    - cmd/main.go
  deleted:
    - []

next:
  ok: true
`
	_ = os.WriteFile(path, []byte(body), 0o644)
	rows, err := ParseChangesFromSummary(path)
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for _, r := range rows {
		count[r.Action]++
	}
	if count["Created"] != 2 || count["Modified"] != 1 {
		t.Errorf("unexpected counts: %+v (rows=%v)", count, rows)
	}
}

func TestExtractOneLiner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	_ = os.WriteFile(path, []byte(`# Title

## Summary

A tiny one-line description.

## Changes
etc
`), 0o644)
	got := ExtractOneLiner(path, "")
	if got != "A tiny one-line description." {
		t.Errorf("one-liner: got %q", got)
	}
}

func TestFindSummaryFile(t *testing.T) {
	dir := t.TempDir()
	// Direct SUMMARY.md wins.
	_ = os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(""), 0o644)
	if got := FindSummaryFile(dir); got != filepath.Join(dir, "SUMMARY.md") {
		t.Errorf("SUMMARY.md preferred, got %q", got)
	}
	// Fallback to *-SUMMARY.md.
	dir2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir2, "PROJ-1-SUMMARY.md"), []byte(""), 0o644)
	if got := FindSummaryFile(dir2); got != filepath.Join(dir2, "PROJ-1-SUMMARY.md") {
		t.Errorf("fallback summary lookup, got %q", got)
	}
}
