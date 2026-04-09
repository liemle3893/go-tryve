package assertion

import (
	"fmt"
	"strings"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// knownHTTPKeys lists top-level keys that receive special handling in map-format assertions.
// Any other key at the top level is treated as a direct operator.
var knownHTTPKeys = map[string]bool{
	"status":      true,
	"statusRange": true,
	"headers":     true,
	"json":        true,
	"body":        true,
	"duration":    true,
}

// operatorNames is the set of all recognised operator names.
// Used to detect direct operator keys at the top level of a map assertion.
var operatorNames = map[string]bool{
	"equals":             true,
	"notEquals":          true,
	"contains":           true,
	"notContains":        true,
	"matches":            true,
	"type":               true,
	"exists":             true,
	"notExists":          true,
	"isNull":             true,
	"isNotNull":          true,
	"greaterThan":        true,
	"lessThan":           true,
	"greaterThanOrEqual": true,
	"lessThanOrEqual":    true,
	"length":             true,
	"isEmpty":            true,
	"notEmpty":           true,
	"hasProperty":        true,
	"notHasProperty":     true,
}

// RunAssertions evaluates assertDef against data and returns one AssertionOutcome per check.
//
// assertDef may be:
//   - nil — returns empty outcomes
//   - map[string]any — HTTP-style with keys: status, statusRange, headers, json, body, duration,
//     or a direct {path, operator: value} block
//   - []any — generic slice of {path, operator: value} items
func RunAssertions(data map[string]any, assertDef any) ([]tryve.AssertionOutcome, error) {
	if assertDef == nil {
		return nil, nil
	}

	switch def := assertDef.(type) {
	case map[string]any:
		return runMapAssertions(data, def)
	case []any:
		return runSliceAssertions(data, def)
	default:
		return nil, fmt.Errorf("unsupported assertDef type %T", assertDef)
	}
}

// runMapAssertions handles the HTTP-style map format.
func runMapAssertions(data map[string]any, def map[string]any) ([]tryve.AssertionOutcome, error) {
	var outcomes []tryve.AssertionOutcome

	// status — single number or []any oneOf check.
	if statusDef, ok := def["status"]; ok {
		actual := data["status"]
		switch sv := statusDef.(type) {
		case []any:
			// oneOf check — actual must equal one of the values in the array.
			o := assertOneOf("status", actual, sv)
			outcomes = append(outcomes, o)
		default:
			// Single-value equals check.
			r := Match("equals", actual, statusDef)
			outcomes = append(outcomes, tryve.AssertionOutcome{
				Path:     "status",
				Operator: "equals",
				Expected: statusDef,
				Actual:   actual,
				Passed:   r.Pass,
				Message:  r.Message,
			})
		}
	}

	// statusRange — [min, max] inclusive.
	if rangeDef, ok := def["statusRange"]; ok {
		outcomes = append(outcomes, assertStatusRange(data["status"], rangeDef))
	}

	// headers — map of name→expected value with case-insensitive lookup.
	if headersDef, ok := def["headers"]; ok {
		if hm, ok := headersDef.(map[string]any); ok {
			actualHeaders, _ := data["headers"].(map[string]any)
			for wantName, wantVal := range hm {
				actual := headerLookup(actualHeaders, wantName)
				r := Match("equals", actual, wantVal)
				outcomes = append(outcomes, tryve.AssertionOutcome{
					Path:     "headers." + wantName,
					Operator: "equals",
					Expected: wantVal,
					Actual:   actual,
					Passed:   r.Pass,
					Message:  r.Message,
				})
			}
		}
	}

	// json — []any of {path, operator: value} items.
	if jsonDef, ok := def["json"]; ok {
		if items, ok := jsonDef.([]any); ok {
			outs, err := runSliceAssertions(data, items)
			if err != nil {
				return outcomes, err
			}
			outcomes = append(outcomes, outs...)
		}
	}

	// body — map of {contains/matches/equals: value}.
	if bodyDef, ok := def["body"]; ok {
		if bm, ok := bodyDef.(map[string]any); ok {
			bodyStr := fmt.Sprintf("%v", data["body"])
			for op, val := range bm {
				r := Match(op, bodyStr, val)
				outcomes = append(outcomes, tryve.AssertionOutcome{
					Path:     "body",
					Operator: op,
					Expected: val,
					Actual:   bodyStr,
					Passed:   r.Pass,
					Message:  r.Message,
				})
			}
		}
	}

	// duration — map of {lessThan/greaterThan: value}.
	if durationDef, ok := def["duration"]; ok {
		if dm, ok := durationDef.(map[string]any); ok {
			actual := data["duration"]
			for op, val := range dm {
				r := Match(op, actual, val)
				outcomes = append(outcomes, tryve.AssertionOutcome{
					Path:     "duration",
					Operator: op,
					Expected: val,
					Actual:   actual,
					Passed:   r.Pass,
					Message:  r.Message,
				})
			}
		}
	}

	// Direct operator format — top-level map has a "path" key and one or more operator keys.
	if path, hasPath := def["path"]; hasPath {
		pathStr, _ := path.(string)
		actual, _ := EvalJSONPath(data, pathStr)
		for key, val := range def {
			if key == "path" {
				continue
			}
			if !operatorNames[key] {
				continue
			}
			r := Match(key, actual, val)
			outcomes = append(outcomes, tryve.AssertionOutcome{
				Path:     pathStr,
				Operator: key,
				Expected: val,
				Actual:   actual,
				Passed:   r.Pass,
				Message:  r.Message,
			})
		}
	} else {
		// Check for top-level operator keys that are not known HTTP keys and not "path".
		// This handles adapters that place operator checks directly at the map root
		// alongside non-HTTP-specific keys (rare, but supported for completeness).
		for key, val := range def {
			if knownHTTPKeys[key] {
				continue
			}
			if operatorNames[key] {
				r := Match(key, data, val)
				outcomes = append(outcomes, tryve.AssertionOutcome{
					Path:     "$",
					Operator: key,
					Expected: val,
					Actual:   data,
					Passed:   r.Pass,
					Message:  r.Message,
				})
			}
		}
	}

	return outcomes, nil
}

// runSliceAssertions handles the generic []any format where each item is a
// map[string]any with a "path" key and one or more operator keys.
func runSliceAssertions(data map[string]any, items []any) ([]tryve.AssertionOutcome, error) {
	var outcomes []tryve.AssertionOutcome
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pathVal, _ := m["path"]
		pathStr, _ := pathVal.(string)
		actual, _ := EvalJSONPath(data, pathStr)

		for key, val := range m {
			if key == "path" {
				continue
			}
			if !operatorNames[key] {
				continue
			}
			r := Match(key, actual, val)
			outcomes = append(outcomes, tryve.AssertionOutcome{
				Path:     pathStr,
				Operator: key,
				Expected: val,
				Actual:   actual,
				Passed:   r.Pass,
				Message:  r.Message,
			})
		}
	}
	return outcomes, nil
}

