package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// withTempDir changes the working directory to a fresh temp dir for the
// duration of the test and restores it afterwards.
func withTempDir(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// TestSaveLoadFailedNames_RoundTrip verifies that failed test names written by
// saveFailedNames are correctly read back by loadFailedNames.
func TestSaveLoadFailedNames_RoundTrip(t *testing.T) {
	withTempDir(t)

	suite := &tryve.SuiteResult{
		Failed: 2,
		Tests: []tryve.TestResult{
			{Test: &tryve.TestDefinition{Name: "TC-001"}, Status: tryve.StatusFailed},
			{Test: &tryve.TestDefinition{Name: "TC-002"}, Status: tryve.StatusPassed},
			{Test: &tryve.TestDefinition{Name: "TC-003"}, Status: tryve.StatusFailed},
		},
	}

	if err := saveFailedNames(suite); err != nil {
		t.Fatalf("saveFailedNames: %v", err)
	}

	names, err := loadFailedNames()
	if err != nil {
		t.Fatalf("loadFailedNames: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 failed names, got %d: %v", len(names), names)
	}
	for _, want := range []string{"TC-001", "TC-003"} {
		if _, ok := names[want]; !ok {
			t.Errorf("expected %q in failed names", want)
		}
	}
	if _, ok := names["TC-002"]; ok {
		t.Error("TC-002 (passed) should not be in failed names")
	}
}

// TestSaveFailedNames_RemovesFileOnNoFailures verifies that if the suite has
// no failures, saveFailedNames removes the stale file.
func TestSaveFailedNames_RemovesFileOnNoFailures(t *testing.T) {
	withTempDir(t)

	// Create a stale file from a prior run.
	if err := os.WriteFile(failedNamesFile, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	suite := &tryve.SuiteResult{Failed: 0}
	if err := saveFailedNames(suite); err != nil {
		t.Fatalf("saveFailedNames: %v", err)
	}

	if _, err := os.Stat(failedNamesFile); !os.IsNotExist(err) {
		t.Error("expected .tryve-failed to be removed when there are no failures")
	}
}

// TestLoadFailedNames_MissingFile verifies that loadFailedNames returns nil
// (not an error) when no file exists yet.
func TestLoadFailedNames_MissingFile(t *testing.T) {
	withTempDir(t)

	names, err := loadFailedNames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if names != nil {
		t.Errorf("expected nil map when file is absent, got %v", names)
	}
}

// TestSaveFailedNames_NilResult verifies that a nil result is handled safely.
func TestSaveFailedNames_NilResult(t *testing.T) {
	withTempDir(t)

	if err := saveFailedNames(nil); err != nil {
		t.Fatalf("unexpected error for nil result: %v", err)
	}
}

// TestSaveFailedNames_FileLocation verifies that the file is written in the
// current working directory under the expected name.
func TestSaveFailedNames_FileLocation(t *testing.T) {
	withTempDir(t)

	suite := &tryve.SuiteResult{
		Failed: 1,
		Tests: []tryve.TestResult{
			{Test: &tryve.TestDefinition{Name: "TC-X"}, Status: tryve.StatusFailed},
		},
	}
	if err := saveFailedNames(suite); err != nil {
		t.Fatalf("saveFailedNames: %v", err)
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, failedNamesFile)
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file at %s: %v", expected, err)
	}
}
