package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/loader"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

// ANSI escape codes for list output styling.
const (
	listBold      = "\033[1m"
	listRed       = "\033[31m"
	listGreen     = "\033[32m"
	listYellow    = "\033[33m"
	listBlue      = "\033[34m"
	listMagenta   = "\033[35m"
	listCyan      = "\033[36m"
	listDim       = "\033[2m"
	listReset     = "\033[0m"
	listBgRed     = "\033[41m"
	listBgYellow  = "\033[43m"
	listBgBlue    = "\033[44m"
	listBgMagenta = "\033[45m"
	listWhite     = "\033[97m"
)

// listStyler handles conditional ANSI styling based on NO_COLOR env var.
type listStyler struct {
	color bool
}

// styled wraps text in the given ANSI style code.
func (ls *listStyler) styled(text, style string) string {
	if !ls.color {
		return text
	}
	return style + text + listReset
}

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

	ls := &listStyler{color: os.Getenv("NO_COLOR") == ""}

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

	// Count tests by priority for the summary.
	prioCounts := map[tryve.TestPriority]int{}
	for _, td := range filtered {
		prioCounts[td.Priority]++
	}

	// Print header.
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s\n", ls.styled("Discovered Tests", listBold+listCyan))
	fmt.Fprintf(out, "  %s\n", ls.styled(strings.Repeat("─", 56), listDim))
	fmt.Fprintln(out)

	// Print each test.
	for _, td := range filtered {
		prioBadge := ls.priorityBadge(td.Priority)
		name := ls.styled(td.Name, listBold)
		tagStr := ls.formatTags(td.Tags)
		file := ls.styled(td.SourceFile, listDim)
		fmt.Fprintf(out, "  %s  %s  %s\n", prioBadge, name, tagStr)
		fmt.Fprintf(out, "       %s\n", file)
	}

	// Print summary footer.
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s\n", ls.styled(strings.Repeat("─", 56), listDim))

	summaryParts := []string{
		ls.styled(fmt.Sprintf("%d test(s)", len(filtered)), listBold),
	}
	if c := prioCounts[tryve.PriorityP0]; c > 0 {
		summaryParts = append(summaryParts, ls.styled(fmt.Sprintf("P0:%d", c), listRed))
	}
	if c := prioCounts[tryve.PriorityP1]; c > 0 {
		summaryParts = append(summaryParts, ls.styled(fmt.Sprintf("P1:%d", c), listYellow))
	}
	if c := prioCounts[tryve.PriorityP2]; c > 0 {
		summaryParts = append(summaryParts, ls.styled(fmt.Sprintf("P2:%d", c), listBlue))
	}
	if c := prioCounts[tryve.PriorityP3]; c > 0 {
		summaryParts = append(summaryParts, ls.styled(fmt.Sprintf("P3:%d", c), listDim))
	}
	if c := prioCounts[""]; c > 0 {
		summaryParts = append(summaryParts, ls.styled(fmt.Sprintf("unset:%d", c), listDim))
	}
	fmt.Fprintf(out, "  %s\n", strings.Join(summaryParts, "  "))
	fmt.Fprintln(out)

	return nil
}

// priorityBadge returns a coloured priority badge string.
func (ls *listStyler) priorityBadge(p tryve.TestPriority) string {
	label := string(p)
	if label == "" {
		label = "--"
	}
	padded := fmt.Sprintf(" %s ", label)
	switch p {
	case tryve.PriorityP0:
		return ls.styled(padded, listBold+listWhite+listBgRed)
	case tryve.PriorityP1:
		return ls.styled(padded, listBold+listBgYellow)
	case tryve.PriorityP2:
		return ls.styled(padded, listBold+listWhite+listBgBlue)
	case tryve.PriorityP3:
		return ls.styled(padded, listBold+listWhite+listBgMagenta)
	default:
		return ls.styled(padded, listDim)
	}
}

// formatTags returns a coloured, space-separated string of tag badges.
func (ls *listStyler) formatTags(tags []string) string {
	if len(tags) == 0 {
		return ls.styled("(no tags)", listDim)
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = ls.styled("#"+t, listMagenta)
	}
	return strings.Join(parts, " ")
}
