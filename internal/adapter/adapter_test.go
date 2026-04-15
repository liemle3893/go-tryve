package adapter_test

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/adapter"
)

// TestCheckUnresolvedEnvVars_NoPlaceholders verifies that a fully resolved
// string produces no error.
func TestCheckUnresolvedEnvVars_NoPlaceholders(t *testing.T) {
	err := adapter.CheckUnresolvedEnvVars("test", "field", "postgres://user:pass@localhost/db")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// TestCheckUnresolvedEnvVars_SinglePlaceholder verifies that a single
// unresolved ${VAR} is detected and named in the error message.
func TestCheckUnresolvedEnvVars_SinglePlaceholder(t *testing.T) {
	err := adapter.CheckUnresolvedEnvVars("pg", "connectionString", "${PG_URL}")
	if err == nil {
		t.Fatal("expected error for unresolved placeholder, got nil")
	}
	msg := err.Error()
	if got := "PG_URL"; !containsStr(msg, got) {
		t.Errorf("error message %q should mention %q", msg, got)
	}
	if !containsStr(msg, ".env") {
		t.Errorf("error message %q should mention .env file", msg)
	}
}

// TestCheckUnresolvedEnvVars_MultiplePlaceholders verifies that all unresolved
// variables are listed.
func TestCheckUnresolvedEnvVars_MultiplePlaceholders(t *testing.T) {
	err := adapter.CheckUnresolvedEnvVars("db", "dsn", "${HOST}:${PORT}")
	if err == nil {
		t.Fatal("expected error for unresolved placeholders, got nil")
	}
	msg := err.Error()
	if !containsStr(msg, "HOST") || !containsStr(msg, "PORT") {
		t.Errorf("error message %q should mention both HOST and PORT", msg)
	}
}

// TestCheckUnresolvedEnvVars_EmptyString verifies that an empty string is fine.
func TestCheckUnresolvedEnvVars_EmptyString(t *testing.T) {
	err := adapter.CheckUnresolvedEnvVars("test", "field", "")
	if err != nil {
		t.Fatalf("expected nil for empty string, got %v", err)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
