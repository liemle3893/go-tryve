package cli

import "github.com/spf13/cobra"

// newAutoflowCmd builds the `tryve autoflow ...` subtree — Jira/worktree/
// deliver/loop-state/scaffold/doctor commands ported from the
// winx-autoflow skill-set.
func newAutoflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autoflow",
		Short: "Jira-to-PR autoflow helpers (ticket → worktree → PR)",
		Long: `Autoflow ports the winx-autoflow skill scripts into the tryve
binary. Each sub-command is a first-class Go implementation — there are
no external shell scripts to install or maintain.

Commands:
  jira          Manage Jira config + upload/download attachments
  worktree      Bootstrap a git worktree with .claude infrastructure
  deliver       13-step delivery workflow controller
  loop-state    Generic agentic-loop state file manager
  scaffold-e2e  Generate E2E test stubs for a ticket
  doctor        Preflight checklist for autoflow dependencies`,
	}
	cmd.AddCommand(
		newAutoflowJiraCmd(),
		newAutoflowWorktreeCmd(),
		newAutoflowDeliverCmd(),
		newAutoflowLoopStateCmd(),
		newAutoflowScaffoldCmd(),
		newAutoflowDoctorCmd(),
	)
	return cmd
}
