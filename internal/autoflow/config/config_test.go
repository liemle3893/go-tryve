package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRead_MissingReturnsZero(t *testing.T) {
	c, err := Read(t.TempDir())
	if err != nil {
		t.Fatalf("Read on missing: %v", err)
	}
	if c == nil {
		t.Fatal("want non-nil zero Config")
	}
	if c.CodingAgent != "" || c.Sandbox.Enabled || c.Sandbox.Name != "" {
		t.Errorf("want zero-value, got %+v", c)
	}
}

func TestSetGet_RoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := Set(root, "coding_agent", "claude"); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, "sandbox.enabled", "true"); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, "sandbox.name", "my-sbx"); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, "sandbox.policy", "strict"); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, "sandbox.extra_mounts", "/a:/a,/b:/b"); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"coding_agent":         "claude",
		"sandbox.enabled":      "true",
		"sandbox.name":         "my-sbx",
		"sandbox.policy":       "strict",
		"sandbox.extra_mounts": "/a:/a,/b:/b",
	}
	for f, want := range cases {
		got, err := Get(root, f)
		if err != nil {
			t.Errorf("Get(%s): %v", f, err)
		}
		if got != want {
			t.Errorf("Get(%s)=%q, want %q", f, got, want)
		}
	}
}

func TestSet_InvalidAgent(t *testing.T) {
	root := t.TempDir()
	err := Set(root, "coding_agent", "gpt4")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("want ErrInvalidValue, got %v", err)
	}
	// File should NOT have been created.
	if _, err := os.Stat(Path(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("config written despite invalid value: %v", err)
	}
}

func TestSet_UnknownField(t *testing.T) {
	err := Set(t.TempDir(), "bogus", "x")
	if !errors.Is(err, ErrUnknownField) {
		t.Errorf("want ErrUnknownField, got %v", err)
	}
}

func TestSet_InvalidBool(t *testing.T) {
	err := Set(t.TempDir(), "sandbox.enabled", "maybe")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("want ErrInvalidValue, got %v", err)
	}
}

func TestDel_WholeFile(t *testing.T) {
	root := t.TempDir()
	if err := Set(root, "coding_agent", "copilot"); err != nil {
		t.Fatal(err)
	}
	if err := Del(root, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file not removed: %v", err)
	}
	// Idempotent.
	if err := Del(root, ""); err != nil {
		t.Errorf("Del on missing: %v", err)
	}
}

func TestDel_Field(t *testing.T) {
	root := t.TempDir()
	_ = Set(root, "coding_agent", "claude")
	_ = Set(root, "sandbox.name", "x")
	if err := Del(root, "sandbox.name"); err != nil {
		t.Fatal(err)
	}
	got, _ := Get(root, "sandbox.name")
	if got != "" {
		t.Errorf("sandbox.name not cleared, got %q", got)
	}
	agent, _ := Get(root, "coding_agent")
	if agent != "claude" {
		t.Errorf("Del clobbered coding_agent: %q", agent)
	}
}

func TestPath(t *testing.T) {
	got := Path("/r")
	want := filepath.Join("/r", ".autoflow", "config.json")
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestShow(t *testing.T) {
	root := t.TempDir()
	_ = Set(root, "coding_agent", "claude")
	out, err := Show(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Errorf("Show should end in newline")
	}
}

func TestParseList_EmptyClears(t *testing.T) {
	root := t.TempDir()
	_ = Set(root, "sandbox.extra_mounts", "/a,/b")
	_ = Set(root, "sandbox.extra_mounts", "")
	got, _ := Get(root, "sandbox.extra_mounts")
	if got != "" {
		t.Errorf("want empty, got %q", got)
	}
}
