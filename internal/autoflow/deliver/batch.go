package deliver

import (
	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// MaxParallelTasks is the hard cap on executors spawned per batch.
const MaxParallelTasks = 5

// NextBatch returns the set of tasks ready to dispatch next — tasks
// that are still pending and whose deps are all marked done in ps.
// Returned slice preserves PLAN.md declaration order so the dispatches
// are stable across runs, then is truncated to max entries.
//
// The second return value is a progress summary: total tasks, how many
// are already done, and whether the plan is fully complete. Useful for
// callers that want to emit a "nothing left to do" instruction without
// re-iterating.
func NextBatch(plan []Task, ps *state.PlanState, max int) ([]Task, BatchStatus) {
	status := BatchStatus{Total: len(plan)}
	if max <= 0 {
		max = MaxParallelTasks
	}

	done := map[string]bool{}
	running := map[string]bool{}
	for id, r := range ps.Tasks {
		switch r.Status {
		case state.TaskDone:
			done[id] = true
		case state.TaskRunning:
			running[id] = true
		}
	}
	status.Done = len(done)
	status.Running = len(running)

	var ready []Task
	for _, t := range plan {
		if done[t.ID] || running[t.ID] {
			continue
		}
		rec, hasRec := ps.Tasks[t.ID]
		if hasRec && rec.Status == state.TaskFailed {
			// A failed task blocks progress — skip it and surface via
			// BatchStatus. Caller decides whether to escalate or retry.
			status.Failed++
			continue
		}
		if allDepsDone(t.Deps, done) {
			ready = append(ready, t)
			if len(ready) >= max {
				break
			}
		}
	}

	status.AllDone = status.Done == status.Total && status.Total > 0
	return ready, status
}

// BatchStatus is a compact summary of the dep graph state.
type BatchStatus struct {
	Total   int
	Done    int
	Running int
	Failed  int
	AllDone bool
}

func allDepsDone(deps []string, done map[string]bool) bool {
	for _, d := range deps {
		if !done[d] {
			return false
		}
	}
	return true
}

// TaskByID returns the task with the given id, or nil if not found.
// Used by the commit-task CLI path which only carries the id forward.
func TaskByID(plan []Task, id string) *Task {
	for i := range plan {
		if plan[i].ID == id {
			return &plan[i]
		}
	}
	return nil
}
