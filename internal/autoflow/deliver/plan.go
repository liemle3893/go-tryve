package deliver

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Task is one parsed entry from PLAN.md. The planner emits a <task>
// block per task; each block carries an id, dependencies (other task
// ids), the file list the executor should touch, and human-readable
// action/verify/done text.
//
// Deps is the set of task ids this task waits on. Empty = root. The
// controller only dispatches a task when every dep is marked done in
// plan-tasks.json.
type Task struct {
	ID     string
	Name   string
	Files  []string
	Deps   []string
	Action string
	Verify string
	Done   string
}

// ParsePlan reads PLAN.md at path and returns the tasks in declaration
// order. Validates: non-empty ids are unique, every dep resolves to a
// known id, and the dep graph is acyclic. Missing <id> is synthesised
// as task-N (1-based) so hand-written plans without ids still work.
func ParsePlan(path string) ([]Task, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan %s: %w", path, err)
	}
	blocks := extractTaskBlocks(string(raw))
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no <task> blocks found in %s", path)
	}

	tasks := make([]Task, 0, len(blocks))
	seen := map[string]bool{}
	for i, body := range blocks {
		t := Task{
			ID:     strings.TrimSpace(getTag(body, "id")),
			Name:   strings.TrimSpace(getTag(body, "name")),
			Action: strings.TrimSpace(getTag(body, "action")),
			Verify: strings.TrimSpace(getTag(body, "verify")),
			Done:   strings.TrimSpace(getTag(body, "done")),
			Files:  splitCSV(getTag(body, "files")),
			Deps:   splitCSV(getTag(body, "deps")),
		}
		if t.ID == "" {
			t.ID = fmt.Sprintf("task-%02d", i+1)
		}
		if seen[t.ID] {
			return nil, fmt.Errorf("duplicate task id %q in %s", t.ID, path)
		}
		seen[t.ID] = true
		tasks = append(tasks, t)
	}

	// Resolve deps: every referenced id must exist.
	for _, t := range tasks {
		for _, d := range t.Deps {
			if !seen[d] {
				return nil, fmt.Errorf("task %s depends on unknown task %q", t.ID, d)
			}
		}
	}

	if cycle := findCycle(tasks); cycle != "" {
		return nil, fmt.Errorf("dependency cycle: %s", cycle)
	}
	return tasks, nil
}

// taskBlockRE matches a single <task>…</task> block, non-greedy, across
// lines. Multiline mode so `.` matches newlines (equivalent to DOTALL).
var taskBlockRE = regexp.MustCompile(`(?s)<task>(.*?)</task>`)

func extractTaskBlocks(s string) []string {
	matches := taskBlockRE.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// getTag extracts the content of a <name>…</name> inside a task block.
// Returns empty string when the tag is missing — callers decide what
// "empty" means for each field.
func getTag(block, tag string) string {
	re := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(tag) + `>(.*?)</` + regexp.QuoteMeta(tag) + `>`)
	m := re.FindStringSubmatch(block)
	if len(m) == 0 {
		return ""
	}
	return m[1]
}

// splitCSV parses "a, b , c" into ["a","b","c"]. Empty/blank input
// returns nil so callers can use `len(x) == 0` uniformly.
func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// findCycle returns a human-readable description of any cycle it finds
// in the dep graph, or "" when the graph is acyclic. Uses DFS with a
// grey/black colouring.
func findCycle(tasks []Task) string {
	idx := map[string]*Task{}
	for i := range tasks {
		idx[tasks[i].ID] = &tasks[i]
	}
	const (
		white = 0
		grey  = 1
		black = 2
	)
	colour := map[string]int{}
	var stack []string
	var dfs func(id string) string
	dfs = func(id string) string {
		colour[id] = grey
		stack = append(stack, id)
		for _, d := range idx[id].Deps {
			switch colour[d] {
			case grey:
				// Found a back-edge → cycle.
				start := 0
				for i, s := range stack {
					if s == d {
						start = i
						break
					}
				}
				return strings.Join(append(append([]string{}, stack[start:]...), d), " → ")
			case white:
				if msg := dfs(d); msg != "" {
					return msg
				}
			}
		}
		stack = stack[:len(stack)-1]
		colour[id] = black
		return ""
	}
	for _, t := range tasks {
		if colour[t.ID] == white {
			if msg := dfs(t.ID); msg != "" {
				return msg
			}
		}
	}
	return ""
}
