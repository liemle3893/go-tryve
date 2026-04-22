package adapter

import (
	"context"
	"time"

	"github.com/liemle3893/autoflow/internal/core"
)

// Adapter is the core interface all protocol adapters must implement.
// Each method must be safe to call concurrently from a single goroutine;
// the Registry serialises Connect/Close calls via its own mutex.
type Adapter interface {
	// Name returns the adapter's registered identifier (e.g. "http", "db").
	Name() string

	// Connect establishes the underlying connection or session.
	// It is called at most once per adapter instance by the Registry.
	Connect(ctx context.Context) error

	// Close tears down the adapter's connection or session.
	// It is called by Registry.CloseAll for every connected adapter.
	Close(ctx context.Context) error

	// Health performs a lightweight connectivity check without executing any
	// test logic. Returns nil when the adapter is reachable.
	Health(ctx context.Context) error

	// Execute runs the named action with the provided parameters and returns
	// a StepResult containing the adapter's output data and timing.
	Execute(ctx context.Context, action string, params map[string]any) (*core.StepResult, error)
}

// MeasureDuration executes fn, measures its wall-clock duration, and returns
// that duration alongside any error fn produced.
func MeasureDuration(fn func() error) (time.Duration, error) {
	start := time.Now()
	err := fn()
	return time.Since(start), err
}

// SuccessResult constructs a StepResult for a successful adapter action.
// data holds the primary output, duration records execution time, and
// metadata carries optional diagnostic key-value pairs.
func SuccessResult(data map[string]any, duration time.Duration, metadata map[string]any) *core.StepResult {
	return &core.StepResult{
		Data:     data,
		Duration: duration,
		Metadata: metadata,
	}
}
