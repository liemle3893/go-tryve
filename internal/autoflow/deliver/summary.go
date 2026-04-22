package deliver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// WriteSummary renders SUMMARY.md at <ticket-dir>/SUMMARY.md from the
// current PLAN.md + plan-tasks.json. Called by the controller once every
// task is marked done, so no executor round-trip is needed just to
// assemble the summary.
func WriteSummary(root, key string) (string, error) {
	tdir := state.TicketDir(root, key)
	planPath := filepath.Join(tdir, "PLAN.md")
	summaryPath := filepath.Join(tdir, "SUMMARY.md")

	plan, err := ParsePlan(planPath)
	if err != nil {
		return "", fmt.Errorf("summary: parse plan: %w", err)
	}
	ps, err := state.ReadPlanState(root, key)
	if err != nil {
		return "", fmt.Errorf("summary: read state: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s: Implementation Summary\n\n", key)
	fmt.Fprintf(&b, "Generated: %s\n\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(&b, "Plan: `%s`\n\n", planPath)

	// Task table in plan order.
	fmt.Fprintln(&b, "## Tasks")
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "| ID | Name | Status | Commit |")
	fmt.Fprintln(&b, "|---|---|---|---|")
	for _, t := range plan {
		rec := ps.Tasks[t.ID]
		commit := rec.Commit
		if commit == "" {
			commit = "—"
		} else if len(commit) > 10 {
			commit = commit[:10]
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			t.ID, escapeMD(t.Name), rec.Status, commit)
	}

	// Commit list, chronological by task.
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "## Commits")
	fmt.Fprintln(&b, "")
	type row struct {
		id, sha, when string
	}
	var rows []row
	for _, t := range plan {
		rec := ps.Tasks[t.ID]
		if rec.Commit == "" {
			continue
		}
		rows = append(rows, row{id: t.ID, sha: rec.Commit, when: rec.EndedAt})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].when < rows[j].when })
	if len(rows) == 0 {
		fmt.Fprintln(&b, "(no commits recorded)")
	} else {
		for _, r := range rows {
			fmt.Fprintf(&b, "- `%s` — %s @ %s\n", r.sha, r.id, r.when)
		}
	}

	if err := os.WriteFile(summaryPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return summaryPath, nil
}

// escapeMD hides pipe chars so task names don't break the table layout.
func escapeMD(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}
