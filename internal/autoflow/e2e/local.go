package e2e

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/liemle3893/go-tryve/internal/autoflow/worktree"
	"github.com/liemle3893/go-tryve/internal/tryve"
	"github.com/liemle3893/go-tryve/pkg/runner"
)

// LocalOptions controls one E2E run against the main working directory.
// Zero values inherit the bash script's defaults where applicable.
type LocalOptions struct {
	// MainDir is the primary checkout (not the worktree). E2E tests
	// actually execute here, because the typical project target (Azure
	// Functions host, a single local server bound to a port) can only
	// be running in one tree at a time.
	MainDir string
	// Branch is the remote branch (usually in origin) whose tip should
	// be merged into MainDir before the test run.
	Branch string
	// WorktreeDir, when non-empty, is a linked worktree whose test
	// files should be synced into MainDir ahead of the run. This
	// handles the case where new *.test.yaml files exist in the
	// worktree but have not yet been pushed to the branch.
	WorktreeDir string
	// TestSelection is either a tag expression ("--tag PROJ-42") or a
	// glob/path. When it begins with "--" the value is treated as CLI
	// args and parsed into runner.Options.Tags or .Grep accordingly.
	TestSelection string
	// ConfigPath points at e2e.config.yaml in MainDir.
	ConfigPath string
	// Environment is the key selected from the config.
	Environment string
	// OutputFile is where the run log gets written for the report
	// stage (EXECUTION-REPORT.md links it). Optional — when empty,
	// defaults to /tmp/e2e-results-<safe-branch>.txt.
	OutputFile string
	// UseLock enables concurrency serialisation via flock on a sentinel
	// file inside MainDir.
	UseLock bool
	// Stdout/Stderr capture human-readable progress.
	Stdout, Stderr io.Writer
}

// LocalResult summarises one run. Outcome holds the structured tryve
// SuiteResult so callers can display per-test detail (count, duration,
// status) without regex-parsing console output.
type LocalResult struct {
	// Outcome is nil on a setup error (e.g. merge conflict). On any
	// run that actually executed tests, it is populated.
	Outcome    *tryve.SuiteResult
	OutputFile string
}

// ErrMainDirty is returned when MainDir has uncommitted changes at the
// start of a run. Matches the bash script's early exit.
var ErrMainDirty = errors.New("main working directory has uncommitted changes")

// ErrMergeConflict is returned when applying Branch fails to merge
// cleanly. Matches the bash script's exit 2.
var ErrMergeConflict = errors.New("merge conflict applying branch")

// RunLocal performs the full sequence: lock, git merge, build if set,
// import env, run tests via the runner API, unmerge on exit. Returns a
// LocalResult plus any fatal error.
func RunLocal(ctx context.Context, opts LocalOptions) (*LocalResult, error) {
	if opts.MainDir == "" || opts.Branch == "" {
		return nil, fmt.Errorf("MainDir and Branch are required")
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(os.TempDir(),
			"e2e-results-"+safeBranch(opts.Branch)+".txt")
	}

	if opts.UseLock {
		lock, err := Acquire(ctx, filepath.Join(opts.MainDir, ".e2e-lock"), 10*time.Minute)
		if err != nil {
			return nil, err
		}
		defer lock.Release()
	}

	if err := ensureClean(opts.MainDir); err != nil {
		return nil, err
	}

	preSHA, err := rev(opts.MainDir, "HEAD")
	if err != nil {
		return nil, err
	}

	// Fetch the branch first so FETCH_HEAD exists.
	if err := gitCmd(opts.MainDir, opts.Stdout, opts.Stderr, "fetch", "origin", opts.Branch); err != nil {
		return nil, fmt.Errorf("git fetch: %w", err)
	}

	// Attempt the merge. On conflict, abort and bail out.
	if err := gitCmd(opts.MainDir, opts.Stdout, opts.Stderr, "merge", "FETCH_HEAD", "--no-edit"); err != nil {
		_ = gitCmd(opts.MainDir, nil, nil, "merge", "--abort")
		return nil, fmt.Errorf("%w: %v", ErrMergeConflict, err)
	}

	// Cleanup runs on any exit — reset to preSHA and restore tests/e2e.
	defer func() {
		_ = gitCmd(opts.MainDir, opts.Stdout, opts.Stderr, "reset", "--hard", preSHA)
		_ = gitCmd(opts.MainDir, nil, nil, "checkout", "--", "tests/e2e/")
	}()

	// Pull freshly authored test files from the worktree if requested.
	if opts.WorktreeDir != "" {
		if err := syncTestFiles(opts.WorktreeDir, opts.MainDir, opts.TestSelection, opts.Stdout); err != nil {
			// Non-fatal — proceed with whatever the branch contained.
			fmt.Fprintf(opts.Stderr, "WARN: test-file sync: %v\n", err)
		}
	}

	// Run the build step from bootstrap.json so a newly merged branch
	// rebuilds before tests execute.
	if cfg, err := worktree.ReadConfig(opts.MainDir); err == nil {
		build := cfg.BuildCmd
		if build == "" {
			build = cfg.VerifyCmd
		}
		if build != "" {
			fmt.Fprintf(opts.Stdout, "\n── BUILD: %s ──\n", build)
			if err := worktree.RunSafeCmd("build", build, opts.MainDir, worktree.NonInteractivePrompter{}, opts.Stdout, opts.Stderr); err != nil && !errors.Is(err, worktree.ErrSkipped) {
				return nil, fmt.Errorf("build: %w", err)
			}
		}
	}

	// Import env vars. .env overrides caller env; local.settings.json
	// fills remaining gaps.
	if err := ImportWorkspaceEnv(opts.MainDir); err != nil {
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
// filter ("--tag PROJ-42") or a file glob passed as-is to Grep. The
// bash version accepted both; preserve the same entry points.
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

func ensureClean(dir string) error {
	out, err := runGit(dir, "status", "--short")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("%w:\n%s", ErrMainDirty, out)
	}
	return nil
}

func rev(dir, ref string) (string, error) {
	out, err := runGit(dir, "rev-parse", ref)
	return strings.TrimSpace(out), err
}

func gitCmd(dir string, stdout, stderr io.Writer, args ...string) error {
	c := exec.Command("git", args...)
	c.Dir = dir
	if stdout != nil {
		c.Stdout = stdout
	}
	if stderr != nil {
		c.Stderr = stderr
	}
	return c.Run()
}

func runGit(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.Output()
	return string(out), err
}

// syncTestFiles copies *.test.yaml from worktree's test tree into main.
// The set of files to copy is inferred from the TestSelection — when
// the selection is a path-ish glob we copy its parent directory's
// contents; when it's a tag expression we copy the whole tests/e2e
// tree (cheapest-correct: agents writing tests put them under that
// subtree).
func syncTestFiles(worktreeDir, mainDir, sel string, stdout io.Writer) error {
	base := "tests/e2e"
	if !strings.HasPrefix(strings.TrimSpace(sel), "--") && sel != "" {
		base = filepath.Dir(sel)
	}
	src := filepath.Join(worktreeDir, base)
	dst := filepath.Join(mainDir, base)
	if _, err := os.Stat(src); err != nil {
		return nil // nothing to sync
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".test.yaml") {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyOne(path, target)
	})
}

func copyOne(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func writeRunSummary(path string, res *tryve.SuiteResult, runErr error) error {
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

func safeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}
