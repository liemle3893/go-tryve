package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TaskStatus is a plan-tasks.json per-task status value.
type TaskStatus string

const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskDone    TaskStatus = "done"
	TaskFailed  TaskStatus = "failed"
)

// PlanState is the durable per-task state for a ticket's PLAN.md run.
// Keys in Tasks are the task ids from PLAN.md (e.g. "task-01").
type PlanState struct {
	Ticket string                `json:"ticket"`
	Tasks  map[string]TaskRecord `json:"tasks"`
}

// TaskRecord captures one task's lifecycle.
type TaskRecord struct {
	Status    TaskStatus `json:"status"`
	Commit    string     `json:"commit,omitempty"`
	StartedAt string     `json:"started_at,omitempty"`
	EndedAt   string     `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// PlanTasksFile returns the canonical path of plan-tasks.json for a ticket.
func PlanTasksFile(root, key string) string {
	return filepath.Join(TicketStateDir(root, key), "plan-tasks.json")
}

// ReadPlanState returns the current plan-tasks state, or an empty state
// (nil error) when the file does not yet exist. A parse error is
// surfaced.
func ReadPlanState(root, key string) (*PlanState, error) {
	if err := ValidateTicketKey(key); err != nil {
		return nil, err
	}
	var ps PlanState
	path := PlanTasksFile(root, key)
	err := readJSON(path, &ps)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &PlanState{Ticket: key, Tasks: map[string]TaskRecord{}}, nil
		}
		return nil, err
	}
	if ps.Tasks == nil {
		ps.Tasks = map[string]TaskRecord{}
	}
	return &ps, nil
}

// WritePlanState persists the state atomically.
func WritePlanState(root, key string, ps *PlanState) error {
	if err := ValidateTicketKey(key); err != nil {
		return err
	}
	if ps.Ticket == "" {
		ps.Ticket = key
	}
	if ps.Tasks == nil {
		ps.Tasks = map[string]TaskRecord{}
	}
	return WriteJSONAtomic(PlanTasksFile(root, key), ps)
}

// MarkTaskRunning flips a task to "running" and stamps started_at if
// unset. Idempotent: re-calling on a running task is a no-op; calling
// on a done task returns ErrTaskAlreadyDone so the caller doesn't
// accidentally overwrite its commit SHA.
func MarkTaskRunning(root, key, taskID string) error {
	return updateTask(root, key, taskID, func(r *TaskRecord) error {
		if r.Status == TaskDone {
			return fmt.Errorf("%w: %s", ErrTaskAlreadyDone, taskID)
		}
		r.Status = TaskRunning
		if r.StartedAt == "" {
			r.StartedAt = nowISO8601()
		}
		r.Error = ""
		return nil
	})
}

// MarkTaskDone records a successful completion with its commit SHA.
func MarkTaskDone(root, key, taskID, commit string) error {
	return updateTask(root, key, taskID, func(r *TaskRecord) error {
		r.Status = TaskDone
		r.Commit = commit
		r.EndedAt = nowISO8601()
		r.Error = ""
		return nil
	})
}

// MarkTaskFailed records a failure with a short message. Executors may
// retry by writing pending back.
func MarkTaskFailed(root, key, taskID, errMsg string) error {
	return updateTask(root, key, taskID, func(r *TaskRecord) error {
		r.Status = TaskFailed
		r.EndedAt = nowISO8601()
		r.Error = errMsg
		return nil
	})
}

// ResetStaleRunning demotes any task marked "running" but without a
// commit SHA back to pending. Called on every controller entry to
// recover from crashed executors.
func ResetStaleRunning(root, key string) error {
	ps, err := ReadPlanState(root, key)
	if err != nil {
		return err
	}
	changed := false
	for id, r := range ps.Tasks {
		if r.Status == TaskRunning && r.Commit == "" {
			r.Status = TaskPending
			r.StartedAt = ""
			ps.Tasks[id] = r
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return WritePlanState(root, key, ps)
}

// ErrTaskAlreadyDone protects against overwriting a recorded commit.
var ErrTaskAlreadyDone = errors.New("task already done")

func updateTask(root, key, taskID string, mutate func(*TaskRecord) error) error {
	if err := ValidateTicketKey(key); err != nil {
		return err
	}
	ps, err := ReadPlanState(root, key)
	if err != nil {
		return err
	}
	rec := ps.Tasks[taskID]
	if err := mutate(&rec); err != nil {
		return err
	}
	ps.Tasks[taskID] = rec
	return WritePlanState(root, key, ps)
}

func nowISO8601() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
