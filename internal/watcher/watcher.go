// Package watcher provides file-system watching for .test.yaml/.test.yml files.
// It debounces change events so that rapid bursts of edits (e.g. editor writes)
// result in a single callback invocation after the burst has settled.
package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// debounceDuration is how long the watcher waits after the last event
	// before triggering the onChange callback.
	debounceDuration = 500 * time.Millisecond
)

// Watcher monitors one or more directory trees for changes to .test.yaml and
// .test.yml files, calling onChange after each settled burst of events.
type Watcher struct {
	dirs     []string
	onChange func()
	fsw      *fsnotify.Watcher

	mu      sync.Mutex
	timer   *time.Timer
	stopped bool
}

// New creates a Watcher that recursively monitors dirs and calls onChange
// (debounced at 500 ms) when any .test.yaml or .test.yml file is created,
// modified, or deleted.
//
// Parameters:
//   - dirs     – list of root directories to watch recursively.
//   - onChange – callback invoked after a debounced change event fires.
//
// Returns an error if the underlying fsnotify watcher cannot be created or if
// any of the provided directories does not exist.
func New(dirs []string, onChange func()) (*Watcher, error) {
	if onChange == nil {
		return nil, fmt.Errorf("watcher: onChange callback must not be nil")
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watcher: creating fsnotify watcher: %w", err)
	}

	w := &Watcher{
		dirs:     dirs,
		onChange: onChange,
		fsw:      fsw,
	}

	for _, dir := range dirs {
		if err := w.addDirRecursive(dir); err != nil {
			fsw.Close() //nolint:errcheck
			return nil, fmt.Errorf("watcher: watching %q: %w", dir, err)
		}
	}

	return w, nil
}

// Start begins processing file-system events and blocks until ctx is cancelled.
// It is safe to call Stop concurrently; once stopped, Start returns nil.
func (w *Watcher) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			w.Stop()
			return nil

		case event, ok := <-w.fsw.Events:
			if !ok {
				// Channel closed; watcher was stopped.
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			log.Printf("WARN  watcher: fsnotify error: %v", err)
		}
	}
}

// Stop shuts down the underlying fsnotify watcher and cancels any pending
// debounce timer. It is idempotent.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}
	w.stopped = true

	if w.timer != nil {
		w.timer.Stop()
	}

	if err := w.fsw.Close(); err != nil {
		log.Printf("WARN  watcher: closing fsnotify: %v", err)
	}
}

// handleEvent processes a single fsnotify event.
// It ignores non-test files, hidden directories, and node_modules, and resets
// the debounce timer for every qualifying event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// When a new directory is created, register it (and its children) for
	// watching so that files created inside it are observed.
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if !isExcludedDir(filepath.Base(path)) {
				if err := w.addDirRecursive(path); err != nil {
					log.Printf("WARN  watcher: could not watch new dir %q: %v", path, err)
				}
			}
			// Directory creation alone does not trigger the callback.
			return
		}
	}

	// Filter: only react to test file extensions.
	if !isTestFile(path) {
		return
	}

	// Filter: ignore paths inside excluded directories.
	if containsExcludedSegment(path) {
		return
	}

	// Only act on create, write, and remove events.
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) && !event.Has(fsnotify.Remove) {
		return
	}

	w.scheduleCallback()
}

// scheduleCallback resets the debounce timer so that onChange is called 500 ms
// after the last qualifying event.
func (w *Watcher) scheduleCallback() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	if w.timer != nil {
		w.timer.Stop()
	}

	w.timer = time.AfterFunc(debounceDuration, func() {
		w.mu.Lock()
		stopped := w.stopped
		w.mu.Unlock()

		if !stopped {
			w.onChange()
		}
	})
}

// addDirRecursive walks path and registers every non-excluded sub-directory
// (including path itself) with the fsnotify watcher.
func (w *Watcher) addDirRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// For the root itself, propagate the error so New() can surface it.
			if path == root {
				return err
			}
			// For sub-paths, log and continue rather than aborting the walk.
			log.Printf("WARN  watcher: walking %q: %v", path, err)
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		name := info.Name()
		// Prune excluded directories during the walk (but never the root itself).
		if path != root && isExcludedDir(name) {
			return filepath.SkipDir
		}

		if err := w.fsw.Add(path); err != nil {
			return fmt.Errorf("adding %q: %w", path, err)
		}
		return nil
	})
}

// isTestFile reports whether path refers to a .test.yaml or .test.yml file.
func isTestFile(path string) bool {
	name := filepath.Base(path)
	return strings.HasSuffix(name, ".test.yaml") || strings.HasSuffix(name, ".test.yml")
}

// isExcludedDir reports whether a directory name should be skipped entirely.
// Hidden directories (those starting with ".") and "node_modules" are excluded.
func isExcludedDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules"
}

// containsExcludedSegment reports whether any path segment is excluded so that
// events from files inside excluded trees are silently dropped.
func containsExcludedSegment(path string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if isExcludedDir(segment) {
			return true
		}
	}
	return false
}
