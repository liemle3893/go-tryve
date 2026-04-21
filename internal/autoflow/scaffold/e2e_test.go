package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_FreshCreatesN(t *testing.T) {
	root := t.TempDir()
	results, err := Generate(Options{Root: root, Ticket: "PROJ-1", Area: "user-api", Count: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("want 3 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Created {
			t.Errorf("result %d not created", i)
		}
		if _, err := os.Stat(r.Path); err != nil {
			t.Errorf("file missing: %v", err)
		}
	}
	// First file should be TC-PROJ-1-001-STUB.test.yaml
	want := filepath.Join(root, "tests", "e2e", "user-api", "TC-PROJ-1-001-STUB.test.yaml")
	if results[0].Path != want {
		t.Errorf("first path: got %q want %q", results[0].Path, want)
	}
}

func TestGenerate_ContinuesNumbering(t *testing.T) {
	root := t.TempDir()
	// Seed two pre-existing files with mixed forms (stub + finished).
	dir := filepath.Join(root, "tests", "e2e", "pay")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "TC-PROJ-2-005-STUB.test.yaml"), []byte("stub"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "TC-PROJ-2-012.test.yaml"), []byte("done"), 0o644)

	results, err := Generate(Options{Root: root, Ticket: "PROJ-2", Area: "pay", Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(results[0].Path, "TC-PROJ-2-013-STUB.test.yaml") {
		t.Errorf("continue from 12, got %q", results[0].Path)
	}
}

func TestGenerate_RejectsBadArea(t *testing.T) {
	root := t.TempDir()
	bad := []string{"", "../escape", "a/b", `a\b`, ".", ".."}
	for _, area := range bad {
		_, err := Generate(Options{Root: root, Ticket: "PROJ-1", Area: area, Count: 1})
		if err == nil {
			t.Errorf("area=%q should be rejected", area)
		}
	}
}

func TestGenerate_RejectsInvalidTicket(t *testing.T) {
	_, err := Generate(Options{Root: t.TempDir(), Ticket: "../PROJ-1", Area: "x", Count: 1})
	if err == nil {
		t.Errorf("invalid ticket not rejected")
	}
}

func TestGenerate_ZeroCount(t *testing.T) {
	_, err := Generate(Options{Root: t.TempDir(), Ticket: "PROJ-1", Area: "x", Count: 0})
	if err == nil {
		t.Errorf("count=0 should error")
	}
}

func TestGenerate_TemplateRenderedCorrectly(t *testing.T) {
	root := t.TempDir()
	res, err := Generate(Options{Root: root, Ticket: "PROJ-1", Area: "user-api", Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(res[0].Path)
	content := string(data)

	// Name + tags + placeholder should reflect the inputs.
	if !strings.Contains(content, "name: TC-PROJ-1-001-STUB") {
		t.Errorf("name placeholder not substituted")
	}
	if !strings.Contains(content, "- user-api") || !strings.Contains(content, "- PROJ-1") {
		t.Errorf("tags not substituted, content: %s", content)
	}
	// The Mustache-style {{baseUrl}} must survive un-mangled.
	if !strings.Contains(content, "{{baseUrl}}") {
		t.Errorf("Mustache placeholder lost, content: %s", content)
	}
	if !strings.Contains(content, "{{captured.access_token}}") {
		t.Errorf("Mustache capture ref lost")
	}
}

func TestNextNumber(t *testing.T) {
	dir := t.TempDir()
	if got := nextNumber(dir, "PROJ-1"); got != 0 {
		t.Errorf("empty dir should return 0, got %d", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "TC-PROJ-1-007-STUB.test.yaml"), []byte(""), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "TC-PROJ-1-003.test.yaml"), []byte(""), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "TC-OTHER-999.test.yaml"), []byte(""), 0o644) // wrong ticket
	if got := nextNumber(dir, "PROJ-1"); got != 7 {
		t.Errorf("want 7, got %d", got)
	}
}

func TestGenerate_WrapsErrorsSurface(t *testing.T) {
	// Sanity: writing into an unwritable path returns an error.
	root := t.TempDir()
	bogus := filepath.Join(root, "tests", "e2e", "x")
	_ = os.MkdirAll(bogus, 0o555)
	defer os.Chmod(bogus, 0o755)
	_, err := Generate(Options{Root: root, Ticket: "PROJ-1", Area: "x", Count: 1})
	if err == nil {
		t.Skipf("expected a permission error, got none; likely running as root")
	}
	if errors.Is(err, ErrBadArea) {
		t.Errorf("should not be ErrBadArea: %v", err)
	}
}
