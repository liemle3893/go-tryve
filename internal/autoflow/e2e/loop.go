package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// LoopOptions wraps LocalOptions with the state-file path for round
// tracking. Mirrors the surface of skills/autoflow-deliver/scripts/
// e2e-loop.sh.
type LoopOptions struct {
	Local          LocalOptions
	Ticket         string
	StateFile      string // defaults to .autoflow/ticket/<KEY>/state/e2e-fix-state.json
	MaxRounds      int    // defaults to 5
	Diagnosis      []byte // optional — problems+fixes JSON for the previous round
	SkipDiagnose   bool   // bypass the diagnosis requirement for round 2+
	DiagnoseOnFail bool   // (unused — placeholder for future "auto-diagnose" mode)
}

// LoopResult is returned by RunLoop — exposes the round number run and
// the final status so callers can decide whether to dispatch a fixer.
type LoopResult struct {
	Round  int
	Status string // "PASSED" | "FAILED"
	Output string // output file location
}

// ErrDiagnoseRequired is returned when round N+1 starts without
// diagnostic input for round N having been supplied.
var ErrDiagnoseRequired = errors.New("round N+1 requires --diagnose (problems and fixes from round N)")

// round is the per-round state written to StateFile.
type round struct {
	Round      int             `json:"round"`
	Timestamp  string          `json:"timestamp"`
	Status     string          `json:"status"`
	OutputFile string          `json:"output_file,omitempty"`
	Problems   json.RawMessage `json:"problems,omitempty"`
	Fixes      json.RawMessage `json:"fixes,omitempty"`
}

// loopState is the disk shape used by both Go and the bash script, kept
// compatible so co-existing scripts continue to read each other's files.
type loopState struct {
	Loop      string  `json:"loop"`
	Ticket    string  `json:"ticket"`
	MaxRounds int     `json:"max_rounds"`
	Rounds    []round `json:"rounds"`
}

// RunLoop performs one iteration of the fix-and-retry loop. It:
//
//  1. Resolves defaults for StateFile + MaxRounds.
//  2. Loads the current state and counts rounds.
//  3. Rejects the call if max_rounds has been reached.
//  4. Writes Diagnosis onto the PREVIOUS round (when supplied).
//  5. Calls RunLocal and records a new round with PASSED/FAILED.
func RunLoop(ctx context.Context, opts LoopOptions) (*LoopResult, error) {
	if err := state.ValidateTicketKey(opts.Ticket); err != nil {
		return nil, err
	}
	if opts.MaxRounds == 0 {
		opts.MaxRounds = 5
	}
	if opts.StateFile == "" {
		opts.StateFile = filepath.Join(
			state.TicketStateDir(opts.Local.WorkDir, opts.Ticket),
			"e2e-fix-state.json",
		)
	}

	s, err := readOrInitLoop(opts.StateFile, opts.Ticket, opts.MaxRounds)
	if err != nil {
		return nil, err
	}
	current := len(s.Rounds) + 1
	if current > opts.MaxRounds {
		return nil, fmt.Errorf("max rounds %d exceeded — cannot run round %d",
			opts.MaxRounds, current)
	}

	if current > 1 {
		last := s.Rounds[len(s.Rounds)-1]
		if last.Status == "FAILED" && len(opts.Diagnosis) == 0 && !opts.SkipDiagnose {
			return nil, ErrDiagnoseRequired
		}
		if len(opts.Diagnosis) > 0 {
			if err := writeDiagnosis(opts.StateFile, s, opts.Diagnosis); err != nil {
				return nil, err
			}
			s, err = readOrInitLoop(opts.StateFile, opts.Ticket, opts.MaxRounds)
			if err != nil {
				return nil, err
			}
		}
	}

	outFile := opts.Local.OutputFile
	if outFile == "" {
		outFile = filepath.Join(os.TempDir(),
			"e2e-results-"+opts.Ticket+".txt")
		opts.Local.OutputFile = outFile
	}

	result, runErr := RunLocal(ctx, opts.Local)

	status := "PASSED"
	if runErr != nil || (result != nil && result.Outcome != nil && result.Outcome.Failed > 0) {
		status = "FAILED"
	}

	newRound := round{
		Round:      current,
		Timestamp:  time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Status:     status,
		OutputFile: outFile,
		Problems:   json.RawMessage(`[]`),
		Fixes:      json.RawMessage(`[]`),
	}
	s.Rounds = append(s.Rounds, newRound)
	if err := state.WriteJSONAtomic(opts.StateFile, s); err != nil {
		return nil, err
	}

	return &LoopResult{Round: current, Status: status, Output: outFile}, nil
}

// ReadLoopState returns the loop state for inspection. Missing file yields
// (nil, nil).
func ReadLoopState(path string) (*loopState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var s loopState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &s, nil
}

func readOrInitLoop(path, ticket string, maxRounds int) (*loopState, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		loopName := "e2e-fix"
		if base := filepath.Base(path); base != "" {
			trimmed := base
			if ext := filepath.Ext(trimmed); ext != "" {
				trimmed = trimmed[:len(trimmed)-len(ext)]
			}
			// Strip trailing "-state" so "e2e-fix-state.json" → "e2e-fix".
			if n := len(trimmed); n > 6 && trimmed[n-6:] == "-state" {
				trimmed = trimmed[:n-6]
			}
			loopName = trimmed
		}
		s := &loopState{
			Loop:      loopName,
			Ticket:    ticket,
			MaxRounds: maxRounds,
			Rounds:    []round{},
		}
		if err := state.WriteJSONAtomic(path, s); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var s loopState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &s, nil
}

func writeDiagnosis(path string, s *loopState, diagnosis []byte) error {
	var dx struct {
		Problems json.RawMessage `json:"problems"`
		Fixes    json.RawMessage `json:"fixes"`
	}
	if err := json.Unmarshal(diagnosis, &dx); err != nil {
		return fmt.Errorf("parse diagnosis json: %w", err)
	}
	if len(s.Rounds) == 0 {
		return errors.New("cannot write diagnosis: no rounds recorded yet")
	}
	last := &s.Rounds[len(s.Rounds)-1]
	if len(dx.Problems) > 0 {
		last.Problems = dx.Problems
	}
	if len(dx.Fixes) > 0 {
		last.Fixes = dx.Fixes
	}
	return state.WriteJSONAtomic(path, s)
}
