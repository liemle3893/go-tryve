package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// ErrLockTimeout is returned by Acquire when the timeout elapses before
// the lock could be taken.
var ErrLockTimeout = errors.New("e2e lock timeout")

// Lock is a released-on-close file lock, suitable for serialising E2E
// runs that share the main repository directory. The on-disk file is
// only a handle — its contents are irrelevant.
type Lock struct {
	f *os.File
}

// Acquire blocks up to timeout waiting for an exclusive flock on path.
// The file is created if it does not exist (0o644). The process holds
// the lock until Release is called or the process exits.
func Acquire(ctx context.Context, path string, timeout time.Duration) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}

	deadline := time.Now().Add(timeout)
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return &Lock{f: f}, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && err != unix.EAGAIN {
			_ = f.Close()
			return nil, fmt.Errorf("flock %s: %w", path, err)
		}
		// Another holder has the lock. Back off briefly and retry.
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("%w: waited %s on %s", ErrLockTimeout, timeout, path)
		}
	}
}

// Release unlocks and closes the handle. Idempotent — safe to call from
// defer even on the error path.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = unix.Flock(int(l.f.Fd()), unix.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}
