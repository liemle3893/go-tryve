// Package sandbox wraps the `sbx` CLI so autoflow can bootstrap and inspect
// the per-repo sandbox in which coding agents run. All shell-outs go through
// the Runner interface to keep tests hermetic.
package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Runner abstracts command execution so unit tests can inject a fake without
// actually shelling out to `sbx` or `go`.
type Runner interface {
	// Run executes name with args. stdin may be nil. Combined stdout+stderr is
	// captured to stdout (when non-nil); stderr additionally gets a duplicate
	// stream.
	Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// ExecRunner is the default production Runner using os/exec.
type ExecRunner struct{}

// Run implements Runner via exec.CommandContext.
func (ExecRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// captureRun runs and returns stdout as a trimmed string. stderr is discarded.
func captureRun(ctx context.Context, r Runner, name string, args ...string) (string, error) {
	var out, errBuf bytes.Buffer
	if err := r.Run(ctx, name, args, nil, &out, &errBuf); err != nil {
		return "", fmt.Errorf("%s %v: %w (stderr: %s)", name, args, err, errBuf.String())
	}
	return out.String(), nil
}
