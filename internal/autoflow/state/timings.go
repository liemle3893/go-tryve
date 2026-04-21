package state

import (
	"os"
	"path/filepath"
	"slices"
	"time"
)

// ListTickets returns every ticket key under .autoflow/ticket/ that has a
// workflow-progress.json (i.e. has been initialised at least once).
// Returns nil with no error when the .autoflow tree doesn't exist.
func ListTickets(root string) ([]string, error) {
	base := filepath.Join(root, ".autoflow", "ticket")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var keys []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		key := e.Name()
		if ValidateTicketKey(key) != nil {
			continue
		}
		if _, err := os.Stat(ProgressFile(root, key)); err == nil {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys, nil
}

// TicketTotal returns the total elapsed time on a ticket, computed as
// max(ended_at) − started_at across recorded step timings. Returns 0 when
// no step has been completed yet.
func (p *Progress) TicketTotal() time.Duration {
	if p == nil || p.StartedAt == "" {
		return 0
	}
	start, err := time.Parse("2006-01-02T15:04:05Z", p.StartedAt)
	if err != nil {
		return 0
	}
	var latest time.Time
	for _, t := range p.StepTimings {
		if t.EndedAt == "" {
			continue
		}
		e, err := time.Parse("2006-01-02T15:04:05Z", t.EndedAt)
		if err != nil {
			continue
		}
		if e.After(latest) {
			latest = e
		}
	}
	if latest.IsZero() {
		return 0
	}
	d := latest.Sub(start).Round(time.Second)
	if d < 0 {
		return 0
	}
	return d
}
