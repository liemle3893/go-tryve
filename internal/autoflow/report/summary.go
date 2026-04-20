package report

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ChangeRow is one entry in the "Changes" table.
type ChangeRow struct {
	Path   string
	Action string // "Created" | "Modified"
}

// ParseChangesFromSummary reads SUMMARY.md and extracts file lists from
// the `  created:` and `  modified:` YAML-style sections. Returns an
// empty slice when the file is absent or lacks those sections.
func ParseChangesFromSummary(summaryPath string) ([]ChangeRow, error) {
	f, err := os.Open(summaryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var rows []ChangeRow
	section := ""
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		switch {
		case indentedKey(line, "created"):
			section = "Created"
			continue
		case indentedKey(line, "modified"):
			section = "Modified"
			continue
		case startsWithLowercase(line):
			// new top-level key — end of list section
			section = ""
			continue
		}
		if section == "" {
			continue
		}
		path := extractListItem(line)
		if path == "" || path == "[]" {
			continue
		}
		rows = append(rows, ChangeRow{Path: path, Action: section})
	}
	return rows, s.Err()
}

// FindSummaryFile returns the best match under summaryDir. Prefers
// SUMMARY.md, falls back to *-SUMMARY.md for gsd-quick compatibility.
// Returns "" when none exists.
func FindSummaryFile(summaryDir string) string {
	if summaryDir == "" {
		return ""
	}
	exact := filepath.Join(summaryDir, "SUMMARY.md")
	if _, err := os.Stat(exact); err == nil {
		return exact
	}
	entries, err := os.ReadDir(summaryDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "-SUMMARY.md") {
			return filepath.Join(summaryDir, e.Name())
		}
	}
	return ""
}

// ExtractOneLiner returns the first non-empty line under ## One-liner,
// ## Summary, or (falling back to brief) ## Description. Truncated at
// 200 characters to match the bash helper.
func ExtractOneLiner(summaryPath, briefPath string) string {
	for _, probe := range []struct {
		path    string
		headers []string
	}{
		{summaryPath, []string{"## One-liner", "## Summary"}},
		{briefPath, []string{"## Description"}},
	} {
		if probe.path == "" {
			continue
		}
		if line := firstLineUnder(probe.path, probe.headers); line != "" {
			if len(line) > 200 {
				line = line[:197] + "..."
			}
			return line
		}
	}
	return "See PR for details"
}

// ExtractUsageSection returns the content of `## Usage`, `## API Changes`
// or `## Breaking Changes` from SUMMARY.md, concatenated. Defaults to a
// placeholder when not found.
func ExtractUsageSection(summaryPath string) string {
	if summaryPath == "" {
		return "No API changes."
	}
	f, err := os.Open(summaryPath)
	if err != nil {
		return "No API changes."
	}
	defer f.Close()

	var buf strings.Builder
	inSection := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "## Usage" || trimmed == "## API Changes" || trimmed == "## Breaking Changes":
			inSection = true
			continue
		case strings.HasPrefix(trimmed, "## "):
			inSection = false
		}
		if inSection {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		return "No API changes."
	}
	return out
}

// indentedKey returns true when line starts with whitespace + key + ":".
func indentedKey(line, key string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) == len(line) {
		return false // no indent
	}
	return strings.HasPrefix(trimmed, key+":")
}

// startsWithLowercase checks whether the first non-space char is a
// lowercase letter — SUMMARY.md top-level keys are all lowercase.
func startsWithLowercase(line string) bool {
	for _, r := range line {
		if r == ' ' || r == '\t' {
			continue
		}
		return r >= 'a' && r <= 'z'
	}
	return false
}

// extractListItem parses `  - <path>` → "<path>".
func extractListItem(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "- ") {
		return ""
	}
	return strings.TrimSpace(trimmed[2:])
}

// firstLineUnder returns the first non-empty line that appears after one
// of the given headers, or "". Scans sequentially.
func firstLineUnder(path string, headers []string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	matching := ""
	for s.Scan() {
		line := strings.TrimRight(s.Text(), " \t")
		if matching != "" {
			if strings.TrimSpace(line) == "" {
				continue
			}
			return strings.TrimSpace(line)
		}
		for _, h := range headers {
			if strings.TrimSpace(line) == h {
				matching = h
				break
			}
		}
	}
	return ""
}
