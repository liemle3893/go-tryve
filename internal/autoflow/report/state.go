// Package report generates the three markdown delivery reports
// (PR-BODY.md, JIRA-COMMENT.md, EXECUTION-REPORT.md) consumed by a human
// reviewer + the Jira upload step of autoflow-deliver. Replaces
// skills/autoflow-deliver/scripts/generate-report.sh.
package report

import (
	"encoding/json"
	"os"
)

// LoopSummary is the condensed form a loop state file is reduced to for
// both the loop table and the overall status line.
type LoopSummary struct {
	Rounds     int
	LastStatus string // "NONE" when no rounds, "SKIPPED" when file missing
	Present    bool   // true when the state file was found
}

// ParseLoopSummary reads path and returns counts + last status. A missing
// file is SKIPPED — matches the bash loop_summary helper.
func ParseLoopSummary(path string) LoopSummary {
	data, err := os.ReadFile(path)
	if err != nil {
		return LoopSummary{LastStatus: "SKIPPED"}
	}
	var s struct {
		Rounds []struct {
			Status string `json:"status"`
		} `json:"rounds"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return LoopSummary{LastStatus: "SKIPPED", Present: true}
	}
	out := LoopSummary{Rounds: len(s.Rounds), Present: true}
	if len(s.Rounds) == 0 {
		out.LastStatus = "NONE"
	} else {
		out.LastStatus = s.Rounds[len(s.Rounds)-1].Status
	}
	return out
}

// RoundDetail is one row of the detail table produced for JIRA-COMMENT.md
// and EXECUTION-REPORT.md.
type RoundDetail struct {
	Round     int             `json:"round"`
	Timestamp string          `json:"timestamp"`
	Status    string          `json:"status"`
	Problems  []GenericEntry  `json:"problems"`
	Fixes     []GenericEntry  `json:"fixes"`
	Raw       json.RawMessage `json:"-"` // preserved for rare custom fields
}

// GenericEntry is a loose shape that works across the different loops:
// coverage-review stores {ac, description}, e2e-fix stores {description,
// test_id}, enhance-loop uses {description, file, line, severity,
// category}. The fields here cover all of them.
type GenericEntry struct {
	AC          string `json:"ac,omitempty"`
	Description string `json:"description,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Category    string `json:"category,omitempty"`
	File        string `json:"file,omitempty"`
	Line        any    `json:"line,omitempty"`
	Action      string `json:"action,omitempty"`
	Commit      string `json:"commit,omitempty"`
}

// ReadRounds returns per-round detail from a state file. Missing files
// produce an empty slice, not an error — downstream report code will
// render them as SKIPPED.
func ReadRounds(path string) ([]RoundDetail, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s struct {
		Rounds []RoundDetail `json:"rounds"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s.Rounds, nil
}
