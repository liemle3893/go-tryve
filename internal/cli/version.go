package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd constructs the `version` sub-command which prints the binary
// version string and exits.
func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the autoflow version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "autoflow %s\n", version)
		},
	}
}
