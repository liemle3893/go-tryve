package e2e

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLock_AcquireAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	ctx := context.Background()
	lock, err := Acquire(ctx, path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	// Idempotent release.
	if err := lock.Release(); err != nil {
		t.Errorf("double release: %v", err)
	}
}

func TestLock_SerialisesConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	ctx := context.Background()

	lock1, err := Acquire(ctx, path, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// A second acquire inside a very short timeout must fail, proving
	// lock1 is actually held.
	_, err = Acquire(ctx, path, 100*time.Millisecond)
	if !errors.Is(err, ErrLockTimeout) {
		t.Errorf("want ErrLockTimeout while held, got %v", err)
	}

	// Release — a third acquire should now succeed quickly.
	if err := lock1.Release(); err != nil {
		t.Fatal(err)
	}
	lock2, err := Acquire(ctx, path, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("post-release acquire: %v", err)
	}
	_ = lock2.Release()
}

func TestLock_HandoffOnRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	ctx := context.Background()

	lock1, err := Acquire(ctx, path, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Second goroutine waits for the lock; release triggers handoff.
	var wg sync.WaitGroup
	wg.Add(1)
	var acquireErr error
	go func() {
		defer wg.Done()
		l, err := Acquire(ctx, path, 2*time.Second)
		if err != nil {
			acquireErr = err
			return
		}
		_ = l.Release()
	}()

	time.Sleep(300 * time.Millisecond)
	_ = lock1.Release()
	wg.Wait()
	if acquireErr != nil {
		t.Errorf("waiter did not acquire: %v", acquireErr)
	}
}

func TestLock_ContextCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	// Hold the lock.
	held, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	_, err = Acquire(ctx, path, 5*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}
