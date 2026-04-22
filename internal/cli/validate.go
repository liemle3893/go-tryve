package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/liemle3893/autoflow/internal/loader"
)

// newValidateCmd constructs the `validate` sub-command which parses and
// validates every discovered YAML test file and reports any errors.
func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Parse and validate YAML test files without running them",
		Args:  cobra.NoArgs,
		RunE:  validateCmdHandler,
	}
	cmd.Flags().StringP("test-dir", "d", "tests", "directory to search for test files")
	return cmd
}

// validateCmdHandler implements the `validate` command execution logic.
func validateCmdHandler(cmd *cobra.Command, _ []string) error {
	testDir := resolveTestDir(cmd)

	paths, err := loader.Discover(testDir)
	if err != nil {
		return fmt.Errorf("discovering tests in %q: %w", testDir, err)
	}

	if len(paths) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No test files found.")
		return nil
	}

	hasError := false
	for _, p := range paths {
		td, parseErr := loader.ParseFile(p)
		if parseErr != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "FAIL  %s\n  parse error: %v\n", p, parseErr)
			hasError = true
			continue
		}
		if errs := loader.Validate(td); len(errs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "FAIL  %s\n", p)
			for _, ve := range errs {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %v\n", ve)
			}
			hasError = true
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "OK    %s\n", p)
	}

	if hasError {
		os.Exit(1)
	}
	return nil
}
