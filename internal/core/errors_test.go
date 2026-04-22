package core_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/liemle3893/autoflow/internal/core"
)

// TestError_Error_MessageOnly verifies Error() returns only the message when no cause is set.
func TestError_Error_MessageOnly(t *testing.T) {
	err := &core.Error{
		Code:    "TEST_CODE",
		Message: "something went wrong",
	}
	got := err.Error()
	want := "something went wrong"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestError_Error_WithCause verifies Error() returns "message: cause" when a cause is present.
func TestError_Error_WithCause(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := &core.Error{
		Code:    "TEST_CODE",
		Message: "something went wrong",
		Cause:   cause,
	}
	got := err.Error()
	want := "something went wrong: root cause"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestError_ErrorsIs verifies errors.Is() can find the wrapped cause.
func TestError_ErrorsIs(t *testing.T) {
	sentinel := fmt.Errorf("sentinel error")
	err := &core.Error{
		Code:    "TEST_CODE",
		Message: "wrapper",
		Cause:   sentinel,
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is() should find wrapped cause via Unwrap()")
	}
}

// TestError_ErrorsAs verifies errors.As() can match a Error type.
func TestError_ErrorsAs(t *testing.T) {
	inner := &core.Error{
		Code:    "INNER_CODE",
		Message: "inner error",
	}
	outer := fmt.Errorf("outer: %w", inner)

	var target *core.Error
	if !errors.As(outer, &target) {
		t.Errorf("errors.As() should match Error type")
	}
	if target.Code != "INNER_CODE" {
		t.Errorf("errors.As() target Code = %q, want %q", target.Code, "INNER_CODE")
	}
}

// TestConfigError verifies ConfigError sets the correct code and fields.
func TestConfigError(t *testing.T) {
	cause := fmt.Errorf("file not found")
	err := core.ConfigError("bad config", "check path", cause)
	if err.Code != "CONFIG_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "CONFIG_ERROR")
	}
	if err.Message != "bad config" {
		t.Errorf("Message = %q, want %q", err.Message, "bad config")
	}
	if err.Hint != "check path" {
		t.Errorf("Hint = %q, want %q", err.Hint, "check path")
	}
	if !errors.Is(err, cause) {
		t.Errorf("ConfigError should wrap cause")
	}
}

// TestValidationError verifies ValidationError sets the correct code and fields.
func TestValidationError(t *testing.T) {
	err := core.ValidationError("invalid field", "use correct type", nil)
	if err.Code != "VALIDATION_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "VALIDATION_ERROR")
	}
	if err.Message != "invalid field" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid field")
	}
	if err.Hint != "use correct type" {
		t.Errorf("Hint = %q, want %q", err.Hint, "use correct type")
	}
}

// TestConnectionError verifies ConnectionError sets the correct code, message, and hint format.
func TestConnectionError(t *testing.T) {
	cause := fmt.Errorf("dial tcp refused")
	err := core.ConnectionError("http", "connection refused", cause)
	if err.Code != "CONNECTION_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "CONNECTION_ERROR")
	}
	wantHint := "check http connection settings in e2e.config.yaml"
	if err.Hint != wantHint {
		t.Errorf("Hint = %q, want %q", err.Hint, wantHint)
	}
	if !errors.Is(err, cause) {
		t.Errorf("ConnectionError should wrap cause")
	}
}

// TestExecutionError verifies ExecutionError sets the correct code and hint format.
func TestExecutionError(t *testing.T) {
	err := core.ExecutionError("step-1", "action failed", nil)
	if err.Code != "EXECUTION_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "EXECUTION_ERROR")
	}
	wantHint := "check step step-1 configuration"
	if err.Hint != wantHint {
		t.Errorf("Hint = %q, want %q", err.Hint, wantHint)
	}
}

// TestAssertionError verifies AssertionError sets the correct code and constructs the message.
func TestAssertionError(t *testing.T) {
	err := core.AssertionError("$.status", "equals", 200, 404)
	if err.Code != "ASSERTION_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "ASSERTION_ERROR")
	}
	wantMsg := "assertion failed: $.status equals 200, got 404"
	if err.Message != wantMsg {
		t.Errorf("Message = %q, want %q", err.Message, wantMsg)
	}
	if err.Cause != nil {
		t.Errorf("AssertionError should have nil cause, got %v", err.Cause)
	}
}

// TestTimeoutError verifies TimeoutError sets the correct code, message, and hint.
func TestTimeoutError(t *testing.T) {
	err := core.TimeoutError("HTTP request", 30*time.Second)
	if err.Code != "TIMEOUT_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "TIMEOUT_ERROR")
	}
	wantHint := "increase timeout in config or step definition"
	if err.Hint != wantHint {
		t.Errorf("Hint = %q, want %q", err.Hint, wantHint)
	}
	if err.Message == "" {
		t.Errorf("TimeoutError message should not be empty")
	}
}

// TestInterpolationError verifies InterpolationError sets the correct code and message.
func TestInterpolationError(t *testing.T) {
	err := core.InterpolationError("${unknown}", "variable not found")
	if err.Code != "INTERPOLATION_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "INTERPOLATION_ERROR")
	}
	if err.Message == "" {
		t.Errorf("InterpolationError message should not be empty")
	}
}

// TestAdapterError verifies AdapterError sets the correct code and constructs the message.
func TestAdapterError(t *testing.T) {
	cause := fmt.Errorf("network error")
	err := core.AdapterError("http", "GET", "request failed", cause)
	if err.Code != "ADAPTER_ERROR" {
		t.Errorf("Code = %q, want %q", err.Code, "ADAPTER_ERROR")
	}
	wantMsg := "http.GET: request failed"
	if err.Message != wantMsg {
		t.Errorf("Message = %q, want %q", err.Message, wantMsg)
	}
	if !errors.Is(err, cause) {
		t.Errorf("AdapterError should wrap cause")
	}
}
