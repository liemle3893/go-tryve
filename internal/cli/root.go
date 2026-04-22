// Package cli provides the top-level cobra command tree for the autoflow binary.
package cli

import "github.com/spf13/cobra"

// NewRoot builds and returns the root cobra command for the autoflow binary.
// Delivery-workflow commands sit at top level; the e2e test-runner surface
// lives under the `e2e` subtree.
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:          "autoflow",
		Short:        "autoflow — Jira-to-PR delivery workflow + YAML-driven E2E test runner",
		SilenceUsage: true,
	}

	root.PersistentFlags().StringP("config", "c", "e2e.config.yaml", "config file path")
	root.PersistentFlags().StringP("env", "e", "local", "environment name")

	root.AddCommand(
		// Delivery workflow (top-level peers)
		newAutoflowJiraCmd(),
		newAutoflowWorktreeCmd(),
		newAutoflowDeliverCmd(),
		newAutoflowLoopStateCmd(),
		newAutoflowScaffoldCmd(),
		newAutoflowDoctorCmd(),
		// E2E test-runner subtree
		newE2ECmd(),
		// Cross-cutting
		newInstallCmd(),
		newVersionCmd(version),
	)

	return root
}
