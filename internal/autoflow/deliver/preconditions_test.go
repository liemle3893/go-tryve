package deliver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// writeFile is a small helper that creates parent dirs + writes content.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyStepComplete_NoopForSkippedSteps(t *testing.T) {
	// Steps 3, 8, 13 have no preconditions.
	root := t.TempDir()
	for _, step := range []int{3, 8, 13} {
		if err := VerifyStepComplete(root, "PROJ-1", step, nil); err != nil {
			t.Errorf("step %d must be a no-op, got %v", step, err)
		}
	}
}

func TestStep1_Precondition(t *testing.T) {
	root := t.TempDir()
	// No brief → reject.
	if err := VerifyStepComplete(root, "PROJ-1", 1, nil); err == nil {
		t.Fatal("expected error when brief missing")
	} else if !strings.Contains(err.Error(), "task-brief.md") {
		t.Errorf("error should mention file, got %v", err)
	}
	// Seed brief → pass.
	writeFile(t, filepath.Join(state.TicketDir(root, "PROJ-1"), "task-brief.md"), "")
	if err := VerifyStepComplete(root, "PROJ-1", 1, nil); err != nil {
		t.Errorf("with brief, should pass: %v", err)
	}
}

func TestStep2_Preconditions(t *testing.T) {
	root := t.TempDir()
	wt := t.TempDir()

	// Progress missing + worktree unset → reject with progress error.
	if err := VerifyStepComplete(root, "PROJ-1", 2, nil); err == nil {
		t.Fatal("expected error when progress missing")
	}

	// Progress set but worktree path missing → reject.
	_, _ = state.InitProgress(root, "PROJ-1", "/does/not/exist", "b", false)
	p, _ := state.ReadProgress(root, "PROJ-1")
	if err := VerifyStepComplete(root, "PROJ-1", 2, p); err == nil {
		t.Fatal("expected error when worktree dir missing")
	}

	// Progress set + worktree exists → pass.
	_, _ = state.InitProgress(root, "PROJ-1", wt, "b", true)
	p, _ = state.ReadProgress(root, "PROJ-1")
	if err := VerifyStepComplete(root, "PROJ-1", 2, p); err != nil {
		t.Errorf("should pass: %v", err)
	}
}

func TestStep4_RejectsUnlessPass(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(state.TicketStateDir(root, "PROJ-1"), "coverage-review-state.json")
	// Missing file.
	if err := VerifyStepComplete(root, "PROJ-1", 4, nil); err == nil {
		t.Fatal("expected error when file missing")
	}
	// Last round not PASS.
	writeFile(t, path, `{"rounds":[{"status":"GAPS_FOUND"}]}`)
	if err := VerifyStepComplete(root, "PROJ-1", 4, nil); err == nil {
		t.Fatal("expected error when last status != PASS")
	}
	// Last round PASS.
	writeFile(t, path, `{"rounds":[{"status":"GAPS_FOUND"},{"status":"PASS"}]}`)
	if err := VerifyStepComplete(root, "PROJ-1", 4, nil); err != nil {
		t.Errorf("should pass: %v", err)
	}
}

func TestStep5_RequiresPlanOrSummary(t *testing.T) {
	root := t.TempDir()
	if err := VerifyStepComplete(root, "PROJ-1", 5, nil); err == nil {
		t.Fatal("expected error when neither present")
	}
	writeFile(t, filepath.Join(state.TicketDir(root, "PROJ-1"), "SUMMARY.md"), "ok")
	if err := VerifyStepComplete(root, "PROJ-1", 5, nil); err != nil {
		t.Errorf("should pass with SUMMARY.md: %v", err)
	}
}

func TestStep6_SkipsWhenNoBuildCmd(t *testing.T) {
	root := t.TempDir()
	// No bootstrap.json → no build cmds configured → step 6 auto-passes.
	if err := VerifyStepComplete(root, "PROJ-1", 6, nil); err != nil {
		t.Errorf("should pass when no build cmds: %v", err)
	}
}

func TestStep6_RejectsWhenGateFailed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".autoflow", "bootstrap.json"),
		`{"build_cmd":"make","test_cmd":"go test ./..."}`)
	path := filepath.Join(state.TicketStateDir(root, "PROJ-1"), "build-gate-state.json")
	writeFile(t, path, `{"last_result":"fail"}`)
	if err := VerifyStepComplete(root, "PROJ-1", 6, nil); err == nil {
		t.Fatal("expected error when gate failed")
	}
	writeFile(t, path, `{"last_result":"pass"}`)
	if err := VerifyStepComplete(root, "PROJ-1", 6, nil); err != nil {
		t.Errorf("should pass when gate passed: %v", err)
	}
}