// assertOneOf checks that actual equals one value in the allowed slice.
func assertOneOf(path string, actual any, allowed []any) tryve.AssertionOutcome {
	normalActual := normalizeNumeric(actual)
	for _, v := range allowed {
		if fmt.Sprintf("%v", normalizeNumeric(v)) == fmt.Sprintf("%v", normalActual) {
			return tryve.AssertionOutcome{
				Path:     path,
				Operator: "oneOf",
				Expected: allowed,
				Actual:   actual,
				Passed:   true,
			}
		}
	}
	return tryve.AssertionOutcome{
		Path:     path,
		Operator: "oneOf",
		Expected: allowed,
		Actual:   actual,
		Passed:   false,
		Message:  fmt.Sprintf("expected %v to be one of %v", actual, allowed),
	}
}

// assertStatusRange checks that actual status is within [min, max] inclusive.
func assertStatusRange(actual any, rangeDef any) tryve.AssertionOutcome {
	arr, ok := rangeDef.([]any)
	if !ok || len(arr) < 2 {
		return tryve.AssertionOutcome{
			Path:     "statusRange",
			Operator: "statusRange",
			Expected: rangeDef,
			Actual:   actual,
			Passed:   false,
			Message:  "statusRange must be an array with [min, max]",
		}
	}
	min := toFloat64(arr[0])
	max := toFloat64(arr[1])
	val := toFloat64(actual)
	if val >= min && val <= max {
		return tryve.AssertionOutcome{
			Path:     "statusRange",
			Operator: "statusRange",
			Expected: rangeDef,
			Actual:   actual,
			Passed:   true,
		}
	}
	return tryve.AssertionOutcome{
		Path:     "statusRange",
		Operator: "statusRange",
		Expected: rangeDef,
		Actual:   actual,
		Passed:   false,
		Message:  fmt.Sprintf("status %v is not in range [%v, %v]", actual, arr[0], arr[1]),
	}
}

// headerLookup performs a case-insensitive key lookup in a headers map.
// Returns nil when the header is not present.
func headerLookup(headers map[string]any, name string) any {
	if headers == nil {
		return nil
	}
	lower := strings.ToLower(name)
	for k, v := range headers {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return nil
}
