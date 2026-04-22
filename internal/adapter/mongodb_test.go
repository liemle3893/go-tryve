package adapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/liemle3893/autoflow/internal/adapter"
	"github.com/liemle3893/autoflow/internal/core"
)

// TestMongoDBAdapter_Constructor verifies that NewMongoDBAdapter parses
// "connectionString" and "database" from the config map and that Name()
// returns the expected identifier.
func TestMongoDBAdapter_Constructor(t *testing.T) {
	cfg := map[string]any{
		"connectionString": "mongodb://localhost:27017",
		"database":         "testdb",
	}
	a := adapter.NewMongoDBAdapter(cfg)

	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
	if a.Name() != "mongodb" {
		t.Fatalf("expected Name() == %q, got %q", "mongodb", a.Name())
	}
}

// TestMongoDBAdapter_Constructor_MissingKeys verifies that NewMongoDBAdapter
// handles missing config keys without panicking (defaults to empty strings).
func TestMongoDBAdapter_Constructor_MissingKeys(t *testing.T) {
	a := adapter.NewMongoDBAdapter(map[string]any{})
	if a == nil {
		t.Fatal("expected non-nil adapter even with empty config")
	}
	if a.Name() != "mongodb" {
		t.Fatalf("expected Name() == %q, got %q", "mongodb", a.Name())
	}
}

// TestMongoDBAdapter_Constructor_NonStringValues verifies that non-string config
// values are silently ignored and the adapter is still constructed.
func TestMongoDBAdapter_Constructor_NonStringValues(t *testing.T) {
	cfg := map[string]any{
		"connectionString": 12345,
		"database":         true,
	}
	a := adapter.NewMongoDBAdapter(cfg)
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

// TestMongoDBAdapter_InvalidAction verifies that Execute returns an ADAPTER_ERROR
// when an unsupported action name is given, without requiring a real connection.
func TestMongoDBAdapter_InvalidAction(t *testing.T) {
	// Use a valid URI format so Connect does not fail at parse time (mongo-driver v2
	// validates the URI on Connect, not on Execute).
	a := adapter.NewMongoDBAdapter(map[string]any{
		"connectionString": "mongodb://localhost:27017",
		"database":         "testdb",
	})

	ctx := context.Background()
	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	_, err := a.Execute(ctx, "unsupportedAction", map[string]any{
		"collection": "mycollection",
	})
	if err == nil {
		t.Fatal("expected error for unsupported action, got nil")
	}

	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected *core.Error, got %T: %v", err, err)
	}
	if coreErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("expected code ADAPTER_ERROR, got %s", coreErr.Code)
	}
}

// TestMongoDBAdapter_MissingCollection verifies that Execute returns an
// ADAPTER_ERROR when the required "collection" param is absent.
func TestMongoDBAdapter_MissingCollection(t *testing.T) {
	a := adapter.NewMongoDBAdapter(map[string]any{
		"connectionString": "mongodb://localhost:27017",
		"database":         "testdb",
	})

	ctx := context.Background()
	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer a.Close(ctx) //nolint:errcheck

	// Try each action without a collection param.
	actions := []string{
		"insertOne", "insertMany", "findOne", "find",
		"updateOne", "updateMany", "deleteOne", "deleteMany",
		"count", "aggregate",
	}
	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			_, err := a.Execute(ctx, action, map[string]any{})
			if err == nil {
				t.Fatalf("action %q: expected error for missing collection, got nil", action)
			}

			var coreErr *core.Error
			if !errors.As(err, &coreErr) {
				t.Fatalf("action %q: expected *core.Error, got %T: %v", action, err, err)
			}
			if coreErr.Code != "ADAPTER_ERROR" {
				t.Fatalf("action %q: expected code ADAPTER_ERROR, got %s", action, coreErr.Code)
			}
		})
	}
}
