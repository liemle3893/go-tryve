package worktree

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadIncludes_ParsesAndSkipsCommentsBlanks(t *testing.T) {
	dir := t.TempDir()
	body := "" +
		"# a comment\n" +
		"\n" +
		"  config/local.yaml  \n" +
		"certs/\n" +
		"# trailing comment\n" +
		"scripts/dev.sh\n"
	if err := os.WriteFile(filepath.Join(dir, IncludeFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadIncludes(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"config/local.yaml", "certs", "scripts/dev.sh"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadIncludes_MissingFileReturnsNil(t *testing.T) {
	got, err := ReadIncludes(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestReadIncludes_RejectsEscapes(t *testing.T) {
	for _, bad := range []string{"/etc/passwd", "../outside"} {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, IncludeFile), []byte(bad+"\n"), 0o644)
		if _, err := ReadIncludes(dir); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestCopyIncludes_FilesAndDirs(t *testing.T) {
	main := t.TempDir()
	work := t.TempDir()

	// File entry.
	_ = os.MkdirAll(filepath.Join(main, "config"), 0o755)
	_ = os.WriteFile(filepath.Join(main, "config", "local.yaml"), []byte("x: 1"), 0o644)

	// Directory entry with nested content.
	_ = os.MkdirAll(filepath.Join(main, "certs", "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(main, "certs", "root.pem"), []byte("PEM"), 0o644)
	_ = os.WriteFile(filepath.Join(main, "certs", "sub", "leaf.pem"), []byte("LEAF"), 0o644)

	// Missing entry (should SKIP).
	includes := []string{"config/local.yaml", "certs", "not-there.txt"}

	if err := copyIncludes(main, work, includes, io.Discard); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{
		"config/local.yaml",
		"certs/root.pem",
		"certs/sub/leaf.pem",
	} {
		if _, err := os.Stat(filepath.Join(work, p)); err != nil {
			t.Errorf("expected %s in worktree, got %v", p, err)
		}
	}
	// Missing entry shouldn't materialise.
	if _, err := os.Stat(filepath.Join(work, "not-there.txt")); err == nil {
		t.Error("missing entry should not be created in worktree")
	}
}

func TestCopyIncludes_ContentsPreservedExactly(t *testing.T) {
	main := t.TempDir()
	work := t.TempDir()
	payload := strings.Repeat("abc\n", 200)
	_ = os.WriteFile(filepath.Join(main, "data.txt"), []byte(payload), 0o644)

	if err := copyIncludes(main, work, []string{"data.txt"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(work, "data.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Errorf("content mismatch: want %d bytes, got %d", len(payload), len(got))
	}
}
