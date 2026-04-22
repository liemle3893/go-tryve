package cli

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

func newDeliverTimingsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "timings",
		Short: "Show per-step durations for a ticket, or an aggregate report across all tickets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			key, _ := cmd.Flags().GetString("ticket")
			if key != "" {
				return printOneTicketTimings(cmd.OutOrStdout(), root, key)
			}
			return printAllTicketTimings(cmd.OutOrStdout(), root)
		},
	}
	c.Flags().String("ticket", "", "ticket key (omit to report across all tickets)")
	return c
}

func printOneTicketTimings(w io.Writer, root, key string) error {
	p, err := state.ReadProgress(root, key)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("no workflow-progress.json for %s", key)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STEP\tSTARTED\tENDED\tDURATION")

	// Iterate 1..MaxStep so output order is deterministic.
	for i := 1; i <= state.MaxStep; i++ {
		t, ok := p.StepTimings[strconv.Itoa(i)]
		if !ok {
			continue
		}
		end := t.EndedAt
		dur := formatDuration(time.Duration(t.DurationSeconds) * time.Second)
		if end == "" {
			end = "(in progress)"
			dur = "-"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", i, t.StartedAt, end, dur)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(w, "\nticket:   %s\n", p.Ticket)
	fmt.Fprintf(w, "started:  %s\n", p.StartedAt)
	fmt.Fprintf(w, "total:    %s\n", formatDuration(p.TicketTotal()))
	fmt.Fprintf(w, "progress: step %d / %d (%d completed)\n",
		p.CurrentStep, state.MaxStep, len(p.Completed))
	return nil
}

func printAllTicketTimings(w io.Writer, root string) error {
	keys, err := state.ListTickets(root)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		fmt.Fprintln(w, "No tickets found under .autoflow/ticket/.")
		return nil
	}

	type row struct {
		key       string
		started   string
		completed int
		current   int
		total     time.Duration
	}
	rows := make([]row, 0, len(keys))
	var grand time.Duration
	for _, k := range keys {
		p, err := state.ReadProgress(root, k)
		if err != nil || p == nil {
			continue
		}
		tot := p.TicketTotal()
		grand += tot
		rows = append(rows, row{
			key:       k,
			started:   p.StartedAt,
			completed: len(p.Completed),
			current:   p.CurrentStep,
			total:     tot,
		})
	}
	// Most recently started first.
	sort.Slice(rows, func(i, j int) bool { return rows[i].started > rows[j].started })

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TICKET\tSTARTED\tSTEP\tDONE\tTOTAL")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%d/%d\t%d\t%s\n",
			r.key, r.started, r.current, state.MaxStep, r.completed, formatDuration(r.total))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(w, "\n%d tickets, cumulative active time: %s\n", len(rows), formatDuration(grand))
	return nil
}

// formatDuration renders d as "1h23m45s" / "2m05s" / "12s". Returns "-"
// for zero durations so the timings table shows a placeholder instead of
// an ambiguous "0s" for steps that haven't run.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
