package worktree

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// BootstrapOptions controls how Bootstrap runs. Zero values produce the
// same behaviour as scripts/autoflow/worktree-bootstrap.sh invoked on a
// non-TTY caller: all commands run with NonInteractivePrompter, stdout +
// stderr flow to the provided writers, and empty install/verify cmds are
// skipped.
type BootstrapOptions struct {
	// MainDir is the primary repository root (usually `git rev-parse
	// --show-toplevel` from before the worktree was created).
	MainDir string
	// WorktreeDir is the path to the freshly-created git worktree. Must
	// exist, must not equal MainDir.
	WorktreeDir string
	// Config overrides the auto-loaded .autoflow/bootstrap.json. When
	// nil, Bootstrap reads the config from MainDir and applies AutoDetect.
	Config *Config
	// Prompter handles unrecognised install/verify binaries.
	Prompter Prompter
	// Stdout/Stderr receive progress lines and subprocess output.
	Stdout, Stderr io.Writer
}

// ErrSameDir is returned when WorktreeDir points at MainDir — a worktree
// cannot bootstrap itself.
var ErrSameDir = errors.New("worktree path is the main working directory")

// Bootstrap runs the full init sequence: copy .claude/ infrastructure,
// copy config files, run install, run verify. Returns any fatal error.
// ErrSkipped from unsafe commands is treated as non-fatal.
func Bootstrap(opts BootstrapOptions) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Prompter == nil {
		opts.Prompter = NonInteractivePrompter{}
	}

	if err := validateDirs(opts.MainDir, opts.WorktreeDir); err != nil {
		return err
	}

	cfg := opts.Config
	if cfg == nil {
		loaded, err := ReadConfig(opts.MainDir)
		if err != nil {
			return err
		}
		AutoDetect(loaded, opts.MainDir)
		cfg = loaded
	}

	fmt.Fprintf(opts.Stdout, "Bootstrapping worktree: %s\n", opts.WorktreeDir)
	fmt.Fprintf(opts.Stdout, "Source:                 %s\n\n", opts.MainDir)

	if err := copyClaudeInfra(opts.MainDir, opts.WorktreeDir, opts.Stdout); err != nil {
		return fmt.Errorf("copy .claude/: %w", err)
	}

	if err := copyConfigFiles(opts.MainDir, opts.WorktreeDir, cfg.ConfigFiles, opts.Stdout); err != nil {
		return fmt.Errorf("copy config files: %w", err)
	}

	if cfg.InstallCmd != "" {
		fmt.Fprintf(opts.Stdout, "\nRunning install: %s\n", cfg.InstallCmd)
		if err := RunSafeCmd("install", cfg.InstallCmd, opts.WorktreeDir, opts.Prompter, opts.Stdout, opts.Stderr); err != nil {
			if errors.Is(err, ErrSkipped) {
				fmt.Fprintln(opts.Stdout, "  (skipped)")
			} else {
				return fmt.Errorf("install: %w", err)
			}
		}
	}

	if cfg.VerifyCmd != "" {
		fmt.Fprintf(opts.Stdout, "\nRunning verify: %s\n", cfg.VerifyCmd)
		if err := RunSafeCmd("verify", cfg.VerifyCmd, opts.WorktreeDir, opts.Prompter, opts.Stdout, opts.Stderr); err != nil {
			if errors.Is(err, ErrSkipped) {
				fmt.Fprintln(opts.Stdout, "  (skipped)")
			} else {
				return fmt.Errorf("verify: %w", err)
			}
		}
	}

	fmt.Fprintf(opts.Stdout, "\nWorktree bootstrap complete: %s\n", opts.WorktreeDir)
	if cfg.ServicesCmd != "" {
		fmt.Fprintf(opts.Stdout, "\nNote: run dev services with: %s\n", cfg.ServicesCmd)
	}
	return nil
}

func validateDirs(mainDir, worktreeDir string) error {
	if worktreeDir == "" {
		return errors.New("worktree directory is required")
	}
	st, err := os.Stat(worktreeDir)
	if err != nil {
		return fmt.Errorf("worktree dir: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("worktree path is not a directory: %s", worktreeDir)
	}
	absMain, _ := filepath.Abs(mainDir)
	absWork, _ := filepath.Abs(worktreeDir)
	if absMain == absWork {
		return ErrSameDir
	}
	return nil
}

// copyClaudeInfra copies agents and skills — the gitignored .claude
// contents an autoflow worktree needs. The legacy scripts/autoflow/
// directory is no longer copied because the Go port has no bash scripts
// to ship; if it exists in the source tree it's ignored.
func copyClaudeInfra(mainDir, worktreeDir string, stdout io.Writer) error {
	fmt.Fprintln(stdout, "Copying .claude/ infrastructure:")

	agentsSrc := filepath.Join(mainDir, ".claude", "agents")
	agentsDst := filepath.Join(worktreeDir, ".claude", "agents")
	agentsCopied, err := copyMatchingFiles(agentsSrc, agentsDst, "autoflow-", ".md")
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  OK  agents/ (%d autoflow agents)\n", agentsCopied)

	skillsSrc := filepath.Join(mainDir, ".claude", "skills")
	skillsDst := filepath.Join(worktreeDir, ".claude", "skills")
	skillsCopied, err := copyMatchingDirs(skillsSrc, skillsDst, "autoflow-")
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  OK  skills/ (%d autoflow skills)\n", skillsCopied)
	return nil
}

// copyMatchingFiles copies files in src whose name starts with prefix and
// ends with suffix. Returns the count of files copied.
func copyMatchingFiles(src, dst, prefix, suffix string) (int, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// copyMatchingDirs recursively copies sub-directories of src whose name
// starts with prefix. Returns the count copied.
func copyMatchingDirs(src, dst, prefix string) (int, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// copyConfigFiles copies each listed file from mainDir to worktreeDir,
// preserving the relative path and creating parent dirs as needed.
// Missing files are reported as SKIP — matching the bash script.
func copyConfigFiles(mainDir, worktreeDir string, files []string, stdout io.Writer) error {
	if len(files) == 0 {
		return nil
	}
	fmt.Fprintln(stdout, "\nCopying config files:")
	copied, skipped := 0, 0
	for _, rel := range files {
		srcPath := filepath.Join(mainDir, rel)
		dstPath := filepath.Join(worktreeDir, rel)
		if _, err := os.Stat(srcPath); errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "  SKIP  %s (not found in main dir)\n", rel)
			skipped++
			continue
		} else if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "  OK    %s\n", rel)
		copied++
	}
	fmt.Fprintf(stdout, "  → %d copied, %d skipped\n", copied, skipped)
	return nil
}

// copyTree walks src and copies every regular file to the corresponding
// path under dst. Directory permissions are preserved; file permissions
// use the source mode. Symlinks are followed once (no cycles expected).
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
