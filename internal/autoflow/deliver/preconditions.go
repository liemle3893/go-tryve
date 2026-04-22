package deliver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
	"github.com/liemle3893/autoflow/internal/autoflow/worktree"
)

// Precondition is a single check that must pass before a step can be
// marked complete. Checks are intentionally simple — file existence, a
// JSON field read, a glob — so any failure maps to a concrete user
// action the error message can point at.
type Precondition struct {
	Name  string
	Check func(root, key string, progress *state.Progress) error
}

// stepPreconditions returns the checks that must all pass before the
// given step is considered complete. Steps 3, 8, 13 return nil — no
// canonical local signal to key on, so we trust the caller.
func stepPreconditions(step int) []Precondition {
	switch step {
	case 1:
		return []Precondition{{
			Name:  "task-brief.md exists",
			Check: checkTicketFile("task-brief.md", "run the autoflow-jira-fetcher agent (step 1 dispatch)"),
		}}
	case 2:
		return []Precondition{
			{
				Name:  "workflow-progress.json exists",
				Check: checkProgressExists,
			},
			{
				Name:  "worktree directory exists",
				Check: checkWorktreeDir,
			},
		}
	case 4:
		return []Precondition{{
			Name:  "coverage-review-state.json last round is PASS",
			Check: checkLoopLastStatus("coverage-review-state.json", "PASS", "autoflow-ac-reviewer must end with status=PASS"),
		}}
	case 5:
		return []Precondition{{
			Name:  "PLAN.md or SUMMARY.md exists",
			Check: checkAnyTicketFile([]string{"PLAN.md", "SUMMARY.md"}, "run autoflow-executor (direct) or autoflow-planner + autoflow-executor (plan)"),
		}}
	case 6:
		return []Precondition{{
			Name:  "build-gate passed or not configured",
			Check: checkBuildGate,
		}}
	case 7:
		return []Precondition{{
			Name:  "e2e-fix-state.json last round is PASSED",
			Check: checkLoopLastStatus("e2e-fix-state.json", "PASSED", "E2E tests must pass (last round status=PASSED)"),
		}}
	case 9:
		return []Precondition{{
			Name:  "review complete (REVIEW-FIX.md or all reviews clean)",
			Check: checkReviewComplete,
		}}
	case 10:
		return []Precondition{{
			Name:  "IMPL-SUMMARY.md exists",
			Check: checkTicketFile("IMPL-SUMMARY.md", "run the autoflow-docs-writer agent"),
		}}
	case 11:
		return []Precondition{{
			Name:  "pr_url set in workflow-progress.json",
			Check: checkPRURL,
		}}
	case 12:
		return []Precondition{{
			Name:  "PR-BODY / JIRA-COMMENT / EXECUTION-REPORT all exist",
			Check: checkReports,
		}}
	default:
		return nil
	}
}

// VerifyStepComplete returns the first precondition error for step, or
// nil if all pass. Also returns nil when step has no preconditions.
func VerifyStepComplete(root, key string, step int, progress *state.Progress) error {
	for _, p := range stepPreconditions(step) {
		if err := p.Check(root, key, progress); err != nil {
			return fmt.Errorf("step %d — %s: %w", step, p.Name, err)
		}
	}
	return nil
}

// ── Individual checks ────────────────────────────────────────────────

// checkTicketFile returns a check that requires `.autoflow/ticket/<KEY>/
// <rel>` to exist.
func checkTicketFile(rel, hint string) func(root, key string, _ *state.Progress) error {
	return func(root, key string, _ *state.Progress) error {
		path := filepath.Join(state.TicketDir(root, key), rel)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("%s missing (hint: %s)", path, hint)
		}
		return nil
	}
}

// checkAnyTicketFile is like checkTicketFile but passes if any one of
// the listed relative paths exists.
func checkAnyTicketFile(rels []string, hint string) func(root, key string, _ *state.Progress) error {
	return func(root, key string, _ *state.Progress) error {
		for _, rel := range rels {
			if _, err := os.Stat(filepath.Join(state.TicketDir(root, key), rel)); err == nil {
				return nil
			}
		}
		return fmt.Errorf("none of %v found under %s (hint: %s)",
			rels, state.TicketDir(root, key), hint)
	}
}

