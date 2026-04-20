package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadOrInitLoop_FreshCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "e2e-fix-state.json")
	s, err := readOrInitLoop(path, "PROJ-1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if s.Loop != "e2e-fix" {
		t.Errorf("loop name: got %q", s.Loop)
	}
	if s.MaxRounds != 5 {
		t.Errorf("max_rounds: got %d", s.MaxRounds)
	}
	// File created on disk.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestReadOrInitLoop_ReadsExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	body := `{"loop":"x","ticket":"PROJ-1","max_rounds":5,"rounds":[{"round":1,"timestamp":"t","status":"PASSED"}]}`
	_ = os.WriteFile(path, []byte(body), 0o644)
	s, err := readOrInitLoop(path, "PROJ-1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Rounds) != 1 || s.Rounds[0].Status != "PASSED" {
		t.Errorf("read failed: %+v", s)
	}
}

func TestWriteDiagnosis_UpdatesLastRound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := &loopState{
		Loop: "x", Ticket: "PROJ-1", MaxRounds: 5,
		Rounds: []round{{Round: 1, Status: "FAILED"}},
	}
	data, _ := json.Marshal(s)
	_ = os.WriteFile(path, data, 0o644)

	diag := []byte(`{"problems":[{"id":"P1"}],"fixes":[{"file":"a.go"}]}`)
	if err := writeDiagnosis(path, s, diag); err != nil {
		t.Fatal(err)
	}

	var reloaded loopState
	raw, _ := os.ReadFile(path)
	_ = json.Unmarshal(raw, &reloaded)
	if len(reloaded.Rounds[0].Problems) == 0 || len(reloaded.Rounds[0].Fixes) == 0 {
		t.Errorf("diagnosis not persisted: %+v", reloaded.Rounds[0])
	}
}

func TestWriteDiagnosis_InvalidJSON(t *testing.T) {
	s := &loopState{Rounds: []round{{Round: 1, Status: "FAILED"}}}
	err := writeDiagnosis("/tmp/does-not-matter", s, []byte(`not json`))
	if err == nil {
		t.Errorf("expected json parse error")
	}
}

func TestReadOrInitLoop_NaivePathDerivation(t *testing.T) {
	// Check that "coverage-review-state.json" → loop="coverage-review".
	path := filepath.Join(t.TempDir(), "coverage-review-state.json")
	s, err := readOrInitLoop(path, "PROJ-1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if s.Loop != "coverage-review" {
		t.Errorf("loop name derivation: got %q", s.Loop)
	}
}
