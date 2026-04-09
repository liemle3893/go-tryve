// Package executor provides the step and test execution engine.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"

	"os/exec"
)

// RunHook executes a shell command as a lifecycle hook (e.g. beforeAll, afterAll).
//
// If command is empty the call is a no-op and nil is returned immediately.
// The command is executed via "sh -c" on Unix and "cmd /C" on Windows.
// workDir sets the working directory for the subprocess; an empty string
// inherits the current process working directory.
// env is merged on top of the current process environment; keys in env
// take precedence over any identically-named inherited variable.
//
// A non-zero exit code causes an error that includes the captured stdout and
// stderr for diagnostics.
func RunHook(ctx context.Context, command, workDir string, env map[string]string) error {
	if command == "" {
		return nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	// Merge inherited environment with caller-supplied overrides.
	cmd.Env = buildHookEnv(env)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook command failed: %w\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}
	return nil
}

// buildHookEnv merges os.Environ() with the provided extra map.
// Values in extra override identically-named entries from the process environment.
func buildHookEnv(extra map[string]string) []string {
	base := os.Environ()
	if len(extra) == 0 {
		return base
	}
	env := make([]string, len(base), len(base)+len(extra))
	copy(env, base)
	for k, v := range extra {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
