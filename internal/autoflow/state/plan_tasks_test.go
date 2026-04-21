package state

import "testing"

func TestPlanTasks_LifecycleTransitions(t *testing.T) {
	root := t.TempDir()
	key := "PROJ-1"
	_, _ = InitProgress(root, key, "/wt", "b", false)

	if err := MarkTaskRunning(root, key, "task-01"); err != nil {
		t.Fatal(err)
	}
	ps, _ := ReadPlanState(root, key)
	if ps.Tasks["task-01"].Status != TaskRunning {
		t.Fatalf("not running: %+v", ps.Tasks["task-01"])
	}
	if ps.Tasks["task-01"].StartedAt == "" {
		t.Error("started_at not set")
	}

	if err := MarkTaskDone(root, key, "task-01", "abc123"); err != nil {
		t.Fatal(err)
	}
	ps, _ = ReadPlanState(root, key)
	if ps.Tasks["task-01"].Status != TaskDone || ps.Tasks["task-01"].Commit != "abc123" {
		t.Errorf("done/commit wrong: %+v", ps.Tasks["task-01"])
	}

	// Re-marking done is rejected-by-running because we treat done as
	// terminal. MarkTaskRunning on a done task must return an error.
	err := MarkTaskRunning(root, key, "task-01")
	if err == nil {
		t.Error("expected error flipping done→running")
	}
}

func TestResetStaleRunning(t *testing.T) {
	root := t.TempDir()
	key := "PROJ-1"
	_, _ = InitProgress(root, key, "/wt", "b", false)

	// Seed a "running" task with no commit — simulating a crashed executor.
	ps := &PlanState{Ticket: key, Tasks: map[string]TaskRecord{
		"task-01": {Status: TaskRunning, StartedAt: "2024-01-01T00:00:00Z"},
		"task-02": {Status: TaskDone, Commit: "abc"},
	}}
	_ = WritePlanState(root, key, ps)

	if err := ResetStaleRunning(root, key); err != nil {
		t.Fatal(err)
	}
	got, _ := ReadPlanState(root, key)
	if got.Tasks["task-01"].Status != TaskPending {
		t.Errorf("stale running not reset: %+v", got.Tasks["task-01"])
	}
	if got.Tasks["task-02"].Status != TaskDone {
		t.Errorf("done task should not be touched: %+v", got.Tasks["task-02"])
	}
}
