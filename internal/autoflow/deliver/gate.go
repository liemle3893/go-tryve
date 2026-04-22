package deliver

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// buildGateState is the JSON shape of build-gate-state.json. Matches the
// bash writer in step-controller.py _gate-result.
type buildGateState struct {
	Attempt       int    `json:"attempt"`
	LastResult    string `json:"last_result"` // "pass" | "fail" | "pending"
	ErrorFile     string `json:"error_file"`
	FixDispatched bool   `json:"fix_dispatched"`
}

// GateResult writes the build-gate state for an attempt. On failure,
// copies the tail of logPath into a dedicated error file so the fixer
// subagent has a focused log to read. Mirrors cmd_gate_result from
// step-controller.py.
func GateResult(root, ticket string, attempt, exitCode int, logPath string) error {
	if err := state.ValidateTicketKey(ticket); err != nil {
		return err
	}
	stateDir := state.TicketStateDir(root, ticket)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	gateFile := filepath.Join(stateDir, "build-gate-state.json")

	errorFile := ""
	if exitCode != 0 {
		tailPath := filepath.Join(stateDir, formatAttemptFile(attempt))
		if err := writeLogTail(logPath, tailPath, 100); err != nil {
			// Fallback: point the fixer at the full log so it has SOMETHING.
			errorFile = logPath
		} else {
			errorFile = tailPath
		}
	}

	result := "pass"
	if exitCode != 0 {
		result = "fail"
	}
	gs := buildGateState{
		Attempt:       attempt,
		LastResult:    result,
		ErrorFile:     errorFile,
		FixDispatched: false,
	}
	return state.WriteJSONAtomic(gateFile, gs)
}

// readBuildGate returns the current build-gate state or a default
// "pending" state when the file is absent.
func readBuildGate(root, ticket string) (*buildGateState, error) {
	path := filepath.Join(state.TicketStateDir(root, ticket), "build-gate-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &buildGateState{LastResult: "pending"}, nil
		}
		return nil, err
	}
	var gs buildGateState
	if err := json.Unmarshal(data, &gs); err != nil {
		return nil, err
	}
	return &gs, nil
}

// writeBuildGate replaces the gate file atomically.
func writeBuildGate(root, ticket string, gs *buildGateState) error {
	path := filepath.Join(state.TicketStateDir(root, ticket), "build-gate-state.json")
	return state.WriteJSONAtomic(path, gs)
}

func formatAttemptFile(attempt int) string {
	return "build-gate-error-" + itoa(attempt) + ".log"
}

// writeLogTail copies the last maxLines of src into dst. When src has
// fewer lines than maxLines, dst is a copy of src.
func writeLogTail(src, dst string, maxLines int) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	lines := make([]string, 0, maxLines)
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for s.Scan() {
		lines = append(lines, s.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// itoa: minimal integer → string helper avoiding an import of strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return sign + string(b)
}
