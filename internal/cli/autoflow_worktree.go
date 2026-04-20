package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/liemle3893/go-tryve/internal/autoflow/state"
	"github.com/liemle3893/go-tryve/internal/autoflow/worktree"
)

func newAutoflowWorktreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Worktree bootstrap helpers",
	}
	cmd.AddCommand(newAutoflowWorktreeBootstrapCmd())
	return cmd
}

func newAutoflowWorktreeBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap <worktree-path>",
		Short: "Copy .claude + config files and run install/verify inside a worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mainDir, err := state.RepoRoot()
			if err != nil {
				return err
			}
			// Prompt the user on unknown binaries only when stdin is a TTY.
			prompter := worktree.Prompter(worktree.NonInteractivePrompter{})
			if isTerminal(os.Stdin) {
				prompter = worktree.InteractivePrompter{In: os.Stdin, Out: cmd.OutOrStdout()}
			}
			return worktree.Bootstrap(worktree.BootstrapOptions{
				MainDir:     mainDir,
				WorktreeDir: args[0],
				Prompter:    prompter,
				Stdout:      cmd.OutOrStdout(),
				Stderr:      cmd.ErrOrStderr(),
			})
		},
	}
}

// isTerminal is a minimal TTY probe — good enough for "is this interactive"
// without pulling in golang.org/x/term. We check whether Stat on stdin
// reports CharDevice, which is the usual Unix TTY signal.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
