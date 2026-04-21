package state

import (
	"testing"
)

func TestInitProgress_SeedsStep1Timing(t *testing.T) {
	root := t.TempDir()
	p, err := InitProgress(root, "PROJ-1", "/wt", "b", false)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.StepTimings["1"]
	if !ok {
		t.Fatalf("step 1 timing not seeded: %+v", p.StepTimings)
	}
	if got.StartedAt == "" || got.StartedAt != p.StartedAt {
		t.Errorf("step 1 started_at=%q, ticket started_at=%q", got.StartedAt, p.StartedAt)
	}
	if got.EndedAt != "" {
		t.Errorf("step 1 ended_at should be empty, got %q", got.EndedAt)
	}
}

func TestCompleteStep_StampsEndAndNextStart(t *testing.T) {
	root := t.TempDir()
	if _, err := InitProgress(root, "PROJ-1", "/wt", "b", false); err != nil {
		t.Fatal(err)
	}
	if err := CompleteStep(root, "PROJ-1", 1); err != nil {
		t.Fatal(err)
	}
	p, _ := ReadProgress(root, "PROJ-1")
	s1 := p.StepTimings["1"]
	if s1.EndedAt == "" {
		t.Error("step 1 ended_at unset after complete")
	}
	s2, ok := p.StepTimings["2"]
	if !ok || s2.StartedAt == "" {
		t.Errorf("step 2 started_at not seeded: %+v", s2)
	}
	if s2.EndedAt != "" {
		t.Error("step 2 ended_at should be empty")
	}
}

func TestCompleteStep_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	_, _ = InitProgress(root, "PROJ-1", "/wt", "b", false)
	_ = CompleteStep(root, "PROJ-1", 1)
	p1, _ := ReadProgress(root, "PROJ-1")
	firstEnd := p1.StepTimings["1"].EndedAt

	// Re-complete the same step — ended_at should NOT change.
	_ = CompleteStep(root, "PROJ-1", 1)
	p2, _ := ReadProgress(root, "PROJ-1")
	if p2.StepTimings["1"].EndedAt != firstEnd {
		t.Errorf("re-completing step 1 changed ended_at: %q → %q",
			firstEnd, p2.StepTimings["1"].EndedAt)
	}
}

func TestListTickets_FiltersToInitialised(t *testing.T) {
	root := t.TempDir()
	_, _ = InitProgress(root, "PROJ-1", "/wt", "b", false)
	_, _ = InitProgress(root, "FOO-99", "/wt", "b", false)

	got, err := ListTickets(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 tickets, got %v", got)
	}
	// Sorted.
	if got[0] != "FOO-99" || got[1] != "PROJ-1" {
		t.Errorf("unsorted: %v", got)
	}
}

func TestListTickets_EmptyTree(t *testing.T) {
	got, err := ListTickets(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}
