package adapter_test

import (
	"context"
	"testing"

	"github.com/liemle3893/e2e-runner/internal/adapter"
)

// TestShellAdapter_ExecSimple verifies that a basic echo command produces the
// expected stdout output and exits with code 0.
func TestShellAdapter_ExecSimple(t *testing.T) {
	a := adapter.NewShellAdapter(&adapter.ShellConfig{})
	ctx := context.Background()

	result, err := a.Execute(ctx, "exec", map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil StepResult")
	}

	stdout, ok := result.Data["stdout"].(string)
	if !ok {
		t.Fatalf("stdout not a string: %v", result.Data["stdout"])
	}
	if stdout != "hello\n" {
		t.Fatalf("expected stdout %q, got %q", "hello\n", stdout)
	}

	exitCode, ok := result.Data["exitCode"].(float64)
	if !ok {
		t.Fatalf("exitCode not a float64: %v", result.Data["exitCode"])
	}
	if exitCode != 0 {
		t.Fatalf("expected exitCode 0, got %v", exitCode)
	}
}

// TestShellAdapter_ExecWithEnv verifies that custom environment variables are
// passed to the command and accessible via the shell.
func TestShellAdapter_ExecWithEnv(t *testing.T) {
	a := adapter.NewShellAdapter(&adapter.ShellConfig{})
	ctx := context.Background()

	result, err := a.Execute(ctx, "exec", map[string]any{
		"command": "echo $MY_CUSTOM_VAR",
		"env": map[string]any{
			"MY_CUSTOM_VAR": "test_value_123",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil StepResult")
	}

	stdout, ok := result.Data["stdout"].(string)
	if !ok {
		t.Fatalf("stdout not a string: %v", result.Data["stdout"])
	}
	if stdout != "test_value_123\n" {
		t.Fatalf("expected stdout %q, got %q", "test_value_123\n", stdout)
	}
}

// TestShellAdapter_ExecFailure verifies that a non-zero exit code is returned
// in the result data without an adapter error (exit code handling is done by
// the step executor, not the adapter).
func TestShellAdapter_ExecFailure(t *testing.T) {
	a := adapter.NewShellAdapter(&adapter.ShellConfig{})
	ctx := context.Background()

	result, err := a.Execute(ctx, "exec", map[string]any{
		"command": "exit 1",
	})
	if err != nil {
		t.Fatalf("expected nil error for non-zero exit, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil StepResult")
	}

	exitCode, ok := result.Data["exitCode"].(float64)
	if !ok {
		t.Fatalf("exitCode not a float64: %v", result.Data["exitCode"])
	}
	if exitCode != 1 {
		t.Fatalf("expected exitCode 1, got %v", exitCode)
	}
}

// TestShellAdapter_InvalidAction verifies that an unsupported action name
// returns an error.
func TestShellAdapter_InvalidAction(t *testing.T) {
	a := adapter.NewShellAdapter(&adapter.ShellConfig{})
	ctx := context.Background()

	_, err := a.Execute(ctx, "unknown", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unknown action, got nil")
	}
}
