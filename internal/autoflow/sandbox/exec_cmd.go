package sandbox

import (
	"context"
	"os/exec"
)

// defaultExecCommand wraps exec.CommandContext. Extracted so tests can swap it.
func defaultExecCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
