package assertion_test

import (
	"testing"

	"github.com/liemle3893/e2e-runner/internal/assertion"
)

// TestJSONPath_SimpleProperty verifies that a simple top-level property is retrieved correctly.
func TestJSONPath_SimpleProperty(t *testing.T) {
	data := map[string]any{"status": 200}
	got, ok := assertion.EvalJSONPath(data, "$.status")
	if !ok {
		t.Fatal("EvalJSONPath returned not found for $.status")
	}
	if got != 200 {
		t.Errorf("EvalJSONPath = %v, want 200", got)
	}
}

// TestJSONPath_NestedProperty verifies that a deeply nested property is retrieved correctly.
func TestJSONPath_NestedProperty(t *testing.T) {
	data := map[string]any{
		"body": map[string]any{
			"user": map[string]any{
				"id": 42,
			},
		},
	}
	got, ok := assertion.EvalJSONPath(data, "$.body.user.id")
	if !ok {
		t.Fatal("EvalJSONPath returned not found for $.body.user.id")
	}
	if got != 42 {
		t.Errorf("EvalJSONPath = %v, want 42", got)
	}
}

// TestJSONPath_ArrayIndex verifies that array index access returns the correct element.
func TestJSONPath_ArrayIndex(t *testing.T) {
	data := map[string]any{
		"items": []any{"a", "b", "c"},
	}
	got, ok := assertion.EvalJSONPath(data, "$.items[0]")
	if !ok {
		t.Fatal("EvalJSONPath returned not found for $.items[0]")
	}
	if got != "a" {
		t.Errorf("EvalJSONPath = %v, want \"a\"", got)
	}
}

// TestJSONPath_ArrayWildcard verifies that the wildcard operator collects named fields from all array elements.
func TestJSONPath_ArrayWildcard(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"name": "a"},
			map[string]any{"name": "b"},
		},
	}
	got, ok := assertion.EvalJSONPath(data, "$.items[*].name")
	if !ok {
		t.Fatal("EvalJSONPath returned not found for $.items[*].name")
	}
	list, ok2 := got.([]any)
	if !ok2 {
		t.Fatalf("expected []any result for wildcard, got %T: %v", got, got)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(list), list)
	}
	if list[0] != "a" || list[1] != "b" {
		t.Errorf("expected [a b], got %v", list)
	}
}

// TestJSONPath_RecursiveDescent verifies that recursive descent collects all matching fields.
func TestJSONPath_RecursiveDescent(t *testing.T) {
	data := map[string]any{
		"users": []any{
			map[string]any{"id": 1},
			map[string]any{"id": 2},
		},
	}
	got, ok := assertion.EvalJSONPath(data, "$..id")
	if !ok {
		t.Fatal("EvalJSONPath returned not found for $..id")
	}
	list, ok2 := got.([]any)
	if !ok2 {
		t.Fatalf("expected []any result for recursive descent, got %T: %v", got, got)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(list), list)
	}
	// IDs may come in any order; check both are present.
	found := map[any]bool{}
	for _, v := range list {
		found[v] = true
	}
	if !found[1] || !found[2] {
		t.Errorf("expected ids 1 and 2, got %v", list)
	}
}

// TestJSONPath_NotFound verifies that a missing key returns (nil, false).
func TestJSONPath_NotFound(t *testing.T) {
	data := map[string]any{"a": 1}
	got, ok := assertion.EvalJSONPath(data, "$.b")
	if ok {
		t.Errorf("expected not found for $.b, got %v", got)
	}
	if got != nil {
		t.Errorf("expected nil value for not-found path, got %v", got)
	}
}

// TestJSONPath_WithoutDollarPrefix verifies that paths without a leading "$" are auto-prefixed.
func TestJSONPath_WithoutDollarPrefix(t *testing.T) {
	data := map[string]any{"status": 200}
	got, ok := assertion.EvalJSONPath(data, "status")
	if !ok {
		t.Fatal("EvalJSONPath returned not found when using path without $ prefix")
	}
	if got != 200 {
		t.Errorf("EvalJSONPath = %v, want 200", got)
	}
}

// TestJSONPath_BracketNotation verifies that bracket notation with hyphenated keys works correctly.
func TestJSONPath_BracketNotation(t *testing.T) {
	data := map[string]any{"a-b": "value"}
	got, ok := assertion.EvalJSONPath(data, "$['a-b']")
	if !ok {
		t.Fatal("EvalJSONPath returned not found for $['a-b']")
	}
	if got != "value" {
		t.Errorf("EvalJSONPath = %v, want \"value\"", got)
	}
}

// TestHasJSONPath verifies that HasJSONPath reports existence correctly,
// including paths whose value is nil, and returns false for absent keys.
func TestHasJSONPath(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": nil,
		},
	}

	// Path to a key whose value is nil — should still report as existing.
	if !assertion.HasJSONPath(data, "$.a.b") {
		t.Error("HasJSONPath should return true for $.a.b even when value is nil")
	}

	// Path to a key that does not exist.
	if assertion.HasJSONPath(data, "$.a.c") {
		t.Error("HasJSONPath should return false for $.a.c which does not exist")
	}
}

// TestQueryJSONPath verifies that QueryJSONPath returns all matches for a path expression.
func TestQueryJSONPath(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 1},
			map[string]any{"id": 2},
			map[string]any{"id": 3},
		},
	}
	results := assertion.QueryJSONPath(data, "$.items[*].id")
	if len(results) != 3 {
		t.Fatalf("QueryJSONPath returned %d results, want 3: %v", len(results), results)
	}
}
