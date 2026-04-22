package doctor

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/liemle3893/autoflow/internal/autoflow/config"
)

// withStubbedRunners swaps execLookPath + runCombined for the duration of fn.
func withStubbedRunners(t *testing.T, lookPath func(string) (string, error),
	combined func(ctx context.Context, name string, args ...string) (string, error), fn func()) {
	t.Helper()
	origLP, origRC := execLookPath, runCombined
	if lookPath != nil {
		execLookPath = lookPath
	}
	if combined != nil {
		runCombined = combined
	}
	defer func() {
		execLookPath = origLP
		runCombined = origRC
	}()
	fn()
}

func TestCheckCodingAgent_OK(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "coding_agent", "claude")
	r := checkCodingAgent(root)(context.Background())
	if r.Status != OK {
		t.Errorf("want OK, got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckCodingAgent_Unset(t *testing.T) {
	r := checkCodingAgent(t.TempDir())(context.Background())
	if r.Status != Fail {
		t.Errorf("want Fail, got %s", r.Status)
	}
}

func TestCheckSbxInstalled_SkipWhenDisabled(t *testing.T) {
	r := checkSbxInstalled(t.TempDir())(context.Background())
	if r.Status != Skip {
		t.Errorf("want Skip, got %s", r.Status)
	}
}

func TestCheckSbxInstalled_FailWhenMissing(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	withStubbedRunners(t,
		func(string) (string, error) { return "", errors.New("nope") },
		nil, func() {
			r := checkSbxInstalled(root)(context.Background())
			if r.Status != Fail {
				t.Errorf("want Fail, got %s", r.Status)
			}
		})
}

func TestCheckSbxInstalled_OK(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	withStubbedRunners(t,
		func(string) (string, error) { return "/usr/local/bin/sbx", nil },
		nil, func() {
			r := checkSbxInstalled(root)(context.Background())
			if r.Status != OK {
				t.Errorf("want OK, got %s (%s)", r.Status, r.Detail)
			}
		})
}

func TestCheckSbxVersion_OK(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	withStubbedRunners(t,
		func(string) (string, error) { return "sbx", nil },
		func(ctx context.Context, name string, args ...string) (string, error) {
			return "sbx 0.5.1\n", nil
		}, func() {
			r := checkSbxVersion(root)(context.Background())
			if r.Status != OK || r.Detail != "sbx 0.5.1" {
				t.Errorf("unexpected: %+v", r)
			}
		})
}

func TestCheckSbxDocker_FailWhenInfoFails(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	withStubbedRunners(t,
		func(string) (string, error) { return "docker", nil },
		func(ctx context.Context, name string, args ...string) (string, error) {
			return "", &exec.ExitError{}
		}, func() {
			r := checkSbxDocker(root)(context.Background())
			if r.Status != Fail {
				t.Errorf("want Fail, got %s", r.Status)
			}
		})
}

func TestCheckSbxAuth_OK(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	withStubbedRunners(t,
		func(string) (string, error) { return "sbx", nil },
		func(ctx context.Context, name string, args ...string) (string, error) {
			return "NAME\n", nil
		}, func() {
			r := checkSbxAuth(root)(context.Background())
			if r.Status != OK {
				t.Errorf("want OK, got %s (%s)", r.Status, r.Detail)
			}
		})
}

func TestCheckSbxAgentCreds_ClaudeWarnsWhenMissing(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	_ = config.Set(root, "coding_agent", "claude")
	t.Setenv("ANTHROPIC_API_KEY", "")
	withStubbedRunners(t, nil,
		func(ctx context.Context, name string, args ...string) (string, error) {
			return "", nil // no secrets
		}, func() {
			r := checkSbxAgentCreds(root)(context.Background())
			if r.Status != Warn {
				t.Errorf("want Warn, got %s (%s)", r.Status, r.Detail)
			}
		})
}

func TestCheckSbxAgentCreds_ClaudeOKWithEnv(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	_ = config.Set(root, "coding_agent", "claude")
	t.Setenv("ANTHROPIC_API_KEY", "sk-xxx")
	r := checkSbxAgentCreds(root)(context.Background())
	if r.Status != OK {
		t.Errorf("want OK, got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckSbxAgentCreds_CopilotFailWhenMissing(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	_ = config.Set(root, "coding_agent", "copilot")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	withStubbedRunners(t, nil,
		func(ctx context.Context, name string, args ...string) (string, error) {
			return "", nil
		}, func() {
			r := checkSbxAgentCreds(root)(context.Background())
			if r.Status != Fail {
				t.Errorf("want Fail, got %s (%s)", r.Status, r.Detail)
			}
		})
}

func TestCheckSbxAutoflow_SkipWhenSandboxMissing(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	_ = config.Set(root, "sandbox.name", "pinned-name")
	withStubbedRunners(t,
		func(string) (string, error) { return "sbx", nil },
		func(ctx context.Context, name string, args ...string) (string, error) {
			// sbx ls returns nothing matching.
			return "other-sandbox\n", nil
		}, func() {
			r := checkSbxAutoflow(root)(context.Background())
			if r.Status != Skip {
				t.Errorf("want Skip, got %s (%s)", r.Status, r.Detail)
			}
		})
}

func TestCheckSbxAutoflow_FailWhenAutoflowMissing(t *testing.T) {
	root := t.TempDir()
	_ = config.Set(root, "sandbox.enabled", "true")
	_ = config.Set(root, "sandbox.name", "pinned")
	calls := 0
	withStubbedRunners(t,
		func(string) (string, error) { return "sbx", nil },
		func(ctx context.Context, name string, args ...string) (string, error) {
			calls++
			if calls == 1 {
				// sbx ls
				return "pinned\n", nil
			}
			// command -v autoflow fails
			return "", errors.New("not found")
		}, func() {
			r := checkSbxAutoflow(root)(context.Background())
			if r.Status != Fail {
				t.Errorf("want Fail, got %s (%s)", r.Status, r.Detail)
			}
		})
}

func TestWorstOf_SkipDoesNotPromote(t *testing.T) {
	if worstOf(OK, Skip) != OK {
		t.Errorf("Skip must not promote OK")
	}
	if worstOf(Skip, OK) != OK {
		t.Errorf("OK must win over Skip")
	}
	if worstOf(Skip, Fail) != Fail {
		t.Errorf("Fail must win over Skip")
	}
}
