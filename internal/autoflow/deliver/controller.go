package deliver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liemle3893/go-tryve/internal/autoflow/state"
)

// Controller wraps the repo root and holds the step-function registry.
// Initialise via NewController. Methods are safe for sequential use;
// concurrent callers must serialise their own file access (the workflow
// state files are single-writer).
type Controller struct {
	Root string
}

// NewController returns a Controller bound to root. Callers usually pass
// the output of `git rev-parse --show-toplevel`.
func NewController(root string) *Controller {
	return &Controller{Root: root}
}

// Next returns the JSON instruction for the current step. Steps 1 and 2
// have pre-init shortcuts that work before workflow-progress.json exists.
// A missing step function is reported as an internal error — kept
// impossible by the fixed 1..13 registry.
func (c *Controller) Next(key string) (*Instruction, error) {
	if err := state.ValidateTicketKey(key); err != nil {
		return nil, err
	}
	progress, err := state.ReadProgress(c.Root, key)
	if err != nil {
		return nil, fmt.Errorf("read progress: %w", err)
	}

	if progress == nil {
		// Pre-init path: maybe task-brief already exists (step 1 just
		// completed), in which case synthesise a step 2 instruction.
		briefPath := filepath.Join(state.TicketDir(c.Root, key), "task-brief.md")
		if _, err := os.Stat(briefPath); err == nil {
			meta, _ := ParseBrief(briefPath)
			title := meta["title"]
			if title == "" {
				sidecar := filepath.Join(state.TicketDir(c.Root, key), "title.txt")
				if data, err := os.ReadFile(sidecar); err == nil {
					title = string(data)
				}
			}
			if title == "" {
				title = key
			}
			// Simulated "progress" just containing title — enough for step_02.
			titleStr := title
			instr := c.step02(key, &state.Progress{Title: &titleStr})
			instr.Step = 2
			return instr, nil
		}
		// Pure fresh start — run step 1.
		instr := c.step01(key)
		instr.Step = 1
		return instr, nil
	}

	current := progress.CurrentStep
	if current < 1 {
		current = 1
	}
	if current > state.MaxStep || len(progress.Completed) >= state.MaxStep {
		return &Instruction{
			Action:  ActionDone,
			Summary: fmt.Sprintf("Workflow complete for %s", key),
		}, nil
	}

	fn, ok := stepRegistry(c)[current]
	if !ok {
		return nil, fmt.Errorf("unknown step %d", current)
	}

	instr := fn(key, progress)
	instr.Step = current
	return instr, nil
}

// CompleteOpts are optional values extracted by the LLM when reporting
// step completion. Matches the --title / --pr-url flags on the bash
// `step-controller.py complete` command.
type CompleteOpts struct {
	Title string
	PRURL string
}

// CompleteResponse is the JSON shape printed to stdout by `deliver
// complete`. Matches the Python form 1:1.
type CompleteResponse struct {
	CompletedStep int `json:"completed_step"`
	NextStep      int `json:"next_step"`
}

// Complete marks the current step done and persists any extracted values.
// Step 1 completes BEFORE workflow-progress.json exists — in that case
// the title sidecar is written and step 2 picks it up.
func (c *Controller) Complete(key string, opts CompleteOpts) (*CompleteResponse, error) {
	if err := state.ValidateTicketKey(key); err != nil {
		return nil, err
	}
	progress, err := state.ReadProgress(c.Root, key)
	if err != nil {
		return nil, err
	}

	if progress == nil {
		// Pre-init: the LLM claims step 1 (fetch brief) is done. Verify
		// task-brief.md is actually on disk before accepting the claim,
		// otherwise the next `next` call loops back to step 1 and the
		// caller is left wondering why.
		if err := VerifyStepComplete(c.Root, key, 1, nil); err != nil {
			return nil, err
		}
		if opts.Title != "" {
			sidecar := filepath.Join(state.TicketDir(c.Root, key), "title.txt")
			if err := os.MkdirAll(filepath.Dir(sidecar), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(sidecar, []byte(opts.Title), 0o644); err != nil {
				return nil, err
			}
		}
		return &CompleteResponse{CompletedStep: 1, NextStep: 2}, nil
	}

	current := progress.CurrentStep
	// For step 11 specifically, accept --pr-url from the caller BEFORE
	// running the precondition so the same call can supply the required
	// artifact and mark the step done in one shot.
	if current == 11 && opts.PRURL != "" {
		if err := state.SetField(c.Root, key, "pr_url", opts.PRURL); err != nil {
			return nil, err
		}
		// Refresh the in-memory view so the precondition sees the update.
		progress, _ = state.ReadProgress(c.Root, key)
		// Avoid re-setting pr_url later.
		opts.PRURL = ""
	}
	if err := VerifyStepComplete(c.Root, key, current, progress); err != nil {
		return nil, err
	}
	if err := state.CompleteStep(c.Root, key, current); err != nil {
		return nil, err
	}
	if opts.Title != "" {
		if err := state.SetField(c.Root, key, "title", opts.Title); err != nil {
			return nil, err
		}
	}
	if opts.PRURL != "" {
		if err := state.SetField(c.Root, key, "pr_url", opts.PRURL); err != nil {
			return nil, err
		}
	}
	if current == 5 {
		// Record impl_plan_dir so the report step can find SUMMARY.md.
		tdir := state.TicketDir(c.Root, key)
		if err := state.SetField(c.Root, key, "impl_plan_dir", tdir); err != nil {
			return nil, err
		}
	}

	updated, err := state.ReadProgress(c.Root, key)
	if err != nil {
		return nil, err
	}
	return &CompleteResponse{
		CompletedStep: current,
		NextStep:      updated.CurrentStep,
	}, nil
}

// Init seeds workflow-progress.json for a ticket.
func (c *Controller) Init(key, worktree, branch string) error {
	if err := state.ValidateTicketKey(key); err != nil {
		return err
	}
	if worktree == "" || branch == "" {
		return errors.New("worktree and branch are required")
	}
	_, err := state.InitProgress(c.Root, key, worktree, branch, false)
	return err
}

// MarshalIndent serialises an Instruction the way the bash CLI does.
func MarshalIndent(instr *Instruction) ([]byte, error) {
	return json.MarshalIndent(instr, "", "  ")
}
