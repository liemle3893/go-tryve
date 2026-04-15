package interpolate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

const maxDepth = 10

// doubleBraceRe matches {{expression}} patterns.
var doubleBraceRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// dollarBraceRe matches ${expression} patterns.
var dollarBraceRe = regexp.MustCompile(`\$\{([^}]+)\}`)

// builtinCallRe matches $funcName or $funcName(args) inside an expression.
var builtinCallRe = regexp.MustCompile(`^\$(\w+)(?:\(([^)]*)\))?$`)

// ResolveString interpolates a single string against the given context.
// Both {{expression}} and ${expression} syntaxes are supported.
// Resolution is performed up to maxDepth passes until the result stabilises.
func ResolveString(s string, ctx *tryve.InterpolationContext) (string, error) {
	prev := ""
	cur := s
	for i := 0; i < maxDepth; i++ {
		prev = cur
		resolved, err := resolveOnce(cur, ctx)
		if err != nil {
			return "", err
		}
		cur = resolved
		if cur == prev {
			break
		}
	}
	return cur, nil
}

// resolveOnce performs a single pass of expression substitution over s.
// Unresolved expressions are left as {{expr}} (not treated as errors).
// Only errors from actual builtin/function execution are propagated.
func resolveOnce(s string, ctx *tryve.InterpolationContext) (string, error) {
	var firstErr error

	// doubleBracketReplacer replaces {{expr}} occurrences.
	doubleBracketReplacer := func(match string) string {
		if firstErr != nil {
			return match
		}
		inner := doubleBraceRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		expr := inner[1]
		result, err := resolveExpression(strings.TrimSpace(expr), ctx)
		if err != nil {
			if isUnresolved(err) {
				// Not found — leave the original {{expr}} token intact.
				return "{{" + expr + "}}"
			}
			firstErr = err
			return match
		}
		return result
	}

	out := doubleBraceRe.ReplaceAllStringFunc(s, doubleBracketReplacer)
	if firstErr != nil {
		return "", firstErr
	}

	// dollarBraceReplacer replaces ${expr} occurrences.
	dollarBraceReplacer := func(match string) string {
		if firstErr != nil {
			return match
		}
		inner := dollarBraceRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		expr := inner[1]
		result, err := resolveExpression(strings.TrimSpace(expr), ctx)
		if err != nil {
			if isUnresolved(err) {
				// Not found — leave the original ${expr} token intact.
				return "${" + expr + "}"
			}
			firstErr = err
			return match
		}
		return result
	}

	out = dollarBraceRe.ReplaceAllStringFunc(out, dollarBraceReplacer)
	if firstErr != nil {
		return "", firstErr
	}

	return out, nil
}

// resolveExpression resolves a single interpolation expression using the priority:
//  1. Built-in function (expression starts with $)
//  2. Literal "baseUrl"
//  3. "captured.<field>" prefix
//  4. Variable (ctx.Variables, dot-notation)
//  5. Environment (ctx.Env)
//  6. Not found — returns original {{expr}} sentinel so the caller can leave it as-is.
func resolveExpression(expr string, ctx *tryve.InterpolationContext) (string, error) {
	// 1. Built-in function: must start with $ and match $funcName or $funcName(args).
	if strings.HasPrefix(expr, "$") {
		m := builtinCallRe.FindStringSubmatch(expr)
		if m == nil {
			// Malformed builtin reference — leave as-is by returning sentinel error.
			return "", errUnresolved(expr)
		}
		name := m[1]
		var args []string
		if m[2] != "" {
			// Split comma-separated arguments and trim whitespace.
			for _, a := range strings.Split(m[2], ",") {
				args = append(args, strings.TrimSpace(a))
			}
		}
		return CallBuiltin(name, args...)
	}

	// 2. Literal "baseUrl".
	if expr == "baseUrl" {
		return ctx.BaseURL, nil
	}

	// 3. Captured values: "captured.<field>".
	if strings.HasPrefix(expr, "captured.") {
		path := strings.TrimPrefix(expr, "captured.")
		val, ok := getNestedValue(ctx.Captured, path)
		if !ok {
			return "", errUnresolved(expr)
		}
		return stringify(val), nil
	}

	// 4. Variables (dot-notation traversal).
	if ctx.Variables != nil {
		val, ok := getNestedValue(ctx.Variables, expr)
		if ok {
			return stringify(val), nil
		}
	}

	// 5. Environment map.
	if ctx.Env != nil {
		if val, ok := ctx.Env[expr]; ok {
			return val, nil
		}
	}

	// 6. Not found — signal that the expression should stay as-is.
	return "", errUnresolved(expr)
}

// errUnresolved is a sentinel error type that signals the expression was not found.
// The caller leaves the original template token intact when it encounters this error.
type unresolvedError struct{ expr string }

func (e unresolvedError) Error() string { return "unresolved: " + e.expr }

// errUnresolved constructs an unresolvedError for the given expression.
func errUnresolved(expr string) error { return unresolvedError{expr: expr} }

// isUnresolved returns true if the error is an unresolvedError.
func isUnresolved(err error) bool {
	_, ok := err.(unresolvedError)
	return ok
}

// ResolveMap interpolates all string values in a map recursively.
// Non-string values pass through unchanged.
func ResolveMap(m map[string]any, ctx *tryve.InterpolationContext) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		resolved, err := resolveValue(v, ctx)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		result[k] = resolved
	}
	return result, nil
}

