package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const sampleReviewCode = `---
status: ISSUES_FOUND
findings:
  critical: 1
  warning: 1
---

# Code Review

## Summary

Some intro text.

### CR-01: Missing nil guard in handler
File: internal/x/y.go

### WR-02: Shadowed variable
File: internal/x/z.go

### IN-03: Name could be clearer
File: internal/x/y.go
`

const sampleReviewFix = `# Fix Report

## Fixed

### CR-01: Missing nil guard in handler
Added check before dereference.

## Skipped

### IN-03: Name could be clearer
Non-blocking; left for follow-up.
`

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExtract_HappyPath(t *testing.T) {
	dir := t.TempDir()
	codePath := writeFile(t, dir, "REVIEW-code.md", sampleReviewCode)

	rd, err := Extract(Inputs{ReviewCode: codePath})
	if err != nil {
		t.Fatal(err)
	}
	// 1 critical + 1 warning = 2 bugs; 1 info = 1 design-concern.
	if rd.BugsFound != 2 {
		t.Errorf("want bugs_found=2, got %d", rd.BugsFound)
	}
	if rd.DesignConcernsFound != 1 {
		t.Errorf("want design_concerns_found=1, got %d", rd.DesignConcernsFound)
	}
	// Problems contains bugs only.
	if len(rd.Problems) != 2 {
		t.Errorf("want 2 problems (bugs), got %d", len(rd.Problems))
	}
	// Feedback IDs covers ALL findings.
	if len(rd.FeedbackIDs) != 3 {
		t.Errorf("want 3 feedback ids, got %d", len(rd.FeedbackIDs))
	}
	// All findings start in pending.
	for _, f := range rd.Findings {
		if f.Disposition != "pending" {
			t.Errorf("finding %s: want pending, got %s", f.ID, f.Disposition)
		}
	}
}

func TestExtract_WithFixDispositions(t *testing.T) {
	dir := t.TempDir()
	codePath := writeFile(t, dir, "REVIEW-code.md", sampleReviewCode)
	fixPath := writeFile(t, dir, "REVIEW-FIX.md", sampleReviewFix)

	rd, err := Extract(Inputs{ReviewCode: codePath, ReviewFix: fixPath})
	if err != nil {
		t.Fatal(err)
	}

	disp := map[string]string{}
	for _, f := range rd.Findings {
		disp[f.ID] = f.Disposition
	}
	if disp["CR-01"] != "fixed" {
		t.Errorf("CR-01: want fixed, got %q", disp["CR-01"])
	}
	if disp["IN-03"] != "skipped" {
		t.Errorf("IN-03: want skipped, got %q", disp["IN-03"])
	}
	// WR-02 was not mentioned in REVIEW-FIX.md → still pending.
	if disp["WR-02"] != "pending" {
		t.Errorf("WR-02: want pending, got %q", disp["WR-02"])
	}
	// Fixes slice contains only the fixed one.
	if len(rd.Fixes) != 1 || rd.Fixes[0].ID != "CR-01" {
		t.Errorf("fixes: %+v", rd.Fixes)
	}
}

func TestExtract_CleanStatusEmits0(t *testing.T) {
	dir := t.TempDir()
	body := `---
status: clean
---
No findings.
`
	codePath := writeFile(t, dir, "REVIEW-code.md", body)
	rd, _ := Extract(Inputs{ReviewCode: codePath})
	if rd.BugsFound != 0 || rd.DesignConcernsFound != 0 || len(rd.Findings) != 0 {
		t.Errorf("clean status must emit 0 findings, got %+v", rd)
	}
}

func TestExtract_MissingFilesOK(t *testing.T) {
	rd, err := Extract(Inputs{ReviewCode: "/no/such/file.md"})
	if err != nil {
		t.Fatal(err)
	}
	if rd.BugsFound != 0 || len(rd.Findings) != 0 {
		t.Errorf("missing file should produce empty, got %+v", rd)
	}
}

func TestExtract_SeverityPrefixMap(t *testing.T) {
	dir := t.TempDir()
	body := `---
status: ISSUES_FOUND
---
### RLC-01: Rules critical
### SMW-02: Simplify warning
### SMI-03: Simplify info
`
	p := writeFile(t, dir, "REVIEW-rules.md", body)
	rd, _ := Extract(Inputs{ReviewRules: p})
	severities := map[string]string{}
	for _, f := range rd.Findings {
		severities[f.ID] = f.Severity
	}
	if severities["RLC-01"] != "critical" {
		t.Errorf("RLC → critical, got %q", severities["RLC-01"])
	}
	if severities["SMW-02"] != "warning" {
		t.Errorf("SMW → warning, got %q", severities["SMW-02"])
	}
	if severities["SMI-03"] != "info" {
		t.Errorf("SMI → info, got %q", severities["SMI-03"])
	}
}

func TestWriteFiles(t *testing.T) {
	dir := t.TempDir()
	codePath := writeFile(t, dir, "REVIEW-code.md", sampleReviewCode)
	rd, _ := Extract(Inputs{ReviewCode: codePath})

	rdOut := filepath.Join(dir, "round-data.json")
	fbOut := filepath.Join(dir, "feedback.json")
	if err := WriteFiles(rd, rdOut, fbOut); err != nil {
		t.Fatal(err)
	}

	var parsed struct {
		BugsFound int `json:"bugs_found"`
	}
	raw, _ := os.ReadFile(rdOut)
	_ = json.Unmarshal(raw, &parsed)
	if parsed.BugsFound != 2 {
		t.Errorf("round-data output lost bugs_found, got %d", parsed.BugsFound)
	}

	var feedback []Finding
	raw, _ = os.ReadFile(fbOut)
	_ = json.Unmarshal(raw, &feedback)
	if len(feedback) != 3 {
		t.Errorf("feedback output has %d entries, want 3", len(feedback))
	}
}