func checkProgressExists(root, key string, _ *state.Progress) error {
	path := state.ProgressFile(root, key)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%s missing — step 2 should have seeded it", path)
	}
	return nil
}

func checkWorktreeDir(_, _ string, progress *state.Progress) error {
	if progress == nil || progress.Worktree == "" {
		return errors.New("progress.worktree unset — step 2 did not record a worktree path")
	}
	if _, err := os.Stat(progress.Worktree); err != nil {
		return fmt.Errorf("worktree dir %s missing — git worktree add failed or was removed", progress.Worktree)
	}
	return nil
}

// checkLoopLastStatus verifies that the named loop state file's last
// round has status == wantStatus.
func checkLoopLastStatus(stateFile, wantStatus, hint string) func(root, key string, _ *state.Progress) error {
	return func(root, key string, _ *state.Progress) error {
		path := filepath.Join(state.TicketStateDir(root, key), stateFile)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s missing (hint: %s)", path, hint)
		}
		var probe struct {
			Rounds []struct {
				Status string `json:"status"`
			} `json:"rounds"`
		}
		if err := json.Unmarshal(data, &probe); err != nil {
			return fmt.Errorf("%s is not valid JSON: %w", path, err)
		}
		if len(probe.Rounds) == 0 {
			return fmt.Errorf("%s has no rounds recorded yet (hint: %s)", path, hint)
		}
		last := probe.Rounds[len(probe.Rounds)-1].Status
		if last != wantStatus {
			return fmt.Errorf("%s last round status=%q, want %q (hint: %s)",
				path, last, wantStatus, hint)
		}
		return nil
	}
}

// checkBuildGate passes when either the gate reports pass, or the
// bootstrap config has neither a build_cmd nor a test_cmd (step 6
// auto-skips in that case).
func checkBuildGate(root, key string, _ *state.Progress) error {
	cfg, _ := worktree.ReadConfig(root)
	hasCmds := false
	if cfg != nil {
		if (cfg.BuildCmd != "" || cfg.VerifyCmd != "") || cfg.TestCmd != "" {
			hasCmds = true
		}
	}
	if !hasCmds {
		return nil // step 6 auto-completes when nothing is configured
	}
	path := filepath.Join(state.TicketStateDir(root, key), "build-gate-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s missing — build gate has not run", path)
	}
	var probe struct {
		LastResult string `json:"last_result"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	if probe.LastResult != "pass" {
		return fmt.Errorf("%s last_result=%q, want \"pass\"", path, probe.LastResult)
	}
	return nil
}

// checkReviewComplete passes when either REVIEW-FIX.md exists (fixer ran
// against findings) or all three REVIEW-*.md reports have zero critical
// and zero warning findings.
func checkReviewComplete(root, key string, _ *state.Progress) error {
	stateDir := state.TicketStateDir(root, key)
	fix := filepath.Join(stateDir, "REVIEW-FIX.md")
	if _, err := os.Stat(fix); err == nil {
		return nil
	}
	// No fixer output — demand clean reports.
	for _, name := range []string{"REVIEW-code.md", "REVIEW-simplify.md", "REVIEW-rules.md"} {
		path := filepath.Join(stateDir, name)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("%s missing — review has not run", path)
		}
		c, w := countReviewFindings(path)
		if c+w > 0 {
			return fmt.Errorf("%s has %d critical + %d warning findings — run autoflow-code-fixer or write REVIEW-FIX.md",
				path, c, w)
		}
	}
	return nil
}

func checkPRURL(_, _ string, progress *state.Progress) error {
	if progress == nil || progress.PRURL == nil || *progress.PRURL == "" {
		return errors.New("pr_url unset — `deliver complete` for step 11 must include --pr-url")
	}
	return nil
}

func checkReports(root, key string, _ *state.Progress) error {
	tdir := state.TicketDir(root, key)
	for _, name := range []string{"PR-BODY.md", "JIRA-COMMENT.md", "EXECUTION-REPORT.md"} {
		if _, err := os.Stat(filepath.Join(tdir, name)); err != nil {
			return fmt.Errorf("%s/%s missing — run `autoflow deliver _report`", tdir, name)
		}
	}
	return nil
}