func TestStep7_RequiresPassed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(state.TicketStateDir(root, "PROJ-1"), "e2e-fix-state.json")
	writeFile(t, path, `{"rounds":[{"status":"FAILED"}]}`)
	if err := VerifyStepComplete(root, "PROJ-1", 7, nil); err == nil {
		t.Fatal("expected error on FAILED")
	}
	writeFile(t, path, `{"rounds":[{"status":"PASSED"}]}`)
	if err := VerifyStepComplete(root, "PROJ-1", 7, nil); err != nil {
		t.Errorf("should pass on PASSED: %v", err)
	}
}

func TestStep9_CleanPassesFixPasses(t *testing.T) {
	root := t.TempDir()
	stateDir := state.TicketStateDir(root, "PROJ-1")

	cleanReview := "---\nstatus: clean\nfindings:\n  critical: 0\n  warning: 0\n---\n"

	// All three clean → pass.
	for _, name := range []string{"REVIEW-code.md", "REVIEW-simplify.md", "REVIEW-rules.md"} {
		writeFile(t, filepath.Join(stateDir, name), cleanReview)
	}
	if err := VerifyStepComplete(root, "PROJ-1", 9, nil); err != nil {
		t.Errorf("all clean should pass: %v", err)
	}

	// One has findings → reject.
	writeFile(t, filepath.Join(stateDir, "REVIEW-code.md"),
		"---\nstatus: ISSUES_FOUND\nfindings:\n  critical: 1\n  warning: 0\n---\n")
	if err := VerifyStepComplete(root, "PROJ-1", 9, nil); err == nil {
		t.Fatal("expected error with critical findings")
	}

	// REVIEW-FIX.md present → pass regardless of findings.
	writeFile(t, filepath.Join(stateDir, "REVIEW-FIX.md"), "# fixes")
	if err := VerifyStepComplete(root, "PROJ-1", 9, nil); err != nil {
		t.Errorf("REVIEW-FIX.md should allow pass: %v", err)
	}
}

func TestStep10_RequiresImplSummary(t *testing.T) {
	root := t.TempDir()
	if err := VerifyStepComplete(root, "PROJ-1", 10, nil); err == nil {
		t.Fatal("expected error when missing")
	}
	writeFile(t, filepath.Join(state.TicketDir(root, "PROJ-1"), "IMPL-SUMMARY.md"), "")
	if err := VerifyStepComplete(root, "PROJ-1", 10, nil); err != nil {
		t.Errorf("should pass: %v", err)
	}
}

func TestStep11_RequiresPRURL(t *testing.T) {
	root := t.TempDir()
	// Progress with empty pr_url.
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)
	p, _ := state.ReadProgress(root, "PROJ-1")
	if err := VerifyStepComplete(root, "PROJ-1", 11, p); err == nil {
		t.Fatal("expected error when pr_url unset")
	}
	// Set it.
	url := "https://github.com/x/y/pull/1"
	p.PRURL = &url
	if err := VerifyStepComplete(root, "PROJ-1", 11, p); err != nil {
		t.Errorf("should pass: %v", err)
	}
}

func TestStep12_RequiresAllThreeReports(t *testing.T) {
	root := t.TempDir()
	tdir := state.TicketDir(root, "PROJ-1")
	// Zero files → fail.
	if err := VerifyStepComplete(root, "PROJ-1", 12, nil); err == nil {
		t.Fatal("expected error with no reports")
	}
	// Two of three → still fail.
	writeFile(t, filepath.Join(tdir, "PR-BODY.md"), "")
	writeFile(t, filepath.Join(tdir, "JIRA-COMMENT.md"), "")
	if err := VerifyStepComplete(root, "PROJ-1", 12, nil); err == nil {
		t.Fatal("expected error with partial reports")
	}
	// Full set → pass.
	writeFile(t, filepath.Join(tdir, "EXECUTION-REPORT.md"), "")
	if err := VerifyStepComplete(root, "PROJ-1", 12, nil); err != nil {
		t.Errorf("should pass: %v", err)
	}
}

// sanity: verify the helper we rely on in preconditions parses the
// `last_result` field we care about.
func TestBuildGateJSONShape(t *testing.T) {
	var probe struct {
		LastResult string `json:"last_result"`
	}
	_ = json.Unmarshal([]byte(`{"last_result":"pass"}`), &probe)
	if probe.LastResult != "pass" {
		t.Errorf("probe parse: %q", probe.LastResult)
	}
}
