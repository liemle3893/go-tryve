package e2e

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liemle3893/autoflow/internal/autoflow/worktree"
	"github.com/liemle3893/autoflow/internal/core"
	"github.com/liemle3893/autoflow/pkg/runner"
)

// LocalOptions controls one E2E run inside a feature worktree.
//
// With sandbox isolation in place, tests run directly in the worktree
// where the agent authored them — no merge onto a base branch, no
// test-file sync from one tree to another.
type LocalOptions struct {
	// WorkDir is the directory tests execute in (typically a feature
	// worktree). Required.
	WorkDir string
	// TestSelection is either a tag expression ("--tag PROJ-42") or a
	// glob/path. When it begins with "--" the value is treated as CLI
	// args and parsed into runner.Options.Tags or .Grep accordingly.
	TestSelection string
	// ConfigPath points at e2e.config.yaml in WorkDir.
	ConfigPath string
	// Environment is the key selected from the config.
	Environment string
	// OutputFile is where the run log gets written for the report
	// stage (EXECUTION-REPORT.md links it). Optional — when empty,
	// callers should supply a ticket-keyed default.
	OutputFile string
	// UseLock enables concurrency serialisation via flock on a sentinel
	// file inside WorkDir. Off by default — sandboxed runs typically
	// own their worktree exclusively.
	UseLock bool
	// Stdout/Stderr capture human-readable progress.
	Stdout, Stderr io.Writer
}

// LocalResult summarises one run. Outcome holds the structured core
// SuiteResult so callers can display per-test detail (count, duration,
// status) without regex-parsing console output.
type LocalResult struct {
	// Outcome is nil on a setup error. On any run that actually
	// executed tests, it is populated.
	Outcome    *core.SuiteResult
	OutputFile string
}

// RunLocal performs one E2E run in WorkDir: optional lock, build, env
// import, test execution, summary write. Returns a LocalResult plus any
// fatal error.
func RunLocal(ctx context.Context, opts LocalOptions) (*LocalResult, error) {
	if opts.WorkDir == "" {
		return nil, fmt.Errorf("WorkDir is required")
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(os.TempDir(), "e2e-results.txt")
	}

	if opts.UseLock {
		lock, err := Acquire(ctx, filepath.Join(opts.WorkDir, ".e2e-lock"), 10*time.Minute)
		if err != nil {
			return nil, err
		}
		defer lock.Release()
	}

	// Run the build step from bootstrap.json so the worktree is rebuilt
	// before tests execute.
	if cfg, err := worktree.ReadConfig(opts.WorkDir); err == nil {
		build := cfg.BuildCmd
		if build == "" {
			build = cfg.VerifyCmd
		}
		if build != "" {
			fmt.Fprintf(opts.Stdout, "\n── BUILD: %s ──\n", build)
			if err := worktree.RunSafeCmd("build", build, opts.WorkDir, worktree.NonInteractivePrompter{}, opts.Stdout, opts.Stderr); err != nil && !errors.Is(err, worktree.ErrSkipped) {
				return nil, fmt.Errorf("build: %w", err)
			}
		}
	}

	// Import env vars. .env overrides caller env; local.settings.json
	// fills remaining gaps.
	if err := ImportWorkspaceEnv(opts.WorkDir); err != nil {
		return nil, fmt.Errorf("import env: %w", err)
	}

	runOpts := runner.Options{
		ConfigPath:  opts.ConfigPath,
		Environment: opts.Environment,
	}
	applyTestSelection(opts.TestSelection, &runOpts)

	res, runErr := runner.RunTests(ctx, runOpts)
	// Persist a text summary of the run for the report stage. We always
	// write — a missing file downstream is ambiguous.
	if err := writeRunSummary(opts.OutputFile, res, runErr); err != nil {
		fmt.Fprintf(opts.Stderr, "WARN: write run summary: %v\n", err)
	}

	return &LocalResult{Outcome: res, OutputFile: opts.OutputFile}, runErr
}

// applyTestSelection interprets the selection string as either a tag
// filter ("--tag PROJ-42") or a file glob passed as-is to Grep. Both
// forms are preserved from the bash version.
func applyTestSelection(sel string, out *runner.Options) {
	trimmed := strings.TrimSpace(sel)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(trimmed, "--") {
		fields := strings.Fields(trimmed)
		for i := 0; i < len(fields); i++ {
			switch fields[i] {
			case "--tag":
				if i+1 < len(fields) {
					out.Tags = append(out.Tags, fields[i+1])
					i++
				}
			case "--grep":
				if i+1 < len(fields) {
					out.Grep = fields[i+1]
					i++
				}
			}
		}
		return
	}
	// Otherwise assume it's a path/glob and pass through Grep — runner
	// treats Grep as a regex over test names, which is close enough for
	// the common use case.
	out.Grep = trimmed
}

func writeRunSummary(path string, res *core.SuiteResult, runErr error) error {
	var sb strings.Builder
	if res != nil {
		fmt.Fprintf(&sb, "Running %d test(s)\n", res.Total)
		for _, tr := range res.Tests {
			if tr.Test == nil {
				continue
			}
			status := strings.ToLower(string(tr.Status))
			fmt.Fprintf(&sb, "Test %s: %s (%dms)\n",
				tr.Test.Name, status, tr.Duration.Milliseconds())
		}
		fmt.Fprintf(&sb, "Total tests: %d\n", res.Total)
		fmt.Fprintf(&sb, "%d passed %d failed %d skipped\n",
			res.Passed, res.Failed, res.Skipped)
		fmt.Fprintf(&sb, "Total duration: %s\n", res.Duration)
	}
	if runErr != nil {
		fmt.Fprintf(&sb, "ERROR: %v\n", runErr)
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}
