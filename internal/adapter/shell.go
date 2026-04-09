package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// ShellConfig holds configuration options for the ShellAdapter.
type ShellConfig struct {
	// DefaultTimeout is the default command timeout in milliseconds (0 = no default).
	DefaultTimeout int
	// DefaultCwd is the default working directory for commands (empty = inherit).
	DefaultCwd string
}

// ShellAdapter executes shell commands via os/exec.
// Connect and Close are no-ops because shell execution requires no persistent
// connection state.
type ShellAdapter struct {
	config *ShellConfig
}

// NewShellAdapter constructs a ShellAdapter with the given configuration.
// config must not be nil.
func NewShellAdapter(config *ShellConfig) *ShellAdapter {
	if config == nil {
		config = &ShellConfig{}
	}
	return &ShellAdapter{config: config}
}

// Name returns the adapter's registered identifier.
func (a *ShellAdapter) Name() string { return "shell" }

// Connect is a no-op for the shell adapter; shell execution requires no
// persistent connection.
func (a *ShellAdapter) Connect(_ context.Context) error { return nil }

// Close is a no-op for the shell adapter.
func (a *ShellAdapter) Close(_ context.Context) error { return nil }

// Health is a no-op for the shell adapter; shell execution is always available
// when the OS is reachable.
func (a *ShellAdapter) Health(_ context.Context) error { return nil }

// Execute runs the named action with the provided parameters.
// Only the "exec" action is supported.
func (a *ShellAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	if action != "exec" {
		return nil, tryve.AdapterError(
			"shell",
			action,
			fmt.Sprintf("unsupported action %q; only \"exec\" is supported", action),
			nil,
		)
	}
	return a.execAction(ctx, params)
}

// execAction implements the "exec" action: run a shell command and collect its
// stdout, stderr, and exit code.
func (a *ShellAdapter) execAction(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	command, err := getStr(params, "command")
	if err != nil {
		return nil, tryve.AdapterError("shell", "exec", err.Error(), err)
	}

	cwd := getStrDefault(params, "cwd", a.config.DefaultCwd)
	extraEnv := getMap(params, "env")

	cmd := buildCommand(ctx, command)

	if cwd != "" {
		cmd.Dir = cwd
	}

	cmd.Env = buildEnv(extraEnv)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var duration, runErr = MeasureDuration(func() error {
		return cmd.Run()
	})

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			// Non-zero exit: capture the code and do NOT treat it as an error.
			exitCode = exitErr.ExitCode()
			runErr = nil
		} else {
			// Actual execution failure (e.g. command not found).
			return nil, tryve.AdapterError("shell", "exec", runErr.Error(), runErr)
		}
	}

	data := map[string]any{
		"stdout":   stdout.String(),
		"stderr":   stderr.String(),
		"exitCode": float64(exitCode),
	}

	_ = runErr // already handled above
	return SuccessResult(data, duration, nil), nil
}

// buildCommand constructs the exec.Cmd appropriate for the current OS.
// On Windows it uses "cmd /C"; on all other platforms it uses "sh -c".
func buildCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

// buildEnv merges the current process environment with any extra variables
// provided in envMap. Values in envMap take precedence.
func buildEnv(envMap map[string]any) []string {
	base := os.Environ()
	if len(envMap) == 0 {
		return base
	}

	// Copy inherited env and append extras.
	env := make([]string, len(base), len(base)+len(envMap))
	copy(env, base)
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}
	return env
}

// getStr retrieves a required string parameter from params.
// Returns an error if the key is absent or its value is not a string.
func getStr(params map[string]any, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", fmt.Errorf("required parameter %q is missing", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %q must be a string, got %T", key, v)
	}
	return s, nil
}

// getStrDefault retrieves an optional string parameter from params, returning
// defaultVal when the key is absent or its value is not a string.
func getStrDefault(params map[string]any, key, defaultVal string) string {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

// getMap retrieves an optional map[string]any parameter from params.
// Returns nil when the key is absent or the value is not a map[string]any.
func getMap(params map[string]any, key string) map[string]any {
	v, ok := params[key]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}
