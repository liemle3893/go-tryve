// Package state implements the durable JSON state files that drive the
// autoflow-deliver workflow. It replaces the shell scripts progress-state.sh,
// loop-state.sh, review-loop.sh, and verify-gates.sh from winx-autoflow.
//
// File layout under a repo root:
//
//	.autoflow/ticket/<KEY>/
//	    task-brief.md
//	    title.txt
//	    workflow-progress.json
//	    state/
//	        coverage-review-state.json
//	        e2e-fix-state.json
//	        build-gate-state.json
//	        code-review-state.json
//	        review-feedback.json
//	        .review-state-checksum
//	        REVIEW-{code,simplify,rules,FIX}.md
//	        *.marker
package state

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ticketKeyPattern matches Jira-style keys: one uppercase letter, then
// uppercase letters/digits, then a dash, then digits. Guards against path
// traversal in commands that build paths from a ticket key.
var ticketKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)

// ValidateTicketKey returns an error if key is not a syntactically valid
// Jira key (e.g. "PROJ-42"). Used at every boundary that receives a key
// from an external caller.
func ValidateTicketKey(key string) error {
	if !ticketKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid ticket key %q: expected PROJECT-123 format", key)
	}
	return nil
}

// RepoRoot returns the MAIN repository's top-level directory. When called
// from inside a linked git worktree, this is the parent repo (NOT the
// worktree itself) — so state files always land in a single canonical
// location regardless of which checkout the command was launched from.
//
// Detection uses `git rev-parse --git-common-dir`, which always points at
// the main repo's `.git` directory even from a linked worktree. Falls
// back to `--show-toplevel` in the edge cases where common-dir doesn't
// end in `/.git` (bare repos, custom gitdir layouts).
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err == nil {
		commonDir := strings.TrimSpace(string(out))
		if strings.HasSuffix(commonDir, "/.git") {
			return strings.TrimSuffix(commonDir, "/.git"), nil
		}
	}
	out, err = exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// TicketDir returns .autoflow/ticket/<KEY>/ under root. Does not create it.
func TicketDir(root, key string) string {
	return filepath.Join(root, ".autoflow", "ticket", key)
}

// TicketStateDir returns .autoflow/ticket/<KEY>/state/ under root. Does not
// create it.
func TicketStateDir(root, key string) string {
	return filepath.Join(TicketDir(root, key), "state")
}

// ProgressFile returns the path to workflow-progress.json for a ticket.
func ProgressFile(root, key string) string {
	return filepath.Join(TicketDir(root, key), "workflow-progress.json")
}
