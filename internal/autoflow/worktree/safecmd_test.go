package worktree

import (
	"bytes"
	"errors"
	"testing"
)

type allowAll struct{}

func (allowAll) AllowUnknownCommand(_, _, _ string) (bool, error) { return true, nil }

type denyAll struct{}

func (denyAll) AllowUnknownCommand(_, _, _ string) (bool, error) { return false, nil }

func TestFirstTokenBinary(t *testing.T) {
	cases := map[string]string{
		"go mod download":        "go",
		"/usr/local/bin/go test": "go",
		"  npm ci":               "npm",
		"":                       "",
	}
	for in, want := range cases {
		if got := firstTokenBinary(in); got != want {
			t.Errorf("firstTokenBinary(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRunSafeCmd_AllowlistedExecutes(t *testing.T) {
	var out bytes.Buffer
	if err := RunSafeCmd("install", "go version", ".", NonInteractivePrompter{}, &out, &out); err != nil {
		t.Fatalf("go version should succeed: %v", err)
	}
	if out.Len() == 0 {
		t.Errorf("expected go version output")
	}
}

func TestRunSafeCmd_UnknownNonInteractiveSkipped(t *testing.T) {
	err := RunSafeCmd("install", "somecommand --flag", ".", NonInteractivePrompter{}, nil, nil)
	if !errors.Is(err, ErrSkipped) {
		t.Errorf("want ErrSkipped, got %v", err)
	}
}

func TestRunSafeCmd_UnknownApprovedExecutes(t *testing.T) {
	// "true" is a unix builtin — virtually always present and a no-op.
	err := RunSafeCmd("install", "true", ".", allowAll{}, nil, nil)
	if err != nil {
		t.Errorf("approved unknown should execute: %v", err)
	}
}

func TestRunSafeCmd_UnknownDeniedSkipped(t *testing.T) {
	err := RunSafeCmd("install", "nope", ".", denyAll{}, nil, nil)
	if !errors.Is(err, ErrSkipped) {
		t.Errorf("want ErrSkipped, got %v", err)
	}
}

func TestRunSafeCmd_EmptyCommand(t *testing.T) {
	if err := RunSafeCmd("install", "  ", ".", NonInteractivePrompter{}, nil, nil); err == nil {
		t.Errorf("expected error on empty command")
	}
}
