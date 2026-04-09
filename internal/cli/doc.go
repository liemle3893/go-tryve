package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// docSectionEntry represents a single entry in the docs/sections/index.json registry.
type docSectionEntry struct {
	File        string `json:"file"`
	Description string `json:"description"`
}

// newDocCmd constructs the `doc` sub-command which lists or displays embedded
// documentation sections from docs/sections/.
func newDocCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doc [section]",
		Short: "Show built-in documentation",
		Long: `Show built-in documentation for a named section.

Without arguments, lists all available sections.
With a section name (e.g. "assertions", "adapters.http"), prints the full
reference for that section.

Documentation is read from the docs/sections/ directory relative to the
working directory or the config file location.`,
		Args: cobra.MaximumNArgs(1),
		RunE: docCmdHandler,
	}
}

// docCmdHandler implements the `doc` command execution logic.
func docCmdHandler(cmd *cobra.Command, args []string) error {
	docsDir, err := resolveSectionsDir(cmd)
	if err != nil {
		return err
	}

	registry, err := loadDocRegistry(docsDir)
	if err != nil {
		return err
	}

	// No argument: list all available sections.
	if len(args) == 0 {
		return printDocSections(cmd, registry)
	}

	return printDocSection(cmd, docsDir, registry, args[0])
}

// resolveSectionsDir locates the docs/sections directory by searching from the
// config file's directory and then from the current working directory.
func resolveSectionsDir(cmd *cobra.Command) (string, error) {
	// Try config file location first.
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath != "" {
		candidate := filepath.Join(filepath.Dir(cfgPath), "docs", "sections")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	// Fall back to cwd.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("doc: cannot determine working directory: %w", err)
	}
	candidate := filepath.Join(cwd, "docs", "sections")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate, nil
	}

	return "", fmt.Errorf(
		"doc: docs/sections directory not found; run this command from the project root",
	)
}

// loadDocRegistry reads and parses docs/sections/index.json.
func loadDocRegistry(sectionsDir string) (map[string]docSectionEntry, error) {
	indexPath := filepath.Join(sectionsDir, "index.json")
	data, err := os.ReadFile(indexPath) //nolint:gosec // path is derived from controlled sources
	if err != nil {
		return nil, fmt.Errorf("doc: reading index: %w", err)
	}

	var registry map[string]docSectionEntry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("doc: parsing index.json: %w", err)
	}
	return registry, nil
}

// printDocSections writes a sorted list of all registered sections to cmd's
// output writer.
func printDocSections(cmd *cobra.Command, registry map[string]docSectionEntry) error {
	out := cmd.OutOrStdout()

	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxLen := 0
	for _, k := range keys {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	fmt.Fprintln(out, "Available documentation sections:")
	fmt.Fprintln(out, strings.Repeat("-", 50))
	for _, k := range keys {
		entry := registry[k]
		fmt.Fprintf(out, "  %-*s  %s\n", maxLen, k, entry.Description)
	}
	fmt.Fprintf(out, "\nRun `tryve doc <section>` to view a section.\n")
	return nil
}

// printDocSection reads the markdown file for the named section and writes its
// content to cmd's output writer.
func printDocSection(cmd *cobra.Command, sectionsDir string, registry map[string]docSectionEntry, section string) error {
	entry, ok := registry[section]
	if !ok {
		return fmt.Errorf("doc: unknown section %q; run `tryve doc` to list available sections", section)
	}

	docPath := filepath.Join(sectionsDir, filepath.FromSlash(entry.File))
	data, err := os.ReadFile(docPath) //nolint:gosec // path is derived from controlled sources
	if err != nil {
		return fmt.Errorf("doc: reading %s: %w", entry.File, err)
	}

	_, err = fmt.Fprint(cmd.OutOrStdout(), string(data))
	return err
}
