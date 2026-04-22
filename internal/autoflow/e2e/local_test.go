package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liemle3893/autoflow/pkg/runner"
)

func TestApplyTestSelection_Tag(t *testing.T) {
	var opts runner.Options
	applyTestSelection("--tag PROJ-42", &opts)
	if len(opts.Tags) != 1 || opts.Tags[0] != "PROJ-42" {
		t.Errorf("tag parse: got %+v", opts.Tags)
	}
}

func TestApplyTestSelection_Grep(t *testing.T) {
	var opts runner.Options
	applyTestSelection("--grep foo", &opts)
	if opts.Grep != "foo" {
		t.Errorf("grep parse: got %q", opts.Grep)
	}
}

func TestApplyTestSelection_Glob(t *testing.T) {
	var opts runner.Options
	applyTestSelection("tests/e2e/**/TC-PROJ-1-*.test.yaml", &opts)
	if opts.Grep == "" {
		t.Errorf("glob should fall through to Grep")
	}
}

func TestApplyTestSelection_Empty(t *testing.T) {
	var opts runner.Options
	applyTestSelection("   ", &opts)
	if len(opts.Tags) != 0 || opts.Grep != "" {
		t.Errorf("empty selection should leave opts untouched")
	}
}

func TestApplyTestSelection_TagAndGrep(t *testing.T) {
	var opts runner.Options
	applyTestSelection("--tag PROJ-1 --grep smoke", &opts)
	if len(opts.Tags) != 1 || opts.Tags[0] != "PROJ-1" {
		t.Errorf("expected one tag, got %+v", opts.Tags)
	}
	if opts.Grep != "smoke" {
		t.Errorf("expected grep=smoke, got %q", opts.Grep)
	}
}

func TestRunLocal_MissingWorkDir(t *testing.T) {
	_, err := RunLocal(context.Background(), LocalOptions{})
	if err == nil {
		t.Fatalf("expected error for missing WorkDir")
	}
	if !strings.Contains(err.Error(), "WorkDir") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRunLocal_WritesSummaryInWorkDir verifies the happy-path wiring:
// RunLocal reads bootstrap.json from WorkDir (if present), skips build
// when none is configured, imports env, runs the (empty) test suite and
// writes a summary. No merge, no worktree sync.
func TestRunLocal_WritesSummaryInWorkDir(t *testing.T) {
	work := t.TempDir()
	// Minimal bootstrap.json — no build/verify so RunSafeCmd is skipped.
	bootstrap := `{"language":"go","base_branch":"main","config_files":[],"install_cmd":"","verify_cmd":"","build_cmd":"","test_cmd":"","services_cmd":""}`
	if err := os.MkdirAll(filepath.Join(work, ".autoflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".autoflow", "bootstrap.json"), []byte(bootstrap), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(work, "run.txt")
	_, err := RunLocal(context.Background(), LocalOptions{
		WorkDir:    work,
		ConfigPath: filepath.Join(work, "nonexistent.yaml"),
		OutputFile: outFile,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
	// runner.RunTests with a nonexistent config returns an error — that
	// is fine for this wiring test. What we care about is that RunLocal
	// proceeded past the build / env / summary-write stages.
	if _, serr := os.Stat(outFile); serr != nil {
		t.Fatalf("expected summary file written, got stat error %v (runErr=%v)", serr, err)
	}
}
