package cli

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"
)

// TestRootTopLevelCommands pins the top-level command list so accidental
// renames, removals, or additions surface immediately in review.
func TestRootTopLevelCommands(t *testing.T) {
	want := []string{
		"config", "deliver", "doctor", "e2e",
		"install", "jira", "loop-state", "sandbox", "scaffold-e2e", "version", "worktree",
	}
	sort.Strings(want)

	root := NewRoot("test")
	got := commandNames(root.Commands())
	assertEqualNames(t, "root", got, want)
}

// TestE2ESubtreeCommands pins the e2e subtree.
func TestE2ESubtreeCommands(t *testing.T) {
	want := []string{"doc", "health", "init", "list", "run", "test", "validate"}
	sort.Strings(want)

	root := NewRoot("test")
	e2e := findChild(root, "e2e")
	if e2e == nil {
		t.Fatal("e2e subcommand not found under root")
	}
	got := commandNames(e2e.Commands())
	assertEqualNames(t, "e2e", got, want)
}

// TestRootUseIsAutoflow guards against a regression to the old binary name.
func TestRootUseIsAutoflow(t *testing.T) {
	root := NewRoot("test")
	if root.Use != "autoflow" {
		t.Errorf("root.Use = %q, want %q", root.Use, "autoflow")
	}
}

func commandNames(cmds []*cobra.Command) []string {
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, c.Name())
	}
	sort.Strings(out)
	return out
}

func findChild(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func assertEqualNames(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s command count: got %d %v, want %d %v",
			label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s commands[%d]: got %q, want %q", label, i, got[i], want[i])
		}
	}
}
