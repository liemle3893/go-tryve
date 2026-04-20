package deliver

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// BriefMetadata is the machine-readable shape lifted from task-brief.md.
// The jira-fetcher agent writes this frontmatter; step functions read it
// to drive path-A vs path-B, title extraction, etc.
type BriefMetadata map[string]string

// ParseBrief reads path and returns a flat key→value map of the file's
// frontmatter OR bare `key: value` lines under the title heading. Keys
// are lowercased and have spaces replaced by underscores.
//
// Supports two shapes in the same file:
//
//  1. YAML frontmatter — a `---` delimited block at the top of the file.
//  2. Bare `key: value` lines after the first `# Title` heading, up to
//     the next heading or blank line.
//
// Missing file → empty map, no error.
func ParseBrief(path string) (BriefMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return BriefMetadata{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := BriefMetadata{}
	inFrontmatter := false
	seenContent := false

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		trimmed := strings.TrimSpace(line)

		// YAML frontmatter delimiter — only honour `---` at the start
		// of the file. A second `---` closes the block.
		if trimmed == "---" && !seenContent {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break // end of frontmatter
		}
		if inFrontmatter {
			if i := strings.IndexByte(trimmed, ':'); i > 0 {
				k := normaliseKey(trimmed[:i])
				v := unquoteValue(strings.TrimSpace(trimmed[i+1:]))
				out[k] = v
			}
			continue
		}

		// Bare shape — once past the # Title heading, consume key:
		// value lines until a blank line or new heading.
		if !seenContent && strings.HasPrefix(trimmed, "# ") {
			seenContent = true
			continue
		}
		if seenContent && (strings.HasPrefix(trimmed, "#") || trimmed == "") {
			break
		}
		if i := strings.IndexByte(trimmed, ':'); i > 0 {
			seenContent = true
			k := normaliseKey(trimmed[:i])
			v := unquoteValue(strings.TrimSpace(trimmed[i+1:]))
			out[k] = v
		}
	}
	return out, s.Err()
}

func normaliseKey(k string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(k)), " ", "_")
}

func unquoteValue(v string) string {
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
