// Package extract parses REVIEW-*.md artefacts produced by the
// code/simplify/rules reviewer agents into the structured round-data
// and feedback JSON that review-loop expects. Replaces
// skills/autoflow-deliver/scripts/extract-round-data.sh.
package extract

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Finding is one reviewer finding distilled from a REVIEW-*.md file.
type Finding struct {
	ID          string `json:"id"`
	Source      string `json:"source"`   // code | rules | simplify
	Severity    string `json:"severity"` // critical | warning | info
	Type        string `json:"type"`     // bug | design-concern
	Title       string `json:"title"`
	Disposition string `json:"disposition"` // pending | fixed | skipped
}

// RoundData is the shape review-loop expects via --round-data.
type RoundData struct {
	BugsFound           int               `json:"bugs_found"`
	DesignConcernsFound int               `json:"design_concerns_found"`
	FeedbackIDs         []string          `json:"feedback_ids"`
	Problems            []FindingRef      `json:"problems"`
	Fixes               []FindingRef      `json:"fixes"`
	_                   struct{}          `json:"-"` // keep the struct addressable
	Findings            []Finding         `json:"-"` // exposed to the caller, not serialised
	RawFeedback         []json.RawMessage `json:"-"` // unused — placeholder for future expansion
}

// FindingRef is the slimmer shape embedded in RoundData.problems/fixes.
type FindingRef struct {
	ID       string `json:"id"`
	Source   string `json:"source,omitempty"`
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title"`
}

// Inputs groups the five review paths passed by the caller. Any path may
// be empty — a missing file is treated as "reviewer did not run" and
// produces no findings.
type Inputs struct {
	ReviewCode     string // REVIEW-code.md
	ReviewRules    string // REVIEW-rules.md
	ReviewSimplify string // REVIEW-simplify.md
	ReviewFix      string // REVIEW-FIX.md (optional)
}

// Extract parses all review files and returns the aggregated structure.
func Extract(in Inputs) (*RoundData, error) {
	all := []Finding{}

	for _, src := range []struct {
		path   string
		source string
	}{
		{in.ReviewCode, "code"},
		{in.ReviewRules, "rules"},
		{in.ReviewSimplify, "simplify"},
	} {
		if src.path == "" {
			continue
		}
		found, err := findingsFromFile(src.path, src.source)
		if err != nil {
			return nil, err
		}
		all = append(all, found...)
	}

	if in.ReviewFix != "" {
		disp, err := dispositionsFromFix(in.ReviewFix)
		if err != nil {
			return nil, err
		}
		for i := range all {
			if d, ok := disp[all[i].ID]; ok {
				all[i].Disposition = d
			}
		}
	}

	rd := &RoundData{Findings: all}
	seenIDs := map[string]bool{}
	for _, f := range all {
		if !seenIDs[f.ID] {
			seenIDs[f.ID] = true
			rd.FeedbackIDs = append(rd.FeedbackIDs, f.ID)
		}
		switch f.Type {
		case "bug":
			rd.BugsFound++
			rd.Problems = append(rd.Problems, FindingRef{
				ID: f.ID, Source: f.Source, Severity: f.Severity, Title: f.Title,
			})
		case "design-concern":
			rd.DesignConcernsFound++
		}
		if f.Disposition == "fixed" {
			rd.Fixes = append(rd.Fixes, FindingRef{ID: f.ID, Title: f.Title})
		}
	}

	// Never return nil slices — reviewers downstream rely on JSON arrays.
	if rd.FeedbackIDs == nil {
		rd.FeedbackIDs = []string{}
	}
	if rd.Problems == nil {
		rd.Problems = []FindingRef{}
	}
	if rd.Fixes == nil {
		rd.Fixes = []FindingRef{}
	}
	return rd, nil
}

// WriteFiles serialises rd to two files: round-data-out and feedback-out.
// round-data-out contains bugs_found/design_concerns_found/feedback_ids/
// problems/fixes; feedback-out contains the full Finding list.
func WriteFiles(rd *RoundData, roundDataOut, feedbackOut string) error {
	if rd == nil {
		return errors.New("nil RoundData")
	}
	dataBytes, err := json.MarshalIndent(struct {
		BugsFound           int          `json:"bugs_found"`
		DesignConcernsFound int          `json:"design_concerns_found"`
		FeedbackIDs         []string     `json:"feedback_ids"`
		Problems            []FindingRef `json:"problems"`
		Fixes               []FindingRef `json:"fixes"`
	}{
		BugsFound:           rd.BugsFound,
		DesignConcernsFound: rd.DesignConcernsFound,
		FeedbackIDs:         rd.FeedbackIDs,
		Problems:            rd.Problems,
		Fixes:               rd.Fixes,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(roundDataOut, dataBytes, 0o644); err != nil {
		return err
	}

	fbBytes, err := json.MarshalIndent(rd.Findings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(feedbackOut, fbBytes, 0o644)
}

// frontmatterRe matches the YAML frontmatter block at the start of a file.
var frontmatterRe = regexp.MustCompile(`(?s)\A---\s*\n(.*?)\n---`)

// findingHeadingRe matches `### PREFIX-NN: Title` where PREFIX is one of
// the reviewer-assigned codes. Case-sensitive, one finding per line.
var findingHeadingRe = regexp.MustCompile(`^###\s+(([A-Z]+)-\d+):\s+(.*)$`)

// statusLineRe matches "status:" in the frontmatter.
var statusLineRe = regexp.MustCompile(`(?m)^status:\s*(\S+)`)

// Severity of each reviewer prefix. Matches extract-round-data.sh.
//
//	CR / RLC / SMC → critical (bug)
//	WR / RLW / SMW → warning  (bug)
//	IN / RLI / SMI → info     (design-concern)
var severityForPrefix = map[string]string{
	"CR": "critical", "RLC": "critical", "SMC": "critical",
	"WR": "warning", "RLW": "warning", "SMW": "warning",
	"IN": "info", "RLI": "info", "SMI": "info",
}

func findingsFromFile(path, source string) ([]Finding, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	status := frontmatterStatus(data)
	if status == "clean" || status == "skipped" {
		return nil, nil
	}

	var out []Finding
	for _, line := range strings.Split(string(data), "\n") {
		m := findingHeadingRe.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}
		id, prefix, title := m[1], m[2], m[3]
		severity, ok := severityForPrefix[prefix]
		if !ok {
			severity = "info"
		}
		fType := "bug"
		if severity == "info" {
			fType = "design-concern"
		}
		out = append(out, Finding{
			ID:          id,
			Source:      source,
			Severity:    severity,
			Type:        fType,
			Title:       strings.TrimSpace(title),
			Disposition: "pending",
		})
	}
	return out, nil
}

// frontmatterStatus returns the status field from the YAML frontmatter,
// lowercased. Empty string when not present.
func frontmatterStatus(data []byte) string {
	m := frontmatterRe.FindSubmatch(data)
	if len(m) < 2 {
		return ""
	}
	sm := statusLineRe.FindSubmatch(m[1])
	if len(sm) < 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(sm[1])))
}

func dispositionsFromFix(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	out := map[string]string{}
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "## Fixed"):
			section = "fixed"
		case strings.HasPrefix(line, "## Skipped"):
			section = "skipped"
		case strings.HasPrefix(line, "## "):
			section = ""
		}
		m := findingHeadingRe.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		id := m[1]
		if section == "fixed" {
			out[id] = "fixed"
		} else if section == "skipped" {
			out[id] = "skipped"
		}
	}
	return out, nil
}
