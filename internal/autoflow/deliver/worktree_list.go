package deliver

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// worktreeInfo is one entry from `git worktree list --porcelain`.
type worktreeInfo struct {
	Path   string
	Branch string // e.g. "refs/heads/feat"; empty for detached HEAD
}

// listWorktrees returns one entry per linked worktree (and the main
// checkout) of mainDir. Returns nil on any git error; callers treat a
// nil result as "can't verify, be conservative."
func listWorktrees(mainDir string) []worktreeInfo {
	out, err := exec.Command("git", "-C", mainDir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil
	}
	var infos []worktreeInfo
	var current worktreeInfo
	flush := func() {
		if current.Path != "" {
			infos = append(infos, current)
		}
		current = worktreeInfo{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch ")
		}
	}
	flush()
	return infos
}

// findRegisteredWorktree returns (branch, true) when path is a linked
// worktree of mainDir. Comparison is done on canonicalised paths so
// `/var` vs `/private/var` (macOS) does not trip the match.
func findRegisteredWorktree(mainDir, path string) (string, bool) {
	want, err := filepath.EvalSymlinks(path)
	if err != nil {
		want = path
	}
	for _, wt := range listWorktrees(mainDir) {
		got, err := filepath.EvalSymlinks(wt.Path)
		if err != nil {
			got = wt.Path
		}
		if got == want {
			// Strip refs/heads/ prefix so callers get a bare branch name.
			return strings.TrimPrefix(wt.Branch, "refs/heads/"), true
		}
	}
	return "", false
}
