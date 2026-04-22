package deliver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

func TestNext_FreshTicketReturnsStep1Dispatch(t *testing.T) {
	c := NewController(t.TempDir())
	instr, err := c.Next("PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	if instr.Action != ActionDispatch {
		t.Errorf("want dispatch, got %s", instr.Action)
	}
	if instr.SubagentType != "autoflow-jira-fetcher" {
		t.Errorf("want jira-fetcher, got %q", instr.SubagentType)
	}
	if instr.Step != 1 {
		t.Errorf("want step 1, got %d", instr.Step)
	}
	// JSON round-trip sanity check.
	out, _ := MarshalIndent(instr)
	if !strings.Contains(string(out), `"action": "dispatch"`) {
		t.Errorf("missing action=dispatch in JSON:\n%s", out)
	}
}

func TestNext_Step1AutoCompletesWhenBriefExists(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)

	// Seed task-brief.md with frontmatter title.
	briefDir := state.TicketDir(root, "PROJ-1")
	_ = os.MkdirAll(briefDir, 0o755)
	_ = os.WriteFile(filepath.Join(briefDir, "task-brief.md"),
		[]byte("---\ntitle: Hello\n---\n# Hello\n\ncontent\n"), 0o644)

	instr, err := c.Next("PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	// Pre-init path: brief exists, progress file does not → step_02 runs.
	if instr.Step != 2 {
		t.Errorf("expected step 2 after brief, got %d", instr.Step)
	}
	// step_02 now executes inline. Against a non-git tempdir it will
	// escalate on `git fetch`, which confirms the routing without
	// needing a fully-wired test repo — the integration test
	// TestStep02_Integration covers the success path.
	if instr.Action != ActionEscalate {
		t.Errorf("expected escalate from git fetch in non-git tempdir, got %s", instr.Action)
	}
	if !strings.Contains(instr.Reason, "git fetch") {
		t.Errorf("escalate reason should mention git fetch, got %q", instr.Reason)
	}
}

func TestNext_DoneWhenAllCompleted(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)
	// Force all 13 completed.
	p, _ := state.ReadProgress(root, "PROJ-1")
	p.Completed = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	p.CurrentStep = state.MaxStep
	_ = state.WriteJSONAtomic(state.ProgressFile(root, "PROJ-1"), p)

	instr, err := c.Next("PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	if instr.Action != ActionDone {
		t.Errorf("want done, got %s", instr.Action)
	}
}

// seedBrief drops a minimal task-brief.md under the ticket dir so step-1
// preconditions pass in tests that are only exercising later steps.
func seedBrief(t *testing.T, root, key string) {
	t.Helper()
	dir := state.TicketDir(root, key)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task-brief.md"), []byte("# brief"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestComplete_PreInitWritesSidecar(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	seedBrief(t, root, "PROJ-1")

	resp, err := c.Complete("PROJ-1", CompleteOpts{Title: "My Title"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.CompletedStep != 1 || resp.NextStep != 2 {
		t.Errorf("unexpected response: %+v", resp)
	}
	data, err := os.ReadFile(filepath.Join(state.TicketDir(root, "PROJ-1"), "title.txt"))
	if err != nil {
		t.Fatalf("sidecar missing: %v", err)
	}
	if string(data) != "My Title" {
		t.Errorf("sidecar wrong: %q", data)
	}
}

func TestComplete_PreInitRejectsWithoutBrief(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	// No seedBrief — Complete should refuse.
	_, err := c.Complete("PROJ-1", CompleteOpts{Title: "X"})
	if err == nil {
		t.Fatal("expected precondition error, got nil")
	}
	if !strings.Contains(err.Error(), "task-brief.md") {
		t.Errorf("error should mention task-brief.md, got %v", err)
	}
}

func TestComplete_AfterInitAdvances(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	seedBrief(t, root, "PROJ-1")
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)
	resp, err := c.Complete("PROJ-1", CompleteOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.CompletedStep != 1 || resp.NextStep != 2 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestInit_SeedsProgress(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	if err := c.Init("PROJ-1", "/wt", "branch"); err != nil {
		t.Fatal(err)
	}
	p, _ := state.ReadProgress(root, "PROJ-1")
	if p == nil || p.Ticket != "PROJ-1" {
		t.Errorf("init didn't seed progress: %+v", p)
	}
}

func TestInit_ValidatesKey(t *testing.T) {
	c := NewController(t.TempDir())
	if err := c.Init("../bad", "/wt", "b"); err == nil {
		t.Errorf("invalid key should error")
	}
}

func TestStep02_SkipsWhenWorktreeExists(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	existing := t.TempDir()
	_, _ = state.InitProgress(root, "PROJ-1", existing, "b", false)
	p, _ := state.ReadProgress(root, "PROJ-1")
	instr := c.step02("PROJ-1", p)
	if instr.Action != ActionAutoComplete {
		t.Errorf("existing worktree should auto-complete, got %s", instr.Action)
	}
}

func TestStep04_InitLoopWhenNoState(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)
	p, _ := state.ReadProgress(root, "PROJ-1")
	instr := c.step04("PROJ-1", p)
	if instr.Action != ActionBash {
		t.Errorf("step_04 should init via bash, got %s", instr.Action)
	}
	if !strings.Contains(strings.Join(instr.Commands, " "), "loop-state init") {
		t.Errorf("step_04 bash should call loop-state init, got: %v", instr.Commands)
	}
}

func TestStep04_DispatchesReviewerWhenNoPass(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)
	// Seed the state file with one GAPS_FOUND round.
	stateFile := filepath.Join(state.TicketStateDir(root, "PROJ-1"), "coverage-review-state.json")
	_ = os.MkdirAll(filepath.Dir(stateFile), 0o755)
	body := `{"loop":"coverage-review","ticket":"PROJ-1","max_rounds":3,"rounds":[{"round":1,"status":"GAPS_FOUND"}]}`
	_ = os.WriteFile(stateFile, []byte(body), 0o644)

	p, _ := state.ReadProgress(root, "PROJ-1")
	instr := c.step04("PROJ-1", p)
	if instr.Action != ActionDispatch {
		t.Errorf("want dispatch for round 2, got %s", instr.Action)
	}
	if instr.SubagentType != "autoflow-ac-reviewer" {
		t.Errorf("wrong subagent: %q", instr.SubagentType)
	}
}

func TestStep06_SkipsWithoutCmds(t *testing.T) {
	root := t.TempDir()
	c := NewController(root)
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)
	p, _ := state.ReadProgress(root, "PROJ-1")
	instr := c.step06("PROJ-1", p)
	if instr.Action != ActionAutoComplete {
		t.Errorf("step_06 with no cmds should auto-complete, got %s", instr.Action)
	}
}

