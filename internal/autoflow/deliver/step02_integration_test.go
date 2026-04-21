package deliver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liemle3893/go-tryve/internal/autoflow/state"
)

// TestStep02_Integration exercises step_02 end-to-end against a real git
// repo. Verifies the controller:
//   - runs git fetch + git worktree add,
//   - copies .claude infra via worktree.Bootstrap,
//   - writes workflow-progress.json with title set and step 1 completed,
//   - returns an auto_complete instruction with the jira_transition post action.
func TestStep02_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()
	main := filepath.Join(dir, "project")

	// Create an "origin" (bare) and the main working clone. `git worktree
	// add` needs a real remote-tracking branch.
	origin := filepath.Join(dir, "origin.git")
	sh(t, dir, "git", "init", "--bare", origin)
	sh(t, dir, "git", "clone", origin, main)
	sh(t, main, "git", "config", "user.email", "t@x")
	sh(t, main, "git", "config", "user.name", "t")
	sh(t, main, "git", "commit", "--allow-empty", "-m", "seed")
	sh(t, main, "git", "branch", "-M", "main")
	sh(t, main, "git", "push", "-u", "origin", "main")

	// Seed a task-brief with a title so step_02 can derive a slug from it.
	briefDir := filepath.Join(main, ".autoflow", "ticket", "PROJ-1")
	if err := os.MkdirAll(briefDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(briefDir, "task-brief.md"),
		[]byte("---\ntitle: Add example feature\n---\n# Add example feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewController(main)
	instr, err := c.Next("PROJ-1")
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if instr.Action != ActionAutoComplete {
		// Read the log for diagnostics if we fall off the happy path.
		log, _ := os.ReadFile(filepath.Join(briefDir, "step-02.log"))
		t.Fatalf("want auto_complete, got %s (reason=%q)\nlog:\n%s",
			instr.Action, instr.Reason, log)
	}
	if instr.Step != 2 {
		t.Errorf("want step=2, got %d", instr.Step)
	}
	if len(instr.PostActions) != 1 || instr.PostActions[0].Action != "jira_transition" {
		t.Errorf("expected one jira_transition post action, got %+v", instr.PostActions)
	}

	// Worktree directory should exist at <parent>/<basename>-<ticket-lower>.
	wtPath := filepath.Join(dir, "project-proj-1")
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree directory missing at %s: %v", wtPath, err)
	}

	// Progress should be initialised with the branch + title, step 1 completed.
	p, err := state.ReadProgress(main, "PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("progress not created")
	}
	if p.Title == nil || *p.Title != "Add example feature" {
		t.Errorf("title not set: %+v", p.Title)
	}
	if !strings.HasPrefix(p.Branch, "jira-iss/proj-1-") {
		t.Errorf("branch name not derived from title: %q", p.Branch)
	}
	foundStep1 := false
	for _, s := range p.Completed {
		if s == 1 {
			foundStep1 = true
		}
	}
	if !foundStep1 {
		t.Errorf("step 1 not marked complete: %+v", p.Completed)
	}
	if p.CurrentStep != 2 {
		t.Errorf("current_step should be 2 after step 1 done, got %d", p.CurrentStep)
	}
}

// TestStep02_IdempotentWhenWorktreeExists — if progress already records a
// worktree and it exists on disk, step_02 auto-completes without trying to
// fetch/create anything.
func TestStep02_Idempotent(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	existingWT := t.TempDir()
	_, _ = state.InitProgress(root, "PROJ-1", existingWT, "feat", false)
	p, _ := state.ReadProgress(root, "PROJ-1")

	instr := c.step02("PROJ-1", p)
	if instr.Action != ActionAutoComplete {
		t.Errorf("existing worktree should auto-complete, got %s", instr.Action)
	}
	if !strings.Contains(instr.Reason, "already exists") {
		t.Errorf("reason should mention existing worktree, got %q", instr.Reason)
	}
}

func TestMakeSlug(t *testing.T) {
	cases := map[string]string{
		"Hello World":                  "hello-world",
		"feat: add user rate limiting": "feat-add-user-rate-limiting",
		"[PROJ-42] fix bug":            "proj-42-fix-bug",
		"   leading  spaces   ":        "leading-spaces",
		"under_scores/and-dashes":      "under-scores-and-dashes",
		"超长title with 非ASCII":         "title-with-ascii",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 40 max
	}
	for in, want := range cases {
		if got := makeSlug(in); got != want {
			t.Errorf("makeSlug(%q)=%q want %q", in, got, want)
		}
	}
}

// sh runs cmd in dir and fails the test on non-zero exit.
func sh(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	c := exec.Command(name, args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%s %v in %s: %v\n%s", name, args, dir, err, out)
	}
}
