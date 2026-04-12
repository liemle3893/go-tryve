package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/liemle3893/e2e-runner/internal/watcher"
)

// waitTimeout is how long a test waits for the onChange channel to receive.
// It must be long enough for the 500 ms debounce plus OS event delivery.
const waitTimeout = 3 * time.Second

// receiveWithTimeout blocks until ch receives a value or the timeout elapses.
// It returns true if a value was received, false on timeout.
func receiveWithTimeout(ch <-chan struct{}, d time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(d):
		return false
	}
}

// startWatcher creates a Watcher for dirs, launches it in a background
// goroutine, and returns the cancel function together with the watcher.
// The test is failed immediately if construction fails.
func startWatcher(t *testing.T, dirs []string, onChange func()) (*watcher.Watcher, context.CancelFunc) {
	t.Helper()
	w, err := watcher.New(dirs, onChange)
	if err != nil {
		t.Fatalf("watcher.New() unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := w.Start(ctx); err != nil {
			// Only log; the test may have already completed.
			t.Logf("watcher.Start() returned error: %v", err)
		}
	}()

	return w, cancel
}

// TestNew_InvalidCallback verifies that New returns an error when onChange is nil.
func TestNew_InvalidCallback(t *testing.T) {
	dir := t.TempDir()
	_, err := watcher.New([]string{dir}, nil)
	if err == nil {
		t.Fatal("watcher.New() expected error for nil callback, got nil")
	}
}

// TestNew_NonExistentDir verifies that New returns an error when a directory
// does not exist.
func TestNew_NonExistentDir(t *testing.T) {
	_, err := watcher.New([]string{"/does/not/exist/ever"}, func() {})
	if err == nil {
		t.Fatal("watcher.New() expected error for non-existent dir, got nil")
	}
}

