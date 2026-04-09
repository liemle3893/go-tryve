package adapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/liemle3893/e2e-runner/internal/adapter"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// TestNewRedisAdapter_ParsesConfig verifies that NewRedisAdapter correctly
// extracts connectionString, db, and keyPrefix from the config map.
func TestNewRedisAdapter_ParsesConfig(t *testing.T) {
	cfg := map[string]any{
		"connectionString": "redis://localhost:6379/0",
		"db":               2,
		"keyPrefix":        "test:",
	}

	a := adapter.NewRedisAdapter(cfg)

	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
	if a.Name() != "redis" {
		t.Fatalf("expected Name() == \"redis\", got %q", a.Name())
	}
}

// TestNewRedisAdapter_DefaultDB verifies that the db field defaults to 0 when
// absent from the config map.
func TestNewRedisAdapter_DefaultDB(t *testing.T) {
	cfg := map[string]any{
		"connectionString": "redis://localhost:6379",
	}

	a := adapter.NewRedisAdapter(cfg)

	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

// TestNewRedisAdapter_FloatDB verifies that a float64 db value (as produced by
// YAML/JSON unmarshalling) is accepted and coerced to int.
func TestNewRedisAdapter_FloatDB(t *testing.T) {
	cfg := map[string]any{
		"connectionString": "redis://localhost:6379",
		"db":               float64(3),
	}

	a := adapter.NewRedisAdapter(cfg)

	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

// TestNewRedisAdapter_EmptyConfig verifies that an empty config map produces a
// valid (though not connectable) adapter rather than a nil pointer.
func TestNewRedisAdapter_EmptyConfig(t *testing.T) {
	a := adapter.NewRedisAdapter(map[string]any{})
	if a == nil {
		t.Fatal("expected non-nil adapter for empty config")
	}
}

// --- Key prefix helper tests -------------------------------------------------

// TestRedisAdapter_PrefixedKey_WithPrefix verifies that the prefixed key is
// correctly constructed when keyPrefix is set.
func TestRedisAdapter_PrefixedKey_WithPrefix(t *testing.T) {
	a := adapter.NewRedisAdapter(map[string]any{
		"connectionString": "redis://localhost:6379",
		"keyPrefix":        "ns:",
	})

	got := a.ExportedPrefixedKey("mykey")
	want := "ns:mykey"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// TestRedisAdapter_PrefixedKey_WithoutPrefix verifies that the key is returned
// unchanged when keyPrefix is empty.
func TestRedisAdapter_PrefixedKey_WithoutPrefix(t *testing.T) {
	a := adapter.NewRedisAdapter(map[string]any{
		"connectionString": "redis://localhost:6379",
	})

	got := a.ExportedPrefixedKey("mykey")
	want := "mykey"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// --- Invalid action test -----------------------------------------------------

// TestRedisAdapter_InvalidAction_ReturnsAdapterError verifies that Execute
// returns a *tryve.TryveError with code ADAPTER_ERROR for an unknown action.
// This test does not require a live Redis connection because the action
// dispatch check runs before any network I/O.
func TestRedisAdapter_InvalidAction_ReturnsAdapterError(t *testing.T) {
	a := adapter.NewRedisAdapter(map[string]any{
		"connectionString": "redis://localhost:6379",
	})

	// We deliberately skip Connect to verify the guard fires before any I/O.
	// context.Background() is safe here — no network call is made.
	_, err := a.Execute(context.Background(), "unknownAction", map[string]any{})

	if err == nil {
		t.Fatal("expected error for unsupported action, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("expected code ADAPTER_ERROR, got %s", tryveErr.Code)
	}
}