// ResolveSlice interpolates all string values in a slice recursively.
// Non-string values pass through unchanged.
func ResolveSlice(s []any, ctx *tryve.InterpolationContext) ([]any, error) {
	result := make([]any, len(s))
	for i, v := range s {
		resolved, err := resolveValue(v, ctx)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		result[i] = resolved
	}
	return result, nil
}

// resolveValue resolves a single value: strings are interpolated, maps and slices
// are traversed recursively, all other types pass through unchanged.
func resolveValue(v any, ctx *tryve.InterpolationContext) (any, error) {
	switch typed := v.(type) {
	case string:
		return ResolveString(typed, ctx)
	case map[string]any:
		return ResolveMap(typed, ctx)
	case []any:
		return ResolveSlice(typed, ctx)
	default:
		return v, nil
	}
}

// ResolveVariables resolves a map of variables in topological order so that
// variables referencing other variables resolve correctly.
// Returns an error if a circular dependency is detected.
func ResolveVariables(vars map[string]any, ctx *tryve.InterpolationContext) (map[string]any, error) {
	// Build dependency graph: varName → set of variable names it depends on.
	deps := make(map[string][]string, len(vars))
	for name, val := range vars {
		strVal, ok := val.(string)
		if !ok {
			deps[name] = nil
			continue
		}
		deps[name] = findVarDeps(strVal, vars)
	}

	// Kahn's topological sort.
	order, err := topoSort(deps)
	if err != nil {
		return nil, err
	}

	// Resolve variables in dependency order, enriching ctx.Variables as we go.
	resolved := make(map[string]any, len(vars))

	// Work on a copy of ctx so we don't mutate the caller's Variables map.
	workCtx := &tryve.InterpolationContext{
		Variables: shallowCopyMap(ctx.Variables),
		Captured:  ctx.Captured,
		BaseURL:   ctx.BaseURL,
		Env:       ctx.Env,
	}

	for _, name := range order {
		val := vars[name]
		strVal, ok := val.(string)
		if !ok {
			resolved[name] = val
			workCtx.Variables[name] = val
			continue
		}
		r, err := ResolveString(strVal, workCtx)
		if err != nil {
			return nil, fmt.Errorf("variable %q: %w", name, err)
		}
		resolved[name] = r
		workCtx.Variables[name] = r
	}

	return resolved, nil
}

// findVarDeps extracts variable name references inside s that match keys in vars.
// Built-in ($...), baseUrl, and captured.* references are excluded.
func findVarDeps(s string, vars map[string]any) []string {
	seen := make(map[string]struct{})
	var deps []string

	addIfVar := func(expr string) {
		expr = strings.TrimSpace(expr)
		// Skip built-ins, baseUrl, and captured.* — these are not variable deps.
		if strings.HasPrefix(expr, "$") || expr == "baseUrl" || strings.HasPrefix(expr, "captured.") {
			return
		}
		// Top-level key in vars (only the first segment before a dot matters).
		root := strings.SplitN(expr, ".", 2)[0]
		if _, isVar := vars[root]; isVar {
			if _, already := seen[root]; !already {
				seen[root] = struct{}{}
				deps = append(deps, root)
			}
		}
	}

	// Only scan {{expr}} patterns for variable cross-references.
	// ${expr} patterns are environment variable references resolved by the config
	// loader and must not be treated as variable dependencies.
	doubleBraceRe.ReplaceAllStringFunc(s, func(match string) string {
		m := doubleBraceRe.FindStringSubmatch(match)
		if len(m) >= 2 {
			addIfVar(m[1])
		}
		return match
	})

	return deps
}

// topoSort performs Kahn's algorithm on the dependency graph.
// deps[node] = list of nodes that node depends on (prerequisites).
// Returns a slice of keys in dependency-first order, or an error on cycle detection.
func topoSort(deps map[string][]string) ([]string, error) {
	// Build adjacency list: dep → [nodes that depend on dep].
	// Also compute in-degree per node (number of prerequisites).
	adj := make(map[string][]string, len(deps))
	nodeInDegree := make(map[string]int, len(deps))
	for node := range deps {
		nodeInDegree[node] = 0 // ensure every node appears
	}
	for node, nodeDeps := range deps {
		for _, dep := range nodeDeps {
			adj[dep] = append(adj[dep], node)
			nodeInDegree[node]++
		}
	}

	// Queue nodes with zero in-degree (no prerequisites).
	var queue []string
	for node, deg := range nodeInDegree {
		if deg == 0 {
			queue = append(queue, node)
		}
	}

	var order []string
	for len(queue) > 0 {
		// Pop from queue (order within same level doesn't matter for correctness).
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, dependent := range adj[node] {
			nodeInDegree[dependent]--
			if nodeInDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(order) != len(deps) {
		return nil, fmt.Errorf("circular dependency detected in variables")
	}

	return order, nil
}

// getNestedValue traverses map m using dot-separated path segments.
// Returns the value and true if found, or zero and false otherwise.
func getNestedValue(m map[string]any, path string) (any, bool) {
	if m == nil {
		return nil, false
	}
	parts := strings.SplitN(path, ".", 2)
	val, ok := m[parts[0]]
	if !ok {
		return nil, false
	}
	if len(parts) == 1 {
		return val, true
	}
	// Recurse into nested map.
	nested, ok := val.(map[string]any)
	if !ok {
		return nil, false
	}
	return getNestedValue(nested, parts[1])
}

// stringify converts a value to string, handling nil as empty string.
func stringify(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// shallowCopyMap returns a shallow copy of a map[string]any.
func shallowCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
