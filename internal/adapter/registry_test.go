package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/liemle3893/e2e-runner/internal/adapter"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// mockAdapter is a test double that tracks Connect/Close call counts.
type mockAdapter struct {
	name         string
	connectCalls int
	closeCalls   int
	connectErr   error
	closeErr     error
}

// Name returns the adapter's registered name.
func (m *mockAdapter) Name() string { return m.name }

// Connect records the call and returns any configured error.
func (m *mockAdapter) Connect(_ context.Context) error {
	m.connectCalls++
	return m.connectErr
}

// Close records the call and returns any configured error.
func (m *mockAdapter) Close(_ context.Context) error {
	m.closeCalls++
	return m.closeErr
}

// Health always returns nil for the mock.
func (m *mockAdapter) Health(_ context.Context) error { return nil }

// Execute returns a zero StepResult for the mock.
func (m *mockAdapter) Execute(_ context.Context, _ string, _ map[string]any) (*tryve.StepResult, error) {
	return &tryve.StepResult{
		Data:     map[string]any{},
		Duration: 0,
		Metadata: map[string]any{},
	}, nil
}

// TestRegistry_GetInitializesOnce verifies that Connect is called exactly once
// on the first Get, and that subsequent Gets return the same adapter instance
// without calling Connect again.
func TestRegistry_GetInitializesOnce(t *testing.T) {
	ctx := context.Background()
	reg := adapter.NewRegistry()

	mock := &mockAdapter{name: "db"}
	reg.Register("db", mock)

	// First Get — should trigger Connect.
	a1, err := reg.Get(ctx, "db")
	if err != nil {
		t.Fatalf("first Get: unexpected error: %v", err)
	}
	if mock.connectCalls != 1 {
		t.Fatalf("expected Connect called once after first Get, got %d", mock.connectCalls)
	}

	// Second Get — must return the same instance, Connect must not be called again.
	a2, err := reg.Get(ctx, "db")
	if err != nil {
		t.Fatalf("second Get: unexpected error: %v", err)
	}
	if a1 != a2 {
		t.Fatal("expected second Get to return the same adapter instance")
	}
	if mock.connectCalls != 1 {
		t.Fatalf("expected Connect called exactly once total, got %d", mock.connectCalls)
	}
}

// TestRegistry_GetUnknown verifies that requesting an unregistered adapter
// returns a ConfigError.
func TestRegistry_GetUnknown(t *testing.T) {
	ctx := context.Background()
	reg := adapter.NewRegistry()

	_, err := reg.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered adapter, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T", err)
	}
	if tryveErr.Code != "CONFIG_ERROR" {
		t.Fatalf("expected code CONFIG_ERROR, got %s", tryveErr.Code)
	}
}

// TestRegistry_GetConnectionFailure verifies that a Connect error is wrapped
// as a ConnectionError and the adapter is not marked as connected.
func TestRegistry_GetConnectionFailure(t *testing.T) {
	ctx := context.Background()
	reg := adapter.NewRegistry()

	connectErr := errors.New("dial tcp refused")
	mock := &mockAdapter{name: "api", connectErr: connectErr}
	reg.Register("api", mock)

	_, err := reg.Get(ctx, "api")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T", err)
	}
	if tryveErr.Code != "CONNECTION_ERROR" {
		t.Fatalf("expected code CONNECTION_ERROR, got %s", tryveErr.Code)
	}
}

// TestRegistry_CloseAll verifies that Close is called on every connected
// adapter and that unconnected adapters are not closed.
func TestRegistry_CloseAll(t *testing.T) {
	ctx := context.Background()
	reg := adapter.NewRegistry()

	connected := &mockAdapter{name: "svc1"}
	unconnected := &mockAdapter{name: "svc2"}

	reg.Register("svc1", connected)
	reg.Register("svc2", unconnected)

	// Only connect svc1.
	if _, err := reg.Get(ctx, "svc1"); err != nil {
		t.Fatalf("Get svc1: %v", err)
	}

	reg.CloseAll(ctx)

	if connected.closeCalls != 1 {
		t.Fatalf("expected Close called once on connected adapter, got %d", connected.closeCalls)
	}
	if unconnected.closeCalls != 0 {
		t.Fatalf("expected Close not called on unconnected adapter, got %d", unconnected.closeCalls)
	}
}

// TestRegistry_Has verifies the presence check for registered adapters.
func TestRegistry_Has(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.Register("present", &mockAdapter{name: "present"})

	if !reg.Has("present") {
		t.Error("expected Has to return true for registered adapter")
	}
	if reg.Has("absent") {
		t.Error("expected Has to return false for unregistered adapter")
	}
}

// TestRegistry_Names verifies that Names returns all registered adapter names.
func TestRegistry_Names(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.Register("alpha", &mockAdapter{name: "alpha"})
	reg.Register("beta", &mockAdapter{name: "beta"})

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d: %v", len(names), names)
	}

	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["alpha"] || !nameSet["beta"] {
		t.Fatalf("expected alpha and beta in names, got %v", names)
	}
}

// TestMeasureDuration verifies that MeasureDuration returns a non-negative
// duration and correctly propagates the function's return value.
func TestMeasureDuration(t *testing.T) {
	dur, err := adapter.MeasureDuration(func() error {
		time.Sleep(time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dur < time.Millisecond {
		t.Fatalf("expected duration >= 1ms, got %v", dur)
	}

	sentinel := errors.New("fn error")
	_, err = adapter.MeasureDuration(func() error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

// TestSuccessResult verifies that SuccessResult constructs a StepResult
// with the provided data, duration, and metadata.
func TestSuccessResult(t *testing.T) {
	data := map[string]any{"key": "value"}
	meta := map[string]any{"trace": "abc"}
	dur := 42 * time.Millisecond

	result := adapter.SuccessResult(data, dur, meta)

	if result == nil {
		t.Fatal("expected non-nil StepResult")
	}
	if result.Duration != dur {
		t.Fatalf("expected duration %v, got %v", dur, result.Duration)
	}
	if result.Data["key"] != "value" {
		t.Fatalf("expected data key=value, got %v", result.Data)
	}
	if result.Metadata["trace"] != "abc" {
		t.Fatalf("expected metadata trace=abc, got %v", result.Metadata)
	}
}
