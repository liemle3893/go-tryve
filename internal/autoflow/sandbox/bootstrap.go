package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// BootstrapOpts configures Bootstrap. HostVer is the value of the running
// binary's version string (e.g. "v1.2.3" for a release or "dev" for a local
// build). HostBinary is only used in dev mode.
type BootstrapOpts struct {
	Name       string
	Arch       string // amd64|arm64; autodetected when empty
	HostVer    string
	HostBinary string // path to the host autoflow binary for dev-mode install

	// Runner lets tests inject a fake command runner. Nil means use ExecRunner.
	Runner Runner
}

// releaseVersionRE recognises v-prefixed semver: v1.2.3 (no pre-release).
var releaseVersionRE = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// Bootstrap installs the host's autoflow binary into <sandbox>:/usr/local/bin/autoflow
// and verifies the version matches.
func Bootstrap(ctx context.Context, out io.Writer, opts BootstrapOpts) error {
	if opts.Name == "" {
		return fmt.Errorf("bootstrap: sandbox name required")
	}
	r := opts.Runner
	if r == nil {
		r = ExecRunner{}
	}

	arch := opts.Arch
	if arch == "" {
		detected, err := detectArch(ctx, r, opts.Name)
		if err != nil {
			return fmt.Errorf("bootstrap: detect arch: %w", err)
		}
		arch = detected
	}
	fmt.Fprintf(out, "Sandbox %s: target arch %s\n", opts.Name, arch)

	if releaseVersionRE.MatchString(opts.HostVer) {
		fmt.Fprintf(out, "Installing autoflow %s from GitHub release...\n", opts.HostVer)
		if err := installFromRelease(ctx, r, opts.Name, opts.HostVer, arch, out); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(out, "Dev build detected (version=%q); cross-compiling for linux/%s...\n", opts.HostVer, arch)
		if err := installFromDevBuild(ctx, r, opts.Name, arch, out); err != nil {
			return err
		}
	}

	// Verify.
	sbVer, err := captureRun(ctx, r, "sbx", "exec", opts.Name, "--", "autoflow", "version")
	if err != nil {
		return fmt.Errorf("bootstrap: verify autoflow version: %w", err)
	}
	sbVer = strings.TrimSpace(sbVer)
	fmt.Fprintf(out, "Installed autoflow version in sandbox: %s\n", sbVer)
	if opts.HostVer != "" && sbVer != "" && !versionsMatch(opts.HostVer, sbVer) {
		fmt.Fprintf(out, "WARN: host version %q != sandbox version %q\n", opts.HostVer, sbVer)
	}
	return nil
}

// detectArch maps `uname -m` from inside the sandbox to Go's GOARCH naming.
func detectArch(ctx context.Context, r Runner, name string) (string, error) {
	out, err := captureRun(ctx, r, "sbx", "exec", name, "--", "uname", "-m")
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(out) {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported arch %q", strings.TrimSpace(out))
	}
}

// installFromRelease pulls the tar.gz from GitHub into the sandbox via curl.
func installFromRelease(ctx context.Context, r Runner, name, ver, arch string, out io.Writer) error {
	url := fmt.Sprintf(
		"https://github.com/liemle3893/autoflow/releases/download/%s/autoflow_%s_linux_%s.tar.gz",
		ver, ver, arch,
	)
	// Shell pipeline run inside the sandbox via `sh -c`.
	script := fmt.Sprintf(
		"curl -fsSL %s | tar xz -C /usr/local/bin autoflow && chmod +x /usr/local/bin/autoflow",
		url,
	)
	return r.Run(ctx, "sbx", []string{"exec", name, "--", "sh", "-c", script}, nil, out, out)
}

// installFromDevBuild cross-compiles the host source then pipes the binary
// into the sandbox via `tee`. Assumes cwd is the autoflow repo root.
func installFromDevBuild(ctx context.Context, r Runner, name, arch string, out io.Writer) error {
	tmp := "/tmp/autoflow-sandbox"
	buildArgs := []string{"build", "-o", tmp, "./cmd/autoflow"}
	env := append(os.Environ(), "GOOS=linux", "GOARCH="+arch)
	// Custom exec that honours env; sidestep Runner for this one call.
	if err := runWithEnv(ctx, out, env, "go", buildArgs...); err != nil {
		return fmt.Errorf("cross-compile: %w", err)
	}
	defer os.Remove(tmp)

	data, err := os.ReadFile(tmp)
	if err != nil {
		return fmt.Errorf("read built binary: %w", err)
	}
	// Stream bytes into the sandbox via tee.
	if err := r.Run(ctx, "sbx",
		[]string{"exec", name, "--", "sh", "-c", "tee /usr/local/bin/autoflow > /dev/null"},
		bytes.NewReader(data), out, out); err != nil {
		return fmt.Errorf("copy binary into sandbox: %w", err)
	}
	if err := r.Run(ctx, "sbx",
		[]string{"exec", name, "--", "chmod", "+x", "/usr/local/bin/autoflow"},
		nil, out, out); err != nil {
		return fmt.Errorf("chmod +x in sandbox: %w", err)
	}
	return nil
}

// runWithEnv is an exec shim that lets us pass a custom env (which Runner
// does not currently support). Used only for the go-build step.
var runWithEnv = defaultRunWithEnv

func defaultRunWithEnv(ctx context.Context, out io.Writer, env []string, name string, args ...string) error {
	cmd := execCommand(ctx, name, args...)
	cmd.Env = env
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// versionsMatch accepts either exact equality or "version vX.Y.Z" == "vX.Y.Z".
// sbx may print "autoflow vX.Y.Z" depending on the version handler.
func versionsMatch(host, sb string) bool {
	host = strings.TrimSpace(host)
	sb = strings.TrimSpace(sb)
	if host == sb {
		return true
	}
	return strings.Contains(sb, host)
}
