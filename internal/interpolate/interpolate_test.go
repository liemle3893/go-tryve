package interpolate_test

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/interpolate"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

// newCtx creates a fresh InterpolationContext for tests.
func newCtx() *tryve.InterpolationContext {
	return tryve.NewInterpolationContext()
}

// TestResolve_SimpleVariable verifies that {{name}} is replaced from Variables.
func TestResolve_SimpleVariable(t *testing.T) {
	ctx := newCtx()
	ctx.Variables["name"] = "alice"
	got, err := interpolate.ResolveString("hello {{name}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello alice" {
		t.Errorf("expected %q, got %q", "hello alice", got)
	}
}

// TestResolve_DollarBraceSyntax verifies that ${name} works the same as {{name}}.
func TestResolve_DollarBraceSyntax(t *testing.T) {
	ctx := newCtx()
	ctx.Variables["name"] = "bob"
	got, err := interpolate.ResolveString("hello ${name}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello bob" {
		t.Errorf("expected %q, got %q", "hello bob", got)
	}
}

// TestResolve_CapturedValue verifies that {{captured.userId}} reads from Captured.
func TestResolve_CapturedValue(t *testing.T) {
	ctx := newCtx()
	ctx.Captured["userId"] = "user-42"
	got, err := interpolate.ResolveString("{{captured.userId}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "user-42" {
		t.Errorf("expected %q, got %q", "user-42", got)
	}
}

// TestResolve_BuiltinFunction verifies that {{$upper(hello)}} resolves via CallBuiltin.
func TestResolve_BuiltinFunction(t *testing.T) {
	ctx := newCtx()
	got, err := interpolate.ResolveString("{{$upper(hello)}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "HELLO" {
		t.Errorf("expected %q, got %q", "HELLO", got)
	}
}

// TestResolve_BaseURL verifies that {{baseUrl}} resolves to ctx.BaseURL.
func TestResolve_BaseURL(t *testing.T) {
	ctx := newCtx()
	ctx.BaseURL = "https://api.example.com"
	got, err := interpolate.ResolveString("{{baseUrl}}/api", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://api.example.com/api" {
		t.Errorf("expected %q, got %q", "https://api.example.com/api", got)
	}
}

// TestResolve_NestedVariable verifies multi-pass resolution:
// greeting = "hello {{name}}", name = "world" → greeting resolves to "hello world".
func TestResolve_NestedVariable(t *testing.T) {
	ctx := newCtx()
	ctx.Variables["name"] = "world"
	ctx.Variables["greeting"] = "hello {{name}}"
	got, err := interpolate.ResolveString("{{greeting}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

// TestResolve_EnvVariable verifies that {{$env(VAR)}} reads from the OS environment.
func TestResolve_EnvVariable(t *testing.T) {
	t.Setenv("TEST_INTERP_VAR", "env_value_123")
	ctx := newCtx()
	got, err := interpolate.ResolveString("{{$env(TEST_INTERP_VAR)}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env_value_123" {
		t.Errorf("expected %q, got %q", "env_value_123", got)
	}
}

// TestResolve_MapValues verifies that ResolveMap interpolates all string values recursively.
func TestResolve_MapValues(t *testing.T) {
	ctx := newCtx()
	ctx.Variables["host"] = "localhost"
	ctx.Variables["port"] = "8080"

	input := map[string]any{
		"url":    "http://{{host}}:{{port}}",
		"static": "no-interpolation",
		"nested": map[string]any{
			"path": "/api/{{host}}",
		},
	}

	result, err := interpolate.ResolveMap(input, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["url"] != "http://localhost:8080" {
		t.Errorf("url: expected %q, got %q", "http://localhost:8080", result["url"])
	}
	if result["static"] != "no-interpolation" {
		t.Errorf("static: expected %q, got %q", "no-interpolation", result["static"])
	}

	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map, got %T", result["nested"])
	}
	if nested["path"] != "/api/localhost" {
		t.Errorf("nested.path: expected %q, got %q", "/api/localhost", nested["path"])
	}
}

// TestResolve_UnknownVariable_LeftAsIs verifies that {{unknown}} is left as-is when not found.
func TestResolve_UnknownVariable_LeftAsIs(t *testing.T) {
	ctx := newCtx()
	got, err := interpolate.ResolveString("{{unknown}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{{unknown}}" {
		t.Errorf("expected expression to be left as-is, got %q", got)
	}
}

// TestResolveVariables_TopologicalOrder verifies that ResolveVariables resolves vars
// that reference other vars in the correct dependency order.
func TestResolveVariables_TopologicalOrder(t *testing.T) {
	ctx := newCtx()
	vars := map[string]any{
		"greeting": "hello {{name}}",
		"name":     "world",
	}

	resolved, err := interpolate.ResolveVariables(vars, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved["greeting"] != "hello world" {
		t.Errorf("greeting: expected %q, got %q", "hello world", resolved["greeting"])
	}
	if resolved["name"] != "world" {
		t.Errorf("name: expected %q, got %q", "world", resolved["name"])
	}
}

// TestResolveVariables_CircularDependency verifies that circular deps return an error.
func TestResolveVariables_CircularDependency(t *testing.T) {
	ctx := newCtx()
	vars := map[string]any{
		"a": "{{b}}",
		"b": "{{a}}",
	}

	_, err := interpolate.ResolveVariables(vars, ctx)
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
}

// TestResolve_SliceValues verifies that ResolveSlice interpolates string elements.
func TestResolve_SliceValues(t *testing.T) {
	ctx := newCtx()
	ctx.Variables["item"] = "apple"

	input := []any{
		"{{item}}",
		"static",
		42,
	}

	result, err := interpolate.ResolveSlice(input, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result[0] != "apple" {
		t.Errorf("element 0: expected %q, got %v", "apple", result[0])
	}
	if result[1] != "static" {
		t.Errorf("element 1: expected %q, got %v", "static", result[1])
	}
	if result[2] != 42 {
		t.Errorf("element 2: expected 42, got %v", result[2])
	}
}
