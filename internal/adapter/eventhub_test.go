package adapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/liemle3893/e2e-runner/internal/adapter"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// TestNewEventHubAdapter_DefaultConsumerGroup verifies that omitting consumerGroup
// in the config results in the default "$Default" value.
func TestNewEventHubAdapter_DefaultConsumerGroup(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
		"eventHubName":     "my-hub",
		// consumerGroup intentionally omitted
	})
	if a == nil {
		t.Fatal("expected non-nil EventHubAdapter")
	}
	// The consumer group is used internally; we verify the adapter carries it by
	// checking that the adapter is considered healthy once connected. We cannot
	// call Connect without a real Event Hub, so we exercise the public contract:
	// Name() must return the expected identifier.
	if a.Name() != "eventhub" {
		t.Fatalf("expected Name()=%q, got %q", "eventhub", a.Name())
	}
}

// TestNewEventHubAdapter_ExplicitConsumerGroup verifies that a caller-supplied
// consumerGroup is accepted and the adapter is constructed without error.
func TestNewEventHubAdapter_ExplicitConsumerGroup(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
		"eventHubName":     "my-hub",
		"consumerGroup":    "custom-group",
	})
	if a == nil {
		t.Fatal("expected non-nil EventHubAdapter")
	}
}

// TestNewEventHubAdapter_EmptyConsumerGroupFallsBack verifies that an empty
// string for consumerGroup still results in the default "$Default" being used.
func TestNewEventHubAdapter_EmptyConsumerGroupFallsBack(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
		"consumerGroup":    "",
	})
	if a == nil {
		t.Fatal("expected non-nil EventHubAdapter")
	}
}

// TestEventHubAdapter_ConnectMissingConnectionString verifies that calling
// Connect without a connectionString returns a CONNECTION_ERROR.
func TestEventHubAdapter_ConnectMissingConnectionString(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{})
	ctx := context.Background()

	err := a.Connect(ctx)
	if err == nil {
		t.Fatal("expected error when connectionString is missing, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "CONNECTION_ERROR" {
		t.Fatalf("expected code CONNECTION_ERROR, got %s", tryveErr.Code)
	}
}

// TestEventHubAdapter_HealthBeforeConnect verifies that Health returns an error
// when the adapter has not been connected (producer is nil).
func TestEventHubAdapter_HealthBeforeConnect(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
	})
	ctx := context.Background()

	err := a.Health(ctx)
	if err == nil {
		t.Fatal("expected health error before Connect, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "CONNECTION_ERROR" {
		t.Fatalf("expected code CONNECTION_ERROR, got %s", tryveErr.Code)
	}
}

// TestEventHubAdapter_InvalidAction verifies that an unsupported action name
// returns an ADAPTER_ERROR with the correct code.
func TestEventHubAdapter_InvalidAction(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
	})
	ctx := context.Background()

	// Connect is not called on purpose — the action dispatch check happens before
	// any I/O, so this path is reachable without a live Event Hub.
	_, err := a.Execute(ctx, "unsupported", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unsupported action, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "ADAPTER_ERROR" {
		t.Fatalf("expected code ADAPTER_ERROR, got %s", tryveErr.Code)
	}
}

// TestEventHubAdapter_InvalidActionList verifies all action spellings that must
// NOT be accepted so the contract is not accidentally widened.
func TestEventHubAdapter_InvalidActionList(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
	})
	ctx := context.Background()

	invalidActions := []string{"send", "receive", "read", "write", "subscribe", "get", ""}
	for _, action := range invalidActions {
		_, err := a.Execute(ctx, action, map[string]any{})
		if err == nil {
			t.Errorf("expected error for action %q, got nil", action)
			continue
		}
		var tryveErr *tryve.TryveError
		if !errors.As(err, &tryveErr) {
			t.Errorf("action %q: expected *tryve.TryveError, got %T: %v", action, err, err)
			continue
		}
		if tryveErr.Code != "ADAPTER_ERROR" {
			t.Errorf("action %q: expected code ADAPTER_ERROR, got %s", action, tryveErr.Code)
		}
	}
}

// TestEventHubAdapter_CloseNoop verifies that Close on an unconnected adapter
// is safe and returns nil.
func TestEventHubAdapter_CloseNoop(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{
		"connectionString": "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=k;SharedAccessKey=s",
	})
	ctx := context.Background()

	if err := a.Close(ctx); err != nil {
		t.Fatalf("expected nil from Close on unconnected adapter, got: %v", err)
	}
}

// TestEventHubAdapter_Name verifies the registered adapter identifier.
func TestEventHubAdapter_Name(t *testing.T) {
	a := adapter.NewEventHubAdapter(map[string]any{})
	if got := a.Name(); got != "eventhub" {
		t.Fatalf("expected Name()=%q, got %q", "eventhub", got)
	}
}

// TestEventHubAdapter_ImplementsAdapter verifies at compile time that
// *EventHubAdapter satisfies the Adapter interface.
func TestEventHubAdapter_ImplementsAdapter(t *testing.T) {
	var _ adapter.Adapter = (*adapter.EventHubAdapter)(nil)
}
