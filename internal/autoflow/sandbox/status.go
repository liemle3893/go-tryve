package sandbox

import (
	"context"
	"strings"
)

// execCommand is an indirection to os/exec for the one codepath that needs
// a custom Env. Overridable for tests.
var execCommand = defaultExecCommand

// Status returns the host binary's version, the sandbox binary's version, and
// the sandbox arch. Any missing piece is returned as an empty string rather
// than an error so callers can show partial info.
func Status(ctx context.Context, name string) (hostVer, sandboxVer, arch string, err error) {
	r := ExecRunner{}
	// Host version: `autoflow version` from PATH (not the sandbox).
	if v, e := captureRun(ctx, r, "autoflow", "version"); e == nil {
		hostVer = strings.TrimSpace(v)
	}
	if v, e := captureRun(ctx, r, "sbx", "exec", name, "--", "autoflow", "version"); e == nil {
		sandboxVer = strings.TrimSpace(v)
	}
	if a, e := detectArch(ctx, r, name); e == nil {
		arch = a
	}
	return hostVer, sandboxVer, arch, nil
}

// SandboxExists returns true when `sbx ls` output mentions a sandbox with the
// given name. Matching is done on whole-word lines to avoid partial hits.
func SandboxExists(ctx context.Context, name string) (bool, error) {
	out, err := captureRun(ctx, ExecRunner{}, "sbx", "ls")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		for _, tok := range strings.Fields(line) {
			if tok == name {
				return true, nil
			}
		}
	}
	return false, nil
}
