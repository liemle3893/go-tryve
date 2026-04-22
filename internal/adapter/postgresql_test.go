package adapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/liemle3893/autoflow/internal/adapter"
	"github.com/liemle3893/autoflow/internal/core"
)

// TestNewPostgreSQLAdapter_Defaults verifies that the constructor returns a
// non-nil adapter and that the name is set correctly when only the required
// connectionString is supplied.
func TestNewPostgreSQLAdapter_Defaults(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
	})
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
	if a.Name() != "postgresql" {
		t.Fatalf("expected name %q, got %q", "postgresql", a.Name())
	}
}

// TestNewPostgreSQLAdapter_FullConfig verifies that all optional configuration
// fields are accepted without error.
func TestNewPostgreSQLAdapter_FullConfig(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
		"schema":           "public",
		"poolSize":         float64(10),
	})
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

// TestNewPostgreSQLAdapter_PoolSizeInt verifies that an integer poolSize value
// is accepted alongside the float64 representation produced by JSON/YAML unmarshalling.
func TestNewPostgreSQLAdapter_PoolSizeInt(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
		"poolSize":         int(3),
	})
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

// TestNewPostgreSQLAdapter_EmptyConfig verifies that construction succeeds even
// when no fields are provided; the missing connectionString error surfaces only
// on Connect.
func TestNewPostgreSQLAdapter_EmptyConfig(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{})
	if a == nil {
		t.Fatal("expected non-nil adapter from empty config")
	}
}

// TestPostgreSQLAdapter_InvalidAction verifies that an unsupported action name
// returns a *core.Error with code ADAPTER_ERROR without requiring a live
// database connection.
func TestPostgreSQLAdapter_InvalidAction(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
	})

	_, err := a.Execute(context.Background(), "unsupported_action", map[string]any{
		"sql": "SELECT 1",
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

// TestPostgreSQLAdapter_MissingSQLParam_Execute verifies that a missing "sql"
// param returns ADAPTER_ERROR for the "execute" action.
func TestPostgreSQLAdapter_MissingSQLParam_Execute(t *testing.T) {
	assertMissingSQLParam(t, "execute")
}

// TestPostgreSQLAdapter_MissingSQLParam_Query verifies that a missing "sql"
// param returns ADAPTER_ERROR for the "query" action.
func TestPostgreSQLAdapter_MissingSQLParam_Query(t *testing.T) {
	assertMissingSQLParam(t, "query")
}

// TestPostgreSQLAdapter_MissingSQLParam_QueryOne verifies that a missing "sql"
// param returns ADAPTER_ERROR for the "queryOne" action.
func TestPostgreSQLAdapter_MissingSQLParam_QueryOne(t *testing.T) {
	assertMissingSQLParam(t, "queryOne")
}

// TestPostgreSQLAdapter_MissingSQLParam_Count verifies that a missing "sql"
// param returns ADAPTER_ERROR for the "count" action.
func TestPostgreSQLAdapter_MissingSQLParam_Count(t *testing.T) {
	assertMissingSQLParam(t, "count")
}

// assertMissingSQLParam is a shared helper that verifies an absent "sql" param
// produces an ADAPTER_ERROR for the given action name.
func assertMissingSQLParam(t *testing.T, action string) {
	t.Helper()

	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
	})

	_, err := a.Execute(context.Background(), action, map[string]any{})
	if err == nil {
		t.Fatalf("action %q: expected error for missing sql param, got nil", action)
	}

	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("action %q: expected *core.Error, got %T: %v", action, err, err)
	}
	if coreErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("action %q: expected code ADAPTER_ERROR, got %s", action, coreErr.Code)
	}
}

// TestPostgreSQLAdapter_WrongSQLParamType verifies that a non-string "sql"
// value produces an ADAPTER_ERROR.
func TestPostgreSQLAdapter_WrongSQLParamType(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
	})

	_, err := a.Execute(context.Background(), "query", map[string]any{
		"sql": 42, // intentionally wrong type
	})
	if err == nil {
		t.Fatal("expected error for wrong sql param type, got nil")
	}

	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected *core.Error, got %T: %v", err, err)
	}
	if coreErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("expected code ADAPTER_ERROR, got %s", coreErr.Code)
	}
}

// TestPostgreSQLAdapter_EmptySQLParam verifies that an empty "sql" string
// produces an ADAPTER_ERROR.
func TestPostgreSQLAdapter_EmptySQLParam(t *testing.T) {
	a := adapter.NewPostgreSQLAdapter(map[string]any{
		"connectionString": "postgres://user:pass@localhost/testdb",
	})

	_, err := a.Execute(context.Background(), "execute", map[string]any{
		"sql": "",
	})
	if err == nil {
		t.Fatal("expected error for empty sql param, got nil")
	}

	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected *core.Error, got %T: %v", err, err)
	}
	if coreErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("expected code ADAPTER_ERROR, got %s", coreErr.Code)
	}
}
