package assertion_test

import (
	"testing"

	"github.com/liemle3893/autoflow/internal/assertion"
)

// TestMatch_Equals verifies deep-equal comparisons for int, string, bool, and nil values,
// including numeric normalisation across int/float64.
func TestMatch_Equals(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		expected any
		wantPass bool
	}{
		{"int equals int", 42, 42, true},
		{"int equals float64 (normalised)", 42, float64(42), true},
		{"int not equal to different int", 42, 43, false},
		{"string equals string", "hello", "hello", true},
		{"string not equal to different string", "hello", "world", false},
		{"bool true equals true", true, true, true},
		{"bool false equals false", false, false, true},
		{"bool mismatch", true, false, false},
		{"nil equals nil", nil, nil, true},
		{"nil not equal to int", nil, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("equals", tc.actual, tc.expected)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(equals, %v, %v).Pass = %v, want %v — %s",
					tc.actual, tc.expected, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_NotEquals verifies that notEquals is the inverse of equals.
func TestMatch_NotEquals(t *testing.T) {
	if !assertion.Match("notEquals", 1, 2).Pass {
		t.Error("notEquals(1,2) should pass")
	}
	if assertion.Match("notEquals", 1, 1).Pass {
		t.Error("notEquals(1,1) should fail")
	}
}

// TestMatch_Contains verifies substring containment and array element containment.
func TestMatch_Contains(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		expected any
		wantPass bool
	}{
		{"string contains substring", "hello world", "world", true},
		{"string does not contain substring", "hello world", "foo", false},
		{"array contains element", []any{"a", "b", "c"}, "b", true},
		{"array does not contain element", []any{"a", "b", "c"}, "d", false},
		{"array contains int", []any{1, 2, 3}, 2, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("contains", tc.actual, tc.expected)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(contains, %v, %v).Pass = %v, want %v — %s",
					tc.actual, tc.expected, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_NotContains verifies that notContains is the inverse of contains.
func TestMatch_NotContains(t *testing.T) {
	if !assertion.Match("notContains", "hello", "world").Pass {
		t.Error("notContains('hello','world') should pass")
	}
	if assertion.Match("notContains", "hello world", "world").Pass {
		t.Error("notContains('hello world','world') should fail")
	}
}

// TestMatch_Matches verifies regex matching.
func TestMatch_Matches(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		expected any
		wantPass bool
	}{
		{"matches simple pattern", "hello123", `\d+`, true},
		{"does not match pattern", "hello", `^\d+$`, false},
		{"matches email pattern", "test@example.com", `^[^@]+@[^@]+\.[^@]+$`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("matches", tc.actual, tc.expected)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(matches, %v, %v).Pass = %v, want %v — %s",
					tc.actual, tc.expected, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_Type verifies the type operator for all supported type names.
func TestMatch_Type(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		typeName string
		wantPass bool
	}{
		{"string type", "hello", "string", true},
		{"int as number", 42, "number", true},
		{"int64 as number", int64(42), "number", true},
		{"float64 as number", float64(3.14), "number", true},
		{"float32 as number", float32(1.0), "number", true},
		{"bool type", true, "boolean", true},
		{"nil as null", nil, "null", true},
		{"map as object", map[string]any{"k": "v"}, "object", true},
		{"slice as array", []any{1, 2, 3}, "array", true},
		{"string not boolean", "hello", "boolean", false},
		{"int not string", 42, "string", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("type", tc.actual, tc.typeName)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(type, %v, %v).Pass = %v, want %v — %s",
					tc.actual, tc.typeName, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_GreaterThan verifies numeric greater-than comparisons.
func TestMatch_GreaterThan(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		expected any
		wantPass bool
	}{
		{"int greater than int", 10, 5, true},
		{"int not greater than equal", 5, 5, false},
		{"int not greater than larger", 3, 10, false},
		{"float greater than int", float64(10.5), 10, true},
		{"string numeric greater than", "20", 15, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("greaterThan", tc.actual, tc.expected)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(greaterThan, %v, %v).Pass = %v, want %v — %s",
					tc.actual, tc.expected, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_LessThan verifies numeric less-than comparisons.
func TestMatch_LessThan(t *testing.T) {
	if !assertion.Match("lessThan", 3, 10).Pass {
		t.Error("lessThan(3,10) should pass")
	}
	if assertion.Match("lessThan", 10, 3).Pass {
		t.Error("lessThan(10,3) should fail")
	}
	if assertion.Match("lessThan", 5, 5).Pass {
		t.Error("lessThan(5,5) should fail")
	}
}

// TestMatch_GreaterThanOrEqual verifies numeric >= comparisons.
func TestMatch_GreaterThanOrEqual(t *testing.T) {
	if !assertion.Match("greaterThanOrEqual", 5, 5).Pass {
		t.Error("greaterThanOrEqual(5,5) should pass")
	}
	if !assertion.Match("greaterThanOrEqual", 10, 5).Pass {
		t.Error("greaterThanOrEqual(10,5) should pass")
	}
	if assertion.Match("greaterThanOrEqual", 3, 5).Pass {
		t.Error("greaterThanOrEqual(3,5) should fail")
	}
}

// TestMatch_LessThanOrEqual verifies numeric <= comparisons.
func TestMatch_LessThanOrEqual(t *testing.T) {
	if !assertion.Match("lessThanOrEqual", 5, 5).Pass {
		t.Error("lessThanOrEqual(5,5) should pass")
	}
	if !assertion.Match("lessThanOrEqual", 3, 5).Pass {
		t.Error("lessThanOrEqual(3,5) should pass")
	}
	if assertion.Match("lessThanOrEqual", 10, 5).Pass {
		t.Error("lessThanOrEqual(10,5) should fail")
	}
}

// TestMatch_Length verifies length checks for arrays and strings.
func TestMatch_Length(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		expected any
		wantPass bool
	}{
		{"array length match", []any{1, 2, 3}, 3, true},
		{"array length mismatch", []any{1, 2, 3}, 2, false},
		{"string length match", "hello", 5, true},
		{"string length mismatch", "hello", 4, false},
		{"map length match", map[string]any{"a": 1, "b": 2}, 2, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("length", tc.actual, tc.expected)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(length, %v, %v).Pass = %v, want %v — %s",
					tc.actual, tc.expected, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_Exists verifies that exists checks non-nil presence.
func TestMatch_Exists(t *testing.T) {
	if !assertion.Match("exists", "value", true).Pass {
		t.Error("exists('value', true) should pass")
	}
	if assertion.Match("exists", nil, true).Pass {
		t.Error("exists(nil, true) should fail")
	}
	if !assertion.Match("exists", nil, false).Pass {
		t.Error("exists(nil, false) should pass (expect non-existence)")
	}
}

// TestMatch_NotExists verifies that notExists is the inverse of exists.
func TestMatch_NotExists(t *testing.T) {
	if !assertion.Match("notExists", nil, nil).Pass {
		t.Error("notExists(nil) should pass")
	}
	if assertion.Match("notExists", "value", nil).Pass {
		t.Error("notExists('value') should fail")
	}
}

// TestMatch_IsNull verifies nil-value checks.
func TestMatch_IsNull(t *testing.T) {
	if !assertion.Match("isNull", nil, nil).Pass {
		t.Error("isNull(nil) should pass")
	}
	if assertion.Match("isNull", "value", nil).Pass {
		t.Error("isNull('value') should fail")
	}
}

// TestMatch_IsNotNull verifies non-nil-value checks.
func TestMatch_IsNotNull(t *testing.T) {
	if !assertion.Match("isNotNull", "value", nil).Pass {
		t.Error("isNotNull('value') should pass")
	}
	if assertion.Match("isNotNull", nil, nil).Pass {
		t.Error("isNotNull(nil) should fail")
	}
}

// TestMatch_IsEmpty verifies emptiness checks for arrays and strings.
func TestMatch_IsEmpty(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		wantPass bool
	}{
		{"empty array", []any{}, true},
		{"non-empty array", []any{1}, false},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"nil is empty", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := assertion.Match("isEmpty", tc.actual, nil)
			if result.Pass != tc.wantPass {
				t.Errorf("Match(isEmpty, %v, nil).Pass = %v, want %v — %s",
					tc.actual, result.Pass, tc.wantPass, result.Message)
			}
		})
	}
}

// TestMatch_NotEmpty verifies that notEmpty is the inverse of isEmpty.
func TestMatch_NotEmpty(t *testing.T) {
	if !assertion.Match("notEmpty", []any{1, 2}, nil).Pass {
		t.Error("notEmpty([1,2]) should pass")
	}
	if assertion.Match("notEmpty", []any{}, nil).Pass {
		t.Error("notEmpty([]) should fail")
	}
	if assertion.Match("notEmpty", "", nil).Pass {
		t.Error("notEmpty('') should fail")
	}
}

// TestMatch_HasProperty verifies map key existence checks.
func TestMatch_HasProperty(t *testing.T) {
	obj := map[string]any{"name": "Alice", "age": 30, "active": nil}

	if !assertion.Match("hasProperty", obj, "name").Pass {
		t.Error("hasProperty(obj, 'name') should pass")
	}
	if !assertion.Match("hasProperty", obj, "active").Pass {
		t.Error("hasProperty(obj, 'active') should pass even when value is nil")
	}
	if assertion.Match("hasProperty", obj, "missing").Pass {
		t.Error("hasProperty(obj, 'missing') should fail")
	}
}

// TestMatch_NotHasProperty verifies that notHasProperty is the inverse of hasProperty.
func TestMatch_NotHasProperty(t *testing.T) {
	obj := map[string]any{"a": 1}
	if !assertion.Match("notHasProperty", obj, "b").Pass {
		t.Error("notHasProperty(obj, 'b') should pass")
	}
	if assertion.Match("notHasProperty", obj, "a").Pass {
		t.Error("notHasProperty(obj, 'a') should fail")
	}
}

// TestMatch_UnknownOperator verifies that an unknown operator returns a failing result with a clear message.
func TestMatch_UnknownOperator(t *testing.T) {
	result := assertion.Match("unknownOp", "value", "value")
	if result.Pass {
		t.Error("unknown operator should not pass")
	}
	if result.Message == "" {
		t.Error("unknown operator should return a non-empty message")
	}
}