func TestParseBrief_Frontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task-brief.md")
	_ = os.WriteFile(path, []byte(`---
title: Nice
path_recommendation: A
has_fix_strategy: true
estimated_files: 3
---

# Content
`), 0o644)
	meta, err := ParseBrief(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta["title"] != "Nice" || meta["path_recommendation"] != "A" {
		t.Errorf("meta: %+v", meta)
	}
}

func TestParseBrief_BareShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task-brief.md")
	// Bare shape (matching the Python parser): `key: value` lines come
	// IMMEDIATELY after the title heading with no intervening blank.
	_ = os.WriteFile(path, []byte(`# Ticket Title
title: Bare
path_recommendation: B

## Other section
`), 0o644)
	meta, err := ParseBrief(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta["title"] != "Bare" || meta["path_recommendation"] != "B" {
		t.Errorf("meta: %+v", meta)
	}
}

func TestGateResult_WritesStateAndTail(t *testing.T) {
	root := t.TempDir()
	_, _ = state.InitProgress(root, "PROJ-1", "/wt", "b", false)

	logPath := filepath.Join(t.TempDir(), "log")
	var lines strings.Builder
	for i := 0; i < 150; i++ {
		lines.WriteString("line ")
		lines.WriteString(itoa(i))
		lines.WriteByte('\n')
	}
	_ = os.WriteFile(logPath, []byte(lines.String()), 0o644)

	if err := GateResult(root, "PROJ-1", 1, 1, logPath); err != nil {
		t.Fatal(err)
	}

	// State JSON should say fail + point at the tail log.
	data, _ := os.ReadFile(filepath.Join(state.TicketStateDir(root, "PROJ-1"), "build-gate-state.json"))
	var gs buildGateState
	_ = json.Unmarshal(data, &gs)
	if gs.LastResult != "fail" {
		t.Errorf("want fail, got %q", gs.LastResult)
	}
	if gs.ErrorFile == "" {
		t.Errorf("error_file not set")
	}
	// Tail file should exist, 100 lines.
	tail, err := os.ReadFile(gs.ErrorFile)
	if err != nil {
		t.Fatal(err)
	}
	nonEmpty := 0
	for _, l := range strings.Split(string(tail), "\n") {
		if l != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 100 {
		t.Errorf("want 100 tail lines, got %d", nonEmpty)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"simple":      "'simple'",
		"":            "''",
		"it's":        `'it'\''s'`,
		"a b":         "'a b'",
		"$INJECT":     `'$INJECT'`,
		"`evil`":      "'`evil`'",
		"inline'quot": `'inline'\''quot'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCountReviewFindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "REVIEW.md")
	body := `---
status: ISSUES_FOUND
findings:
  critical: 2
  warning: 3
  info: 4
---

# Review

### CR-01: Foo
`
	_ = os.WriteFile(path, []byte(body), 0o644)
	c, w := countReviewFindings(path)
	if c != 2 || w != 3 {
		t.Errorf("want (2,3) got (%d,%d)", c, w)
	}
}
