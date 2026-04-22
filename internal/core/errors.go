package core

import (
	"fmt"
	"time"
)

// Error is the structured error type used throughout the autoflow package.
// It carries a machine-readable Code, a human-readable Message, an optional Hint
// for remediation, and an optional wrapped Cause for error chain traversal.
type Error struct {
	Code    string
	Message string
	Hint    string
	Cause   error
}

// Error returns the string representation of the error.
// When a cause is present it formats as "message: cause", otherwise just "message".
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the wrapped cause to support errors.Is and errors.As traversal.
func (e *Error) Unwrap() error { return e.Cause }

// ConfigError constructs a Error for configuration-related failures.
func ConfigError(msg, hint string, cause error) *Error {
	return &Error{
		Code:    "CONFIG_ERROR",
		Message: msg,
		Hint:    hint,
		Cause:   cause,
	}
}

// ValidationError constructs a Error for input or schema validation failures.
func ValidationError(msg, hint string, cause error) *Error {
	return &Error{
		Code:    "VALIDATION_ERROR",
		Message: msg,
		Hint:    hint,
		Cause:   cause,
	}
}

// ConnectionError constructs a Error for adapter connectivity failures.
// The hint directs the user to check the named adapter's settings in e2e.config.yaml.
func ConnectionError(adapter, msg string, cause error) *Error {
	return &Error{
		Code:    "CONNECTION_ERROR",
		Message: msg,
		Hint:    fmt.Sprintf("check %s connection settings in e2e.config.yaml", adapter),
		Cause:   cause,
	}
}

// ExecutionError constructs a Error for step execution failures.
// The hint directs the user to check the named step's configuration.
func ExecutionError(step, msg string, cause error) *Error {
	return &Error{
		Code:    "EXECUTION_ERROR",
		Message: msg,
		Hint:    fmt.Sprintf("check step %s configuration", step),
		Cause:   cause,
	}
}

// AssertionError constructs a Error for assertion check failures.
// The message encodes the path, operator, expected, and actual values.
func AssertionError(path, operator string, expected, actual any) *Error {
	return &Error{
		Code:    "ASSERTION_ERROR",
		Message: fmt.Sprintf("assertion failed: %s %s %v, got %v", path, operator, expected, actual),
	}
}

// TimeoutError constructs a Error for operations that exceeded their deadline.
func TimeoutError(operation string, duration time.Duration) *Error {
	return &Error{
		Code:    "TIMEOUT_ERROR",
		Message: fmt.Sprintf("%s timed out after %s", operation, duration),
		Hint:    "increase timeout in config or step definition",
	}
}

// InterpolationError constructs a Error for template/variable interpolation failures.
func InterpolationError(expr, msg string) *Error {
	return &Error{
		Code:    "INTERPOLATION_ERROR",
		Message: fmt.Sprintf("interpolation error for %q: %s", expr, msg),
	}
}

// AdapterError constructs a Error for generic adapter action failures.
// The message is formatted as "{adapter}.{action}: {msg}".
func AdapterError(adapter, action, msg string, cause error) *Error {
	return &Error{
		Code:    "ADAPTER_ERROR",
		Message: fmt.Sprintf("%s.%s: %s", adapter, action, msg),
		Cause:   cause,
	}
}
