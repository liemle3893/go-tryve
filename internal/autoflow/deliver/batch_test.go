package deliver

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/autoflow/state"
)

func mkPlanState(done ...string) *state.PlanState {
	ps := &state.PlanState{Tasks: map[string]state.TaskRecord{}}
	for _, id := range done {
		ps.Tasks[id] = state.TaskRecord{Status: state.TaskDone, Commit: "abc"}
	}
	return ps
}

func TestNextBatch_Independent(t *testing.T) {
	plan := []Task{
		{ID: "t1"}, {ID: "t2"}, {ID: "t3"}, {ID: "t4"}, {ID: "t5"}, {ID: "t6"}, {ID: "t7"},
	}
	batch, status := NextBatch(plan, mkPlanState(), 5)
	if len(batch) != 5 {
		t.Errorf("want 5, got %d", len(batch))
	}
	if status.Total != 7 || status.Done != 0 {
		t.Errorf("status: %+v", status)
	}

	// After marking first 5 done, remaining 2 should come.
	ps := mkPlanState("t1", "t2", "t3", "t4", "t5")
	batch, status = NextBatch(plan, ps, 5)
	if len(batch) != 2 {
		t.Errorf("want 2, got %d", len(batch))
	}
	if status.Done != 5 {
		t.Errorf("done: %d", status.Done)
	}
}

func TestNextBatch_Chain(t *testing.T) {
	plan := []Task{
		{ID: "a"},
		{ID: "b", Deps: []string{"a"}},
		{ID: "c", Deps: []string{"b"}},
	}
	// Start: only a ready.
	batch, _ := NextBatch(plan, mkPlanState(), 5)
	if len(batch) != 1 || batch[0].ID != "a" {
		t.Fatalf("want [a], got %+v", batch)
	}
	// After a done: only b.
	batch, _ = NextBatch(plan, mkPlanState("a"), 5)
	if len(batch) != 1 || batch[0].ID != "b" {
		t.Fatalf("want [b], got %+v", batch)
	}
	// After a,b done: only c.
	batch, _ = NextBatch(plan, mkPlanState("a", "b"), 5)
	if len(batch) != 1 || batch[0].ID != "c" {
		t.Fatalf("want [c], got %+v", batch)
	}
	// All done.
	_, status := NextBatch(plan, mkPlanState("a", "b", "c"), 5)
	if !status.AllDone {
		t.Errorf("AllDone should be true: %+v", status)
	}
}

func TestNextBatch_DiamondFanOut(t *testing.T) {
	// a → {b, c} → d
	plan := []Task{
		{ID: "a"},
		{ID: "b", Deps: []string{"a"}},
		{ID: "c", Deps: []string{"a"}},
		{ID: "d", Deps: []string{"b", "c"}},
	}
	batch, _ := NextBatch(plan, mkPlanState("a"), 5)
	// b and c can run in parallel.
	if len(batch) != 2 {
		t.Fatalf("want 2, got %+v", batch)
	}

	// After b done but c not: d is still blocked, only c ready.
	ps := mkPlanState("a", "b")
	batch, _ = NextBatch(plan, ps, 5)
	if len(batch) != 1 || batch[0].ID != "c" {
		t.Errorf("want [c], got %+v", batch)
	}
}

func TestNextBatch_SkipsRunning(t *testing.T) {
	plan := []Task{{ID: "x"}, {ID: "y"}}
	ps := &state.PlanState{Tasks: map[string]state.TaskRecord{
		"x": {Status: state.TaskRunning},
	}}
	batch, status := NextBatch(plan, ps, 5)
	if len(batch) != 1 || batch[0].ID != "y" {
		t.Errorf("want [y], got %+v", batch)
	}
	if status.Running != 1 {
		t.Errorf("running=%d", status.Running)
	}
}

func TestNextBatch_ClampToMax(t *testing.T) {
	plan := []Task{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	batch, _ := NextBatch(plan, mkPlanState(), 2)
	if len(batch) != 2 {
		t.Errorf("want 2, got %d", len(batch))
	}
	if batch[0].ID != "a" || batch[1].ID != "b" {
		t.Errorf("not in declaration order: %+v", batch)
	}
}
