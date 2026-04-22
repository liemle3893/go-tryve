package deliver

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"},
		{"commit", "--allow-empty", "-m", "seed"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// TestCommitTask_SerialisesConcurrentCallers runs two goroutines that
// both try to commit at the same time. The flock inside CommitTask is
// the only thing preventing them from racing on the git index — if it
// breaks, the second call would see "index.lock" errors or overwrite
// the first's commit. Assertion: both commits land with distinct SHAs.
func TestCommitTask_SerialisesConcurrentCallers(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	gitInit(t, root)

	// Set up ticket directory skeleton.
	key := "PROJ-1"
	_, _ = state.InitProgress(root, key, root, "main", false)
	_ = os.MkdirAll(state.TicketStateDir(root, key), 0o755)

	// Seed two plan tasks.
	ps := &state.PlanState{Ticket: key, Tasks: map[string]state.TaskRecord{
		"task-01": {Status: state.TaskPending},
		"task-02": {Status: state.TaskPending},
	}}
	if err := state.WritePlanState(root, key, ps); err != nil {
		t.Fatal(err)
	}

	// Two disjoint files, each modified before CommitTask runs.
	f1 := filepath.Join(root, "a.txt")
	f2 := filepath.Join(root, "b.txt")
	_ = os.WriteFile(f1, []byte("hello"), 0o644)
	_ = os.WriteFile(f2, []byte("world"), 0o644)

	var wg sync.WaitGroup
	shas := make([]string, 2)
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		shas[0], errs[0] = CommitTask(CommitTaskRequest{
			Root: root, Key: key, TaskID: "task-01",
			Message: "task-01: add a", Files: []string{"a.txt"}, Worktree: root,
		})
	}()
	go func() {
		defer wg.Done()
		shas[1], errs[1] = CommitTask(CommitTaskRequest{
			Root: root, Key: key, TaskID: "task-02",
			Message: "task-02: add b", Files: []string{"b.txt"}, Worktree: root,
		})
	}()
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	if shas[0] == "" || shas[1] == "" {
		t.Fatalf("expected two commits, got %v", shas)
	}
	if shas[0] == shas[1] {
		t.Errorf("both goroutines produced the same SHA: %s", shas[0])
	}

	// Both tasks should be marked done with their commit SHAs.
	got, _ := state.ReadPlanState(root, key)
	for _, id := range []string{"task-01", "task-02"} {
		r := got.Tasks[id]
		if r.Status != state.TaskDone || r.Commit == "" {
			t.Errorf("%s not marked done: %+v", id, r)
		}
	}
}

func TestCommitTask_EmptyStagingMarksDoneWithoutCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	gitInit(t, root)
	// Keep .autoflow/ out of the working tree so git add -A inside
	// CommitTask doesn't pick up state files / lock files.
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".autoflow/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", ".gitignore"}, {"commit", "-m", "gitignore"}} {
		c := exec.Command("git", args...)
		c.Dir = root
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("seed: %v\n%s", err, out)
		}
	}
	key := "PROJ-1"
	_, _ = state.InitProgress(root, key, root, "main", false)

	sha, err := CommitTask(CommitTaskRequest{
		Root: root, Key: key, TaskID: "task-01",
		Message: "task-01: no-op", Files: nil, Worktree: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sha != "" {
		t.Errorf("expected empty sha for no-op, got %q", sha)
	}
	ps, _ := state.ReadPlanState(root, key)
	if ps.Tasks["task-01"].Status != state.TaskDone {
		t.Errorf("task not marked done: %+v", ps.Tasks["task-01"])
	}
}
