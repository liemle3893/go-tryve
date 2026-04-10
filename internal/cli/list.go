package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/liemle3893/e2e-runner/internal/executor"
	"github.com/liemle3893/e2e-runner/internal/loader"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// newListCmd constructs the `list` sub-command which discovers, parses, and
// filters test files, then prints a summary of each matching test.
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered test files and their metadata",
		Args:  cobra.NoArgs,
		RunE:  listCmdHandler,
	}
	cmd.Flags().StringP("test-dir", "d", "tests", "directory to search for test files")
	cmd.Flags().StringP("grep", "g", "", "filter tests by name (regexp or substring)")
	cmd.Flags().StringSlice("tag", nil, "filter tests by tag (can be repeated)")
	cmd.Flags().String("priority", "", "filter tests by priority (P0, P1, P2, P3)")
	return cmd
}

// listCmdHandler implements the `list` command execution logic.
func listCmdHandler(cmd *cobra.Command, _ []string) error {
	testDir := resolveTestDir(cmd)
	grep, _ := cmd.Flags().GetString("grep")
	tags, _ := cmd.Flags().GetStringSlice("tag")
	priority, _ := cmd.Flags().GetString("priority")

	// Discover test files.
	paths, err := loader.Discover(testDir)
	if err != nil {
		return fmt.Errorf("discovering tests in %q: %w", testDir, err)
	}

	// Parse each discovered file; silently skip unparseable files.
	var tests []*tryve.TestDefinition
	for _, p := range paths {
		td, parseErr := loader.ParseFile(p)
		if parseErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "WARN  parse error %s: %v\n", p, parseErr)
			continue
		}
		tests = append(tests, td)
	}

	// Apply filters.
	filtered := executor.FilterTests(tests, executor.FilterOptions{
		Tags:     tags,
		Grep:     grep,
		Priority: priority,
	})

	out := cmd.OutOrStdout()
	for _, td := range filtered {
		tags := strings.Join(td.Tags, ", ")
		prio := string(td.Priority)
		if prio == "" {
			prio = "-"
		}
		if tags == "" {
			tags = "-"
		}
		fmt.Fprintf(out, "[%s] %s [%s] (%s)\n", prio, td.Name, tags, td.SourceFile)
	}

	fmt.Fprintf(out, "\n%d test(s) found\n", len(filtered))
	return nil
}
