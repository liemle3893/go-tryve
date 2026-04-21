package deliver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePlan(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "PLAN.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParsePlan_Happy(t *testing.T) {
	body := `# X

<task>
  <id>task-01</id>
  <name>Add types</name>
  <files>a.go, b.go</files>
  <deps></deps>
  <action>add types</action>
  <verify>go build</verify>
  <done>compiles</done>
</task>

<task>
  <id>task-02</id>
  <name>Wire handler</name>
  <files>c.go</files>
  <deps>task-01</deps>
  <action>wire it</action>
  <verify>go test</verify>
  <done>passes</done>
</task>
`
	plan, err := ParsePlan(writePlan(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 {
		t.Fatalf("want 2, got %d", len(plan))
	}
	if plan[0].ID != "task-01" || plan[1].ID != "task-02" {
		t.Errorf("ids: %s / %s", plan[0].ID, plan[1].ID)
	}
	if len(plan[0].Files) != 2 || plan[0].Files[0] != "a.go" {
		t.Errorf("files: %+v", plan[0].Files)
	}
	if len(plan[1].Deps) != 1 || plan[1].Deps[0] != "task-01" {
		t.Errorf("deps: %+v", plan[1].Deps)
	}
}

func TestParsePlan_SynthesisesMissingID(t *testing.T) {
	body := `
<task>
  <name>Only</name>
  <files>x.go</files>
  <action>a</action>
  <verify>v</verify>
  <done>d</done>
</task>
`
	plan, err := ParsePlan(writePlan(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if plan[0].ID != "task-01" {
		t.Errorf("synthesised id: %s", plan[0].ID)
	}
}

func TestParsePlan_RejectsCycle(t *testing.T) {
	body := `
<task><id>a</id><deps>b</deps><name>A</name><files>x</files><action>x</action><verify>x</verify><done>x</done></task>
<task><id>b</id><deps>a</deps><name>B</name><files>x</files><action>x</action><verify>x</verify><done>x</done></task>
`
	_, err := ParsePlan(writePlan(t, body))
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle error, got %v", err)
	}
}

func TestParsePlan_RejectsUnknownDep(t *testing.T) {
	body := `
<task><id>a</id><deps>nope</deps><name>A</name><files>x</files><action>x</action><verify>x</verify><done>x</done></task>
`
	_, err := ParsePlan(writePlan(t, body))
	if err == nil || !strings.Contains(err.Error(), "unknown task") {
		t.Errorf("expected unknown-dep error, got %v", err)
	}
}

func TestParsePlan_RejectsDuplicateID(t *testing.T) {
	body := `
<task><id>a</id><name>1</name><files>x</files><action>x</action><verify>x</verify><done>x</done></task>
<task><id>a</id><name>2</name><files>x</files><action>x</action><verify>x</verify><done>x</done></task>
`
	_, err := ParsePlan(writePlan(t, body))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate-id error, got %v", err)
	}
}