// TestWatcher_TestFileModify verifies that modifying an existing .test.yaml
// file triggers the onChange callback within the timeout window.
func TestWatcher_TestFileModify(t *testing.T) {
	root := t.TempDir()

	// Create a test file before the watcher starts.
	testFile := filepath.Join(root, "example.test.yaml")
	if err := os.WriteFile(testFile, []byte("name: initial"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	// Give fsnotify a moment to register the watch before we modify the file.
	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(testFile, []byte("name: modified"), 0o644); err != nil {
		t.Fatalf("modifying test file: %v", err)
	}

	if !receiveWithTimeout(called, waitTimeout) {
		t.Error("onChange was not called after modifying .test.yaml within the timeout window")
	}
}

// TestWatcher_TestFileCreate verifies that creating a new .test.yaml file in
// a watched directory triggers the onChange callback.
func TestWatcher_TestFileCreate(t *testing.T) {
	root := t.TempDir()

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	newFile := filepath.Join(root, "new.test.yaml")
	if err := os.WriteFile(newFile, []byte("name: new"), 0o644); err != nil {
		t.Fatalf("creating new test file: %v", err)
	}

	if !receiveWithTimeout(called, waitTimeout) {
		t.Error("onChange was not called after creating .test.yaml within the timeout window")
	}
}

// TestWatcher_TestFileDelete verifies that deleting a .test.yaml file triggers
// the onChange callback.
func TestWatcher_TestFileDelete(t *testing.T) {
	root := t.TempDir()

	testFile := filepath.Join(root, "to-delete.test.yaml")
	if err := os.WriteFile(testFile, []byte("name: delete-me"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	if err := os.Remove(testFile); err != nil {
		t.Fatalf("removing test file: %v", err)
	}

	if !receiveWithTimeout(called, waitTimeout) {
		t.Error("onChange was not called after deleting .test.yaml within the timeout window")
	}
}

// TestWatcher_YmlExtension verifies that .test.yml (as well as .test.yaml)
// files are monitored.
func TestWatcher_YmlExtension(t *testing.T) {
	root := t.TempDir()

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	ymlFile := filepath.Join(root, "suite.test.yml")
	if err := os.WriteFile(ymlFile, []byte("name: yml"), 0o644); err != nil {
		t.Fatalf("creating .test.yml file: %v", err)
	}

	if !receiveWithTimeout(called, waitTimeout) {
		t.Error("onChange was not called after creating .test.yml within the timeout window")
	}
}

// TestWatcher_NonTestFileIgnored verifies that modifying a non-test file does
// NOT trigger the onChange callback.
func TestWatcher_NonTestFileIgnored(t *testing.T) {
	root := t.TempDir()

	plainFile := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(plainFile, []byte("version: 1"), 0o644); err != nil {
		t.Fatalf("creating plain file: %v", err)
	}

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(plainFile, []byte("version: 2"), 0o644); err != nil {
		t.Fatalf("modifying plain file: %v", err)
	}

	// The callback must NOT fire within the debounce window + margin.
	// We wait debounce (500ms) + 300ms margin = 800ms.
	if receiveWithTimeout(called, 800*time.Millisecond) {
		t.Error("onChange was unexpectedly called after modifying a non-test file")
	}
}

// TestWatcher_HiddenDirIgnored verifies that test files inside hidden
// directories do not trigger the onChange callback.
func TestWatcher_HiddenDirIgnored(t *testing.T) {
	root := t.TempDir()

	hiddenDir := filepath.Join(root, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("creating hidden dir: %v", err)
	}

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	hiddenTest := filepath.Join(hiddenDir, "hidden.test.yaml")
	if err := os.WriteFile(hiddenTest, []byte("name: hidden"), 0o644); err != nil {
		t.Fatalf("creating hidden test file: %v", err)
	}

	if receiveWithTimeout(called, 800*time.Millisecond) {
		t.Error("onChange was unexpectedly called for a test file inside a hidden directory")
	}
}

// TestWatcher_NodeModulesIgnored verifies that test files inside node_modules
// directories do not trigger the onChange callback.
func TestWatcher_NodeModulesIgnored(t *testing.T) {
	root := t.TempDir()

	nmDir := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatalf("creating node_modules dir: %v", err)
	}

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	nmTest := filepath.Join(nmDir, "dep.test.yaml")
	if err := os.WriteFile(nmTest, []byte("name: dep"), 0o644); err != nil {
		t.Fatalf("creating node_modules test file: %v", err)
	}

	if receiveWithTimeout(called, 800*time.Millisecond) {
		t.Error("onChange was unexpectedly called for a test file inside node_modules")
	}
}

// TestWatcher_RecursiveSubdirectory verifies that test files inside
// sub-directories that exist when the watcher is created are also monitored.
func TestWatcher_RecursiveSubdirectory(t *testing.T) {
	root := t.TempDir()

	subDir := filepath.Join(root, "integration", "api")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("creating nested dir: %v", err)
	}

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	nestedTest := filepath.Join(subDir, "nested.test.yaml")
	if err := os.WriteFile(nestedTest, []byte("name: nested"), 0o644); err != nil {
		t.Fatalf("creating nested test file: %v", err)
	}

	if !receiveWithTimeout(called, waitTimeout) {
		t.Error("onChange was not called after creating .test.yaml in a nested sub-directory")
	}
}

// TestWatcher_Debounce verifies that multiple rapid changes result in only a
// single (or at most a small number of) onChange invocations rather than one
// per event.
func TestWatcher_Debounce(t *testing.T) {
	root := t.TempDir()

	testFile := filepath.Join(root, "debounce.test.yaml")
	if err := os.WriteFile(testFile, []byte("name: v0"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	var callCount atomic.Int32
	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		callCount.Add(1)
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	// Fire 10 rapid writes within a very short window.
	for i := 1; i <= 10; i++ {
		content := []byte("name: v" + string(rune('0'+i)))
		_ = os.WriteFile(testFile, content, 0o644)
		time.Sleep(20 * time.Millisecond)
	}

	// Wait long enough for the debounce to settle (at least 500 ms after the
	// last write) plus some margin.
	if !receiveWithTimeout(called, waitTimeout) {
		t.Fatal("onChange was not called at all after rapid writes")
	}

	// Allow the debounce window to fully expire before counting.
	time.Sleep(700 * time.Millisecond)

	count := callCount.Load()
	if count > 3 {
		// Strictly the debounce should collapse all writes to 1, but allow a
		// small number to account for OS-level batching variations.
		t.Errorf("onChange was called %d times for 10 rapid writes; expected ≤ 3 (debounce should collapse events)", count)
	}
}

// TestWatcher_Stop verifies that Stop prevents further callback invocations.
func TestWatcher_Stop(t *testing.T) {
	root := t.TempDir()

	testFile := filepath.Join(root, "stop.test.yaml")
	if err := os.WriteFile(testFile, []byte("name: initial"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	called := make(chan struct{}, 1)
	w, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	// Stop the watcher explicitly before making a change.
	w.Stop()

	// Write after Stop — callback must not fire.
	_ = os.WriteFile(testFile, []byte("name: after-stop"), 0o644)

	if receiveWithTimeout(called, 800*time.Millisecond) {
		t.Error("onChange was called after Stop(); expected no invocation")
	}
}

// TestWatcher_NewDirectoryWatched verifies that a sub-directory created after
// the watcher starts is also monitored for test file changes.
func TestWatcher_NewDirectoryWatched(t *testing.T) {
	root := t.TempDir()

	called := make(chan struct{}, 1)
	_, cancel := startWatcher(t, []string{root}, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	// Create a new sub-directory dynamically.
	newDir := filepath.Join(root, "dynamic")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("creating dynamic dir: %v", err)
	}

	// Give fsnotify time to process the directory creation and re-watch.
	time.Sleep(200 * time.Millisecond)

	newTest := filepath.Join(newDir, "dynamic.test.yaml")
	if err := os.WriteFile(newTest, []byte("name: dynamic"), 0o644); err != nil {
		t.Fatalf("creating test file in dynamic dir: %v", err)
	}

	if !receiveWithTimeout(called, waitTimeout) {
		t.Error("onChange was not called after creating .test.yaml in a newly created directory")
	}
}
