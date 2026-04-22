package sandbox

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

// fakeRunner records calls and returns scripted stdout/err/error per step.
type fakeRunner struct {
	calls    []call
	responses []response
}

type call struct {
	name string
	args []string
	stdin []byte
}

type response struct {
	stdout string
	stderr string
	err    error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	c := call{name: name, args: append([]string{}, args...)}
	if stdin != nil {
		b, _ := io.ReadAll(stdin)
		c.stdin = b
	}
	f.calls = append(f.calls, c)
	if len(f.responses) == 0 {
		return nil
	}
	r := f.responses[0]
	f.responses = f.responses[1:]
	if stdout != nil && r.stdout != "" {
		_, _ = stdout.Write([]byte(r.stdout))
	}
	if stderr != nil && r.stderr != "" {
		_, _ = stderr.Write([]byte(r.stderr))
	}
	return r.err
}

func TestBootstrap_ReleaseMode(t *testing.T) {
	f := &fakeRunner{
		responses: []response{
			{stdout: "x86_64\n"},         // uname -m
			{},                            // curl | tar
			{stdout: "v1.2.3\n"},         // verify
		},
	}
	var out bytes.Buffer
	err := Bootstrap(context.Background(), &out, BootstrapOpts{
		Name:    "sb1",
		HostVer: "v1.2.3",
		Runner:  f,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if len(f.calls) != 3 {
		t.Fatalf("want 3 calls, got %d: %+v", len(f.calls), f.calls)
	}
	// Second call should be the curl|tar pipeline.
	joined := strings.Join(f.calls[1].args, " ")
	if !strings.Contains(joined, "curl") || !strings.Contains(joined, "linux_amd64") {
		t.Errorf("release install args wrong: %v", f.calls[1].args)
	}
}

func TestBootstrap_ExplicitArchSkipsDetect(t *testing.T) {
	f := &fakeRunner{
		responses: []response{
			{},                     // curl|tar
			{stdout: "v1.0.0\n"},   // verify
		},
	}
	var out bytes.Buffer
	err := Bootstrap(context.Background(), &out, BootstrapOpts{
		Name: "sb1", Arch: "arm64", HostVer: "v1.0.0", Runner: f,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 {
		t.Fatalf("want 2 calls (no uname), got %d", len(f.calls))
	}
	joined := strings.Join(f.calls[0].args, " ")
	if !strings.Contains(joined, "linux_arm64") {
		t.Errorf("expected linux_arm64 in args, got %v", f.calls[0].args)
	}
}

func TestBootstrap_NameRequired(t *testing.T) {
	var out bytes.Buffer
	if err := Bootstrap(context.Background(), &out, BootstrapOpts{}); err == nil {
		t.Error("want error when Name empty")
	}
}

func TestBootstrap_UnsupportedArch(t *testing.T) {
	f := &fakeRunner{responses: []response{{stdout: "riscv64\n"}}}
	var out bytes.Buffer
	if err := Bootstrap(context.Background(), &out, BootstrapOpts{
		Name: "sb", HostVer: "v1.0.0", Runner: f,
	}); err == nil {
		t.Error("want error on unknown arch")
	}
}

func TestBootstrap_DevMode(t *testing.T) {
	// Swap runWithEnv so we don't actually run `go build`.
	orig := runWithEnv
	runWithEnv = func(ctx context.Context, out io.Writer, env []string, name string, args ...string) error {
		// Write a stub binary to the output path.
		for i, a := range args {
			if a == "-o" && i+1 < len(args) {
				if err := writeStub(args[i+1]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	defer func() { runWithEnv = orig }()

	f := &fakeRunner{
		responses: []response{
			{stdout: "aarch64\n"},  // uname -m
			{},                      // tee
			{},                      // chmod
			{stdout: "dev\n"},       // verify
		},
	}
	var out bytes.Buffer
	err := Bootstrap(context.Background(), &out, BootstrapOpts{
		Name:    "sb1",
		HostVer: "dev",
		Runner:  f,
	})
	if err != nil {
		t.Fatalf("Bootstrap dev: %v", err)
	}
	// Verify tee call received stdin bytes.
	var teeCall *call
	for i, c := range f.calls {
		if len(c.args) > 0 && strings.Contains(strings.Join(c.args, " "), "tee") {
			teeCall = &f.calls[i]
			break
		}
	}
	if teeCall == nil {
		t.Fatalf("tee call not found; calls=%+v", f.calls)
	}
	if len(teeCall.stdin) == 0 {
		t.Error("tee call stdin was empty; binary bytes not piped")
	}
}

func TestVersionsMatch(t *testing.T) {
	cases := []struct {
		host, sb string
		want     bool
	}{
		{"v1.0.0", "v1.0.0", true},
		{"v1.0.0", "autoflow v1.0.0", true},
		{"v1.0.0", "v1.0.1", false},
		{"dev", "dev", true},
	}
	for _, tc := range cases {
		if got := versionsMatch(tc.host, tc.sb); got != tc.want {
			t.Errorf("versionsMatch(%q,%q)=%v want %v", tc.host, tc.sb, got, tc.want)
		}
	}
}

func writeStub(path string) error {
	return os.WriteFile(path, []byte("stub-binary"), 0o755)
}
