package worktree

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

// safePrefixes lists binaries allowed to run via RunSafeCmd without user
// confirmation. Matches SAFE_CMD_PREFIXES in worktree-bootstrap.sh.
var safePrefixes = map[string]bool{
	"npm": true, "yarn": true, "pnpm": true, "bun": true, "npx": true,
	"pip": true, "pip3": true, "pipenv": true, "poetry": true, "uv": true,
	"go": true, "cargo": true, "rustup": true,
	"make": true, "cmake": true,
	"mvn": true, "gradle": true, "ant": true,
	"dotnet": true, "nuget": true,
	"composer": true,
	"bundle":   true, "gem": true, "rake": true,
	"mix":   true,
	"swift": true,
}

// Prompter is the interactive front end RunSafeCmd calls on an
// unrecognised binary. InteractivePrompter ticks a stdin-y/N prompt;
// NonInteractivePrompter always returns false, matching the bash script's
// "non-interactive → skip" behaviour. Tests supply their own.
type Prompter interface {
	// AllowUnknownCommand returns true if the user approves running cmd
	// despite binary not being on the allowlist. When not ok, the
	// command is skipped (RunSafeCmd returns nil, ErrSkipped).
	AllowUnknownCommand(label, cmd, binary string) (ok bool, err error)
}

// ErrSkipped is returned by RunSafeCmd when the command was skipped
// because its binary is not on the allowlist and the prompter said no.
var ErrSkipped = errors.New("command skipped")

// NonInteractivePrompter denies every prompt. Use in CI or any path
// without a TTY.
type NonInteractivePrompter struct{}

func (NonInteractivePrompter) AllowUnknownCommand(_, _, _ string) (bool, error) {
	return false, nil
}

// InteractivePrompter reads y/N from in. stdin in most cases.
type InteractivePrompter struct {
	In  io.Reader
	Out io.Writer
}

func (p InteractivePrompter) AllowUnknownCommand(label, cmd, binary string) (bool, error) {
	fmt.Fprintf(p.Out, "WARNING: %s command uses an unrecognised binary: %s\n", label, binary)
	fmt.Fprintf(p.Out, "  Full command: %s\n", cmd)
	fmt.Fprintf(p.Out, "  Allow execution? [y/N] ")
	r := bufio.NewReader(p.In)
	line, _ := r.ReadString('\n')
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes", nil
}

// RunSafeCmd executes cmd in workdir after verifying its first token is
// on the allowlist. Unknown binaries trigger the prompter. Returns
// ErrSkipped when the prompter declines.
func RunSafeCmd(label, cmd, workdir string, prompter Prompter, stdout, stderr io.Writer) error {
	binary := firstTokenBinary(cmd)
	if binary == "" {
		return fmt.Errorf("%s command is empty", label)
	}

	if !safePrefixes[binary] {
		if prompter == nil {
			prompter = NonInteractivePrompter{}
		}
		ok, err := prompter.AllowUnknownCommand(label, cmd, binary)
		if err != nil {
			return err
		}
		if !ok {
			return ErrSkipped
		}
	}

	// Execute via /bin/sh so the bootstrap.json command strings can use
	// shell features the user relies on (redirects, pipes, env=value
	// prefixes). This matches `bash -c "$cmd"` from the original script.
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}

// firstTokenBinary extracts the binary name from the first whitespace-
// separated token, stripping any leading path.
func firstTokenBinary(cmd string) string {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return ""
	}
	first := strings.Fields(trimmed)[0]
	return filepath.Base(first)
}
