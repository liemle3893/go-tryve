package deliver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/liemle3893/autoflow/internal/autoflow/e2e"
	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// CommitTaskRequest bundles the parameters for a per-task commit.
type CommitTaskRequest struct {
	Root       string   // main repo root (used for state files + lock path)
	Key        string   // ticket key
	TaskID     string   // plan task id (e.g. "task-03")
	Message    string   // full commit message
	Files      []string // files to stage, relative to Worktree
	Worktree   string   // worktree dir (where git index lives)
	LockWait   time.Duration
}

// CommitTask stages the named files and creates one commit in the
// ticket's worktree, serialised across concurrent callers by a shared
// file lock on .autoflow/ticket/<KEY>/state/.commit.lock. On success,
// the task's entry in plan-tasks.json is flipped to done with the
// resulting commit SHA.
//
// Empty staging (nothing to commit) is not an error — the task is
// still marked done, with Commit="" — because agents may occasionally
// produce no diff (idempotent task, all-no-op edits). The caller's
// plan is the source of truth for "task was executed".
func CommitTask(req CommitTaskRequest) (string, error) {
	if err := state.ValidateTicketKey(req.Key); err != nil {
		return "", err
	}
	if req.TaskID == "" {
		return "", fmt.Errorf("commit-task: task id required")
	}
	if req.Worktree == "" {
		return "", fmt.Errorf("commit-task: worktree required")
	}
	if req.Message == "" {
		return "", fmt.Errorf("commit-task: message required")
	}

	wait := req.LockWait
	if wait <= 0 {
		wait = 5 * time.Minute
	}
	stateDir := state.TicketStateDir(req.Root, req.Key)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir state dir: %w", err)
	}
	lockPath := filepath.Join(stateDir, ".commit.lock")
	lock, err := e2e.Acquire(context.Background(), lockPath, wait)
	if err != nil {
		return "", fmt.Errorf("acquire commit lock: %w", err)
	}
	defer lock.Release()

	// Flip status → running is done on the executor's entry (via a
	// separate _mark-task call). We only record the commit result.

	// Stage the named files. `git add --` treats everything after as a
	// pathspec, so file names starting with `-` are safe.
	if len(req.Files) > 0 {
		addArgs := append([]string{"-C", req.Worktree, "add", "--"}, req.Files...)
		if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("git add: %w\n%s", err, string(out))
		}
	} else {
		// No explicit file list → stage all tracked+untracked changes.
		if out, err := exec.Command("git", "-C", req.Worktree, "add", "-A").CombinedOutput(); err != nil {
			return "", fmt.Errorf("git add -A: %w\n%s", err, string(out))
		}
	}

	// Detect empty-staging → skip commit but still mark done.
	diffCmd := exec.Command("git", "-C", req.Worktree, "diff", "--cached", "--quiet")
	if err := diffCmd.Run(); err == nil {
		// Exit 0 → nothing staged. Record done with no commit SHA.
		if err := state.MarkTaskDone(req.Root, req.Key, req.TaskID, ""); err != nil {
			return "", err
		}
		return "", nil
	}
	// Any non-zero (including 1 = diff present) means we have something
	// to commit. Real errors surface from the next step.

	commitCmd := exec.Command("git", "-C", req.Worktree, "commit", "-m", req.Message)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit: %w\n%s", err, string(out))
	}

	shaOut, err := exec.Command("git", "-C", req.Worktree, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD: %w", err)
	}
	sha := strings.TrimSpace(string(shaOut))

	if err := state.MarkTaskDone(req.Root, req.Key, req.TaskID, sha); err != nil {
		return sha, err
	}
	return sha, nil
}
