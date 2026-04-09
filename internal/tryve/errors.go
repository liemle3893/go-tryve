package tryve

import (
	"fmt"
	"time"
)

// TryveError is the structured error type used throughout the tryve package.
// It carries a machine-readable Code, a human-readable Message, an optional Hint
// for remediation, and an optional wrapped Cause for error chain traversal.
type TryveError struct {
	Code    string
	Message string
	Hint    string
	Cause   error
}

// Error returns the string representation of the error.
// When a cause is present it formats as "message: cause", otherwise just "message".
func (e *TryveError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the wrapped cause to support errors.Is and errors.As traversal.
func (e *TryveError) Unwrap() error { return e.Cause }

// ConfigError constructs a TryveError for configuration-related failures.
func ConfigError(msg, hint string, cause error) *TryveError {
	return &TryveError{
		Code:    "CONFIG_ERROR",
		Message: msg,
		Hint:    hint,
		Cause:   cause,
	}
}

// ValidationError constructs a TryveError for input or schema validation failures.
func ValidationError(msg, hint string, cause error) *TryveError {
	return &TryveError{
		Code:    "VALIDATION_ERROR",
		Message: msg,
		Hint:    hint,
		Cause:   cause,
	}
}

// ConnectionError constructs a TryveError for adapter connectivity failures.
// The hint directs the user to check the named adapter's settings in e2e.config.yaml.
func ConnectionError(adapter, msg string, cause error) *TryveError {
	return &TryveError{
		Code:    "CONNECTION_ERROR",
		Message: msg,
		Hint:    fmt.Sprintf("check %s connection settings in e2e.config.yaml", adapter),
		Cause:   cause,
	}
}

// ExecutionError constructs a TryveError for step execution failures.
// The hint directs the user to check the named step's configuration.
func ExecutionError(step, msg string, cause error) *TryveError {
	return &TryveError{
		Code:    "EXECUTION_ERROR",
		Message: msg,
		Hint:    fmt.Sprintf("check step %s configuration", step),
		Cause:   cause,
	}
}

// AssertionError constructs a TryveError for assertion check failures.
// The message encodes the path, operator, expected, and actual values.
func AssertionError(path, operator string, expected, actual any) *TryveError {
	return &TryveError{
		Code:    "ASSERTION_ERROR",
		Message: fmt.Sprintf("assertion failed: %s %s %v, got %v", path, operator, expected, actual),
	}
}

// TimeoutError constructs a TryveError for operations that exceeded their deadline.
func TimeoutError(operation string, duration time.Duration) *TryveError {
	return &TryveError{
		Code:    "TIMEOUT_ERROR",
		Message: fmt.Sprintf("%s timed out after %s", operation, duration),
		Hint:    "increase timeout in config or step definition",
	}
}

// InterpolationError constructs a TryveError for template/variable interpolation failures.
func InterpolationError(expr, msg string) *TryveError {
	return &TryveError{
		Code:    "INTERPOLATION_ERROR",
		Message: fmt.Sprintf("interpolation error for %q: %s", expr, msg),
	}
}

// AdapterError constructs a TryveError for generic adapter action failures.
// The message is formatted as "{adapter}.{action}: {msg}".
func AdapterError(adapter, action, msg string, cause error) *TryveError {
	return &TryveError{
		Code:    "ADAPTER_ERROR",
		Message: fmt.Sprintf("%s.%s: %s", adapter, action, msg),
		Cause:   cause,
	}
}
