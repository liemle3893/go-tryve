package state

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepoRoot_FromWorktreeReturnsMain creates a real git repo + a linked
// worktree, chdir's into the worktree, and asserts RepoRoot returns the
// MAIN repo path — not the worktree's own top-level. This is the bug the
// WINX-118 smoke run caught: bash's `git rev-parse --show-toplevel` in a
// linked worktree returns the worktree; we want the primary checkout.
func TestRepoRoot_FromWorktreeReturnsMain(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()
	main := filepath.Join(dir, "main")
	wt := filepath.Join(dir, "main-feat")

	// Seed a minimal repo with one commit so `worktree add` has something
	// to branch from.
	runGit(t, dir, "init", main)
	runGit(t, main, "config", "user.email", "t@x")
	runGit(t, main, "config", "user.name", "t")
	runGit(t, main, "commit", "--allow-empty", "-m", "seed")
	runGit(t, main, "worktree", "add", wt, "-b", "feat")

	// Chdir into the worktree and resolve root.
	t.Chdir(wt)

	got, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	// Normalise both paths through EvalSymlinks so /var vs /private/var
	// (macOS) does not trip the comparison.
	resolvedMain := evalOrSame(t, main)
	resolvedGot := evalOrSame(t, got)
	if !strings.EqualFold(resolvedGot, resolvedMain) {
		t.Errorf("want main=%q, got %q (from worktree %q)", resolvedMain, resolvedGot, wt)
	}
}

// TestRepoRoot_FromMainWorks in the non-worktree case.
func TestRepoRoot_FromMainReturnsTop(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	main := filepath.Join(t.TempDir(), "r")
	runGit(t, filepath.Dir(main), "init", main)
	runGit(t, main, "config", "user.email", "t@x")
	runGit(t, main, "config", "user.name", "t")
	runGit(t, main, "commit", "--allow-empty", "-m", "seed")

	t.Chdir(main)

	got, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if evalOrSame(t, got) != evalOrSame(t, main) {
		t.Errorf("want %q, got %q", main, got)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func evalOrSame(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}
