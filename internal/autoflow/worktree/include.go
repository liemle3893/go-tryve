package worktree

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IncludeFile is the filename of the per-repo bootstrap copy manifest
// at <mainDir>/.worktreeinclude. Complements config_files in
// bootstrap.json: use bootstrap.json for things every worktree of
// every project needs, and .worktreeinclude for this project's
// project-specific extras (local tool configs, generated fixtures,
// certificates, etc.).
const IncludeFile = ".worktreeinclude"

// ReadIncludes returns the list of paths declared in
// <mainDir>/.worktreeinclude, relative to mainDir. Returns a nil slice
// (and no error) when the file does not exist.
//
// Format rules, kept deliberately minimal:
//   - one path per line, resolved relative to mainDir
//   - blank lines and lines starting with # are ignored
//   - leading/trailing whitespace on a line is trimmed
//   - no glob syntax; a path that is a directory is copied recursively
//   - paths starting with / or containing .. are rejected (would escape
//     the main repo / worktree)
func ReadIncludes(mainDir string) ([]string, error) {
	f, err := os.Open(filepath.Join(mainDir, IncludeFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "/") {
			return nil, fmt.Errorf("%s:%d: absolute paths not allowed (%q)", IncludeFile, lineNo, line)
		}
		clean := filepath.Clean(line)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fmt.Errorf("%s:%d: paths must stay inside the repo (%q)", IncludeFile, lineNo, line)
		}
		out = append(out, clean)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// copyIncludes copies every entry from the manifest into worktreeDir,
// preserving the relative path. Directory entries are walked; file
// entries are copied as-is. Missing entries are reported as SKIP (not
// an error), mirroring the existing copy_config_files behaviour.
func copyIncludes(mainDir, worktreeDir string, includes []string, stdout io.Writer) error {
	if len(includes) == 0 {
		return nil
	}
	fmt.Fprintln(stdout, "\nCopying .worktreeinclude entries:")
	copied, skipped, dirs := 0, 0, 0
	for _, rel := range includes {
		srcPath := filepath.Join(mainDir, rel)
		dstPath := filepath.Join(worktreeDir, rel)
		st, err := os.Stat(srcPath)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "  SKIP  %s (not found)\n", rel)
			skipped++
			continue
		}
		if err != nil {
			return err
		}
		if st.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyTree(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy dir %s: %w", rel, err)
			}
			fmt.Fprintf(stdout, "  OK    %s/\n", rel)
			dirs++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy file %s: %w", rel, err)
		}
		fmt.Fprintf(stdout, "  OK    %s\n", rel)
		copied++
	}
	fmt.Fprintf(stdout, "  → %d files, %d dirs, %d skipped\n", copied, dirs, skipped)
	return nil
}
