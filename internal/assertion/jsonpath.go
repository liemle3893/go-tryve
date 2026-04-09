package assertion

import (
	"strings"

	"github.com/ohler55/ojg/jp"
)

// normalizePath ensures a JSONPath expression starts with "$".
// Bare property names like "status" are converted to "$.status".
func normalizePath(path string) string {
	if !strings.HasPrefix(path, "$") {
		return "$." + path
	}
	return path
}

// EvalJSONPath evaluates a JSONPath expression against data.
// Returns the matched value and true when found.
// When the expression matches exactly one result the value is returned directly;
// when it matches multiple results a []any slice is returned.
// Returns (nil, false) when no results are found.
func EvalJSONPath(data any, path string) (any, bool) {
	results := QueryJSONPath(data, path)
	if len(results) == 0 {
		return nil, false
	}
	if len(results) == 1 {
		return results[0], true
	}
	return results, true
}

// HasJSONPath checks whether a path exists in the data.
// A key whose value is nil is still considered present — this uses jp.Expr.Has
// which reports presence regardless of the value at that path.
func HasJSONPath(data any, path string) bool {
	path = normalizePath(path)
	expr, err := jp.ParseString(path)
	if err != nil {
		return dotTraverseExists(data, path)
	}
	return expr.Has(data)
}

// QueryJSONPath returns all matches for a path expression.
// It always returns a (possibly empty) slice; the caller decides how to interpret
// single vs. multiple results.
func QueryJSONPath(data any, path string) []any {
	path = normalizePath(path)
	expr, err := jp.ParseString(path)
	if err != nil {
		// Fallback: attempt simple dot-notation traversal.
		v, ok := dotTraverse(data, path)
		if !ok {
			return nil
		}
		return []any{v}
	}
	return expr.Get(data)
}

// dotTraverse performs a simple dot-notation traversal as a fallback when
// the ojg parser cannot handle the provided expression.
// It strips a leading "$." and splits on "." to walk map keys.
func dotTraverse(data any, path string) (any, bool) {
	// Strip leading "$." or "$"
	clean := strings.TrimPrefix(path, "$.")
	clean = strings.TrimPrefix(clean, "$")
	if clean == "" {
		return data, true
	}

	parts := strings.Split(clean, ".")
	cur := data
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		cur = val
	}
	return cur, true
}

// dotTraverseExists is the HasJSONPath equivalent of dotTraverse.
// It reports whether the final key in the dot-notation path is present,
// regardless of whether its value is nil.
func dotTraverseExists(data any, path string) bool {
	clean := strings.TrimPrefix(path, "$.")
	clean = strings.TrimPrefix(clean, "$")
	if clean == "" {
		return true
	}

	parts := strings.Split(clean, ".")
	cur := data
	for i, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		val, exists := m[part]
		if !exists {
			return false
		}
		if i == len(parts)-1 {
			// Last segment — key is present regardless of value.
			_ = val
			return true
		}
		cur = val
	}
	return true
}
