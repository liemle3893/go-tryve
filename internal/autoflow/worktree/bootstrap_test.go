package worktree

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// setupRepo creates a fake MainDir with .claude/agents + .claude/skills
// populated as they would be after `tryve install --autoflow`, then a
// sibling empty worktree dir. Returns (mainDir, worktreeDir).
func setupRepo(t *testing.T) (string, string) {
	t.Helper()
	parent := t.TempDir()
	main := filepath.Join(parent, "repo")
	work := filepath.Join(parent, "repo-proj-1")
	if err := os.MkdirAll(main, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed .claude/agents with one matching + one non-matching file.
	agents := filepath.Join(main, ".claude", "agents")
	_ = os.MkdirAll(agents, 0o755)
	_ = os.WriteFile(filepath.Join(agents, "autoflow-jira-fetcher.md"), []byte("match"), 0o644)
	_ = os.WriteFile(filepath.Join(agents, "unrelated-other.md"), []byte("skip"), 0o644)

	// Seed .claude/skills with one matching + one non-matching directory.
	_ = os.MkdirAll(filepath.Join(main, ".claude", "skills", "autoflow-deliver"), 0o755)
	_ = os.WriteFile(filepath.Join(main, ".claude", "skills", "autoflow-deliver", "SKILL.md"), []byte("deliver"), 0o644)
	_ = os.MkdirAll(filepath.Join(main, ".claude", "skills", "other-skill"), 0o755)
	_ = os.WriteFile(filepath.Join(main, ".claude", "skills", "other-skill", "SKILL.md"), []byte("x"), 0o644)

	// .env used by AutoDetect
	_ = os.WriteFile(filepath.Join(main, ".env"), []byte("FOO=1\n"), 0o644)

	return main, work
}

func TestBootstrap_CopiesMatchingInfraOnly(t *testing.T) {
	main, work := setupRepo(t)
	var out, errOut bytes.Buffer
	err := Bootstrap(BootstrapOptions{
		MainDir:     main,
		WorktreeDir: work,
		Config:      &Config{ConfigFiles: []string{".env"}},
		Stdout:      &out,
		Stderr:      &errOut,
	})
	if err != nil {
		t.Fatalf("bootstrap: %v\nSTDOUT: %s\nSTDERR: %s", err, out.String(), errOut.String())
	}

	// Matching agent copied.
	if _, err := os.Stat(filepath.Join(work, ".claude", "agents", "autoflow-jira-fetcher.md")); err != nil {
		t.Errorf("autoflow agent not copied: %v", err)
	}
	// Non-matching NOT copied.
	if _, err := os.Stat(filepath.Join(work, ".claude", "agents", "unrelated-other.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("non-autoflow agent was copied")
	}
	// Matching skill dir copied.
	if _, err := os.Stat(filepath.Join(work, ".claude", "skills", "autoflow-deliver", "SKILL.md")); err != nil {
		t.Errorf("autoflow skill not copied: %v", err)
	}
	// Non-matching skill NOT copied.
	if _, err := os.Stat(filepath.Join(work, ".claude", "skills", "other-skill", "SKILL.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("non-autoflow skill was copied")
	}
	// Config file copied.
	if _, err := os.Stat(filepath.Join(work, ".env")); err != nil {
		t.Errorf(".env not copied: %v", err)
	}
}

func TestBootstrap_SameDirFails(t *testing.T) {
	main, _ := setupRepo(t)
	err := Bootstrap(BootstrapOptions{MainDir: main, WorktreeDir: main})
	if !errors.Is(err, ErrSameDir) {
		t.Errorf("want ErrSameDir, got %v", err)
	}
}

func TestBootstrap_WorktreeMustExist(t *testing.T) {
	main, _ := setupRepo(t)
	err := Bootstrap(BootstrapOptions{MainDir: main, WorktreeDir: filepath.Join(t.TempDir(), "does-not-exist")})
	if err == nil {
		t.Errorf("expected error when worktree missing")
	}
}

func TestBootstrap_SkipsInstallIfNoCmd(t *testing.T) {
	main, work := setupRepo(t)
	var out, errOut bytes.Buffer
	err := Bootstrap(BootstrapOptions{
		MainDir:     main,
		WorktreeDir: work,
		Config:      &Config{}, // empty — no install/verify
		Stdout:      &out,
		Stderr:      &errOut,
	})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	// Should not have attempted to run install.
	if bytes.Contains(out.Bytes(), []byte("Running install")) {
		t.Errorf("should not attempt install with empty cmd")
	}
}

func TestBootstrap_SkipsUnrecognisedInstallNonInteractive(t *testing.T) {
	main, work := setupRepo(t)
	var out, errOut bytes.Buffer
	err := Bootstrap(BootstrapOptions{
		MainDir:     main,
		WorktreeDir: work,
		Config:      &Config{InstallCmd: "mysterycommand"},
		Stdout:      &out,
		Stderr:      &errOut,
	})
	if err != nil {
		t.Fatalf("bootstrap should succeed (command skipped), got %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("(skipped)")) {
		t.Errorf("expected skipped message in output, got:\n%s", out.String())
	}
}
