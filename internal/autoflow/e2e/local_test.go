package e2e

import (
	"testing"

	"github.com/liemle3893/go-tryve/pkg/runner"
)

func TestApplyTestSelection_Tag(t *testing.T) {
	var opts runner.Options
	applyTestSelection("--tag PROJ-42", &opts)
	if len(opts.Tags) != 1 || opts.Tags[0] != "PROJ-42" {
		t.Errorf("tag parse: got %+v", opts.Tags)
	}
}

func TestApplyTestSelection_Grep(t *testing.T) {
	var opts runner.Options
	applyTestSelection("--grep foo", &opts)
	if opts.Grep != "foo" {
		t.Errorf("grep parse: got %q", opts.Grep)
	}
}

func TestApplyTestSelection_Glob(t *testing.T) {
	var opts runner.Options
	applyTestSelection("tests/e2e/**/TC-PROJ-1-*.test.yaml", &opts)
	if opts.Grep == "" {
		t.Errorf("glob should fall through to Grep")
	}
}

func TestApplyTestSelection_Empty(t *testing.T) {
	var opts runner.Options
	applyTestSelection("   ", &opts)
	if len(opts.Tags) != 0 || opts.Grep != "" {
		t.Errorf("empty selection should leave opts untouched")
	}
}

func TestSafeBranch(t *testing.T) {
	if got := safeBranch("jira-iss/proj-42"); got != "jira-iss-proj-42" {
		t.Errorf("safeBranch: got %q", got)
	}
}

func TestApplyTestSelection_TagAndGrep(t *testing.T) {
	var opts runner.Options
	applyTestSelection("--tag PROJ-1 --grep smoke", &opts)
	if len(opts.Tags) != 1 || opts.Tags[0] != "PROJ-1" {
		t.Errorf("expected one tag, got %+v", opts.Tags)
	}
	if opts.Grep != "smoke" {
		t.Errorf("expected grep=smoke, got %q", opts.Grep)
	}
}
