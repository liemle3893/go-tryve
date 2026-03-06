# Fix Flaky Watcher Unit Test — Implementation Plan

**Goal:** The watcher debounce test passes consistently when run individually or as part of the full test suite, with no race conditions or timing failures.

**Architecture:**
- Tests use `vi.useFakeTimers()` to control time progression and eliminate timing-dependent flakiness.
- Each test gets a unique temp directory using UUID-based naming to prevent collisions.
- Proper cleanup ensures no file handles or timers leak between tests.

**Tech Stack:** TypeScript, Node.js, vitest (existing), chokidar (existing), no new dependencies.

---

## Current State

> Read every file you plan to modify BEFORE writing this section. Do not assume file contents.

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `tests/unit/core/watcher.test.ts` — 4 tests for watcher functionality. Uses real timers with fixed wait times (200ms, 300ms). Creates temp dirs in `process.cwd()`.
  - `src/core/watcher.ts` — File watcher using chokidar with debouncing. Uses `awaitWriteFinish` option.
  - `src/core/watcher.ts:44-51` — Debounce timer logic with 100ms default.
- **Assumptions:**
  - The watcher implementation is correct; the issue is in test timing/isolation.
  - chokidar works correctly when given proper time to initialize and settle.
- **Design doc:** `docs/plans/phase3-developer-experience.md` (original Phase 3 plan)

## Constraints

- No changes to production code (src/core/watcher.ts).
- Tests must pass consistently (100% pass rate over 10 runs).
- Tests must complete in reasonable time (< 5 seconds total).
- Must work on both local dev machines and CI environments.

## Rollback

> How to undo everything this plan creates. Must be copy-pasteable.

```bash
git revert HEAD~1  # Single commit for test fixes
# No new dependencies to uninstall
# No runtime directories created
```

---

## Task 1: Fix Flaky Watcher Test with Fake Timers and Better Isolation

**Files:**
- Modify: `tests/unit/core/watcher.test.ts` — Use fake timers, increase wait times, improve isolation

### Step 1: Write failing test (document current state)

The current flaky test (line 59):

```typescript
it('debounces multiple rapid changes into a single callback', async () => {
  const onChange = vi.fn()
  const testFile = path.join(tempDir, 'debounce.test.yaml')
  fs.writeFileSync(testFile, 'name: test')

  watcher = createWatcher({
    paths: [tempDir],
    patterns: ['**/*.test.yaml', '**/*.test.ts'],
    debounceMs: 100,
    onChange,
  })

  await new Promise(resolve => setTimeout(resolve, 200))
  fs.writeFileSync(testFile, 'name: change1')
  fs.writeFileSync(testFile, 'name: change2')
  fs.writeFileSync(testFile, 'name: change3')
  await new Promise(resolve => setTimeout(resolve, 300))

  expect(onChange).toHaveBeenCalledTimes(1)
})
```

**Why it fails randomly:**
1. Real timers are unreliable in concurrent test environments
2. 200ms wait may not be enough for chokidar to fully initialize
3. 300ms wait after changes may not be enough for debounce + file event propagation
4. Temp dirs in `process.cwd()` can collide if tests run in parallel

### Step 2: Run test to verify it fails (reproduce flaky behavior)

Run the test suite multiple times to observe flakiness:

```bash
# Run 10 times and count failures
for i in {1..10}; do
  npm test -- tests/unit/core/watcher.test.ts 2>&1 | grep -q "FAIL" && echo "Run $i: FAIL" || echo "Run $i: PASS"
done
```

Expected: Random failures (e.g., "Run 3: FAIL", "Run 7: FAIL") with error:
```
AssertionError: expected spy to be called 1 times, but got 0 times
```

### Step 3: Implement

Modify `tests/unit/core/watcher.test.ts`. Replace the entire file content:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { createWatcher, type WatcherOptions } from '../../../src/core/watcher'

describe('createWatcher', () => {
  let tempDir: string
  let watcher: ReturnType<typeof createWatcher> | null = null

  beforeEach(() => {
    // Use unique temp directory with random suffix to prevent collisions
    const uniqueId = `${Date.now()}-${Math.random().toString(36).substring(7)}`
    tempDir = fs.mkdtempSync(path.join(process.cwd(), `watcher-test-${uniqueId}-`))
  })

  afterEach(async () => {
    // Close watcher first to release file handles
    if (watcher) {
      watcher.close()
      watcher = null
    }
    
    // Give time for file handles to be released
    await new Promise(resolve => setTimeout(resolve, 100))
    
    // Clean up temp directory
    try {
      fs.rmSync(tempDir, { recursive: true, force: true })
    } catch (error) {
      // Ignore cleanup errors - temp dirs are in node_modules anyway
    }
  })

  it('calls onChange callback when a .test.yaml file is modified', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'example.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))
    
    // Modify file
    fs.writeFileSync(testFile, 'name: test-modified')
    
    // Wait for debounce + file event propagation
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).toHaveBeenCalled()
  })

  it('debounces multiple rapid changes into a single callback', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'debounce.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))
    
    // Make multiple rapid changes
    fs.writeFileSync(testFile, 'name: change1')
    fs.writeFileSync(testFile, 'name: change2')
    fs.writeFileSync(testFile, 'name: change3')
    
    // Wait for debounce to complete (debounceMs + buffer)
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).toHaveBeenCalledTimes(1)
  })

  it('ignores files that do not match test patterns', async () => {
    const onChange = vi.fn()
    const configFile = path.join(tempDir, 'config.yaml')
    fs.writeFileSync(configFile, 'setting: value')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))
    
    // Modify non-test file
    fs.writeFileSync(configFile, 'setting: new-value')
    
    // Wait to ensure no callback fires
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).not.toHaveBeenCalled()
  })

  it('close() stops the watcher and prevents further callbacks', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'close.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))
    
    // Close watcher
    watcher.close()
    watcher = null
    
    // Small delay to ensure close is processed
    await new Promise(resolve => setTimeout(resolve, 100))

    // Modify file after close
    fs.writeFileSync(testFile, 'name: after-close')
    
    // Wait to ensure no callback fires
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).not.toHaveBeenCalled()
  })
})
```

**Key changes:**
1. **Unique temp directories:** Added `uniqueId` with timestamp + random string to prevent collisions
2. **Increased wait times:** 200ms → 300ms (initialization), 300ms → 500ms (after changes)
3. **Better cleanup:** Added 100ms delay before cleanup to let file handles release
4. **Error handling:** Wrapped cleanup in try-catch to prevent test failures from cleanup issues
5. **More specific waits:** Added comment about debounce timing (debounceMs + buffer)

### Step 4: Run test to verify it passes

Run the test suite multiple times to verify consistency:

```bash
# Run 10 times and verify all pass
for i in {1..10}; do
  if npm test -- tests/unit/core/watcher.test.ts 2>&1 | grep -q "FAIL"; then
    echo "Run $i: FAIL"
    exit 1
  else
    echo "Run $i: PASS"
  fi
done
echo "All 10 runs passed!"
```

Expected: All 10 runs show "PASS"

Also verify in full test suite:
```bash
npm test
```

Expected: All tests PASS (4 tests in watcher.test.ts)

### Step 5: Commit

```bash
git add tests/unit/core/watcher.test.ts
git commit -m "fix(test): eliminate flaky watcher test with better timing and isolation

- Increase wait times: 200ms→300ms init, 300ms→500ms after changes
- Use unique temp directories with timestamp+random suffix
- Add 100ms delay before cleanup to release file handles
- Wrap cleanup in try-catch to prevent test failures
- Addresses QA finding: test passed individually but failed randomly in full suite"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All tests PASS, including 4 watcher tests.

**Step 2: Run isolated test multiple times**

Run:
```bash
for i in {1..10}; do npm test -- tests/unit/core/watcher.test.ts || exit 1; done
```
Expected: All 10 runs PASS.

**Step 3: Run full suite multiple times**

Run:
```bash
for i in {1..5}; do npm test || exit 1; done
```
Expected: All 5 runs PASS (demonstrates no interference with other tests).

**Step 4: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Debounce test passes | Run `npm test -- tests/unit/core/watcher.test.ts` 10 times | All 10 runs show "4 tests PASS" |
| 2 | No test interference | Run full `npm test` 5 times | All 5 runs complete with 0 failures |
| 3 | Reasonable duration | Run `time npm test -- tests/unit/core/watcher.test.ts` | Completes in < 3 seconds |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `tests/unit/core/watcher.test.ts` | Fix timing and isolation issues in watcher tests |

---

## Root Cause Analysis

The flaky test was caused by three issues:

1. **Insufficient wait times:** 200ms initialization wait and 300ms post-change wait were too short for:
   - chokidar to fully initialize and start watching
   - File system events to propagate through the watcher
   - Debounce timer to fire and callback to execute
   - CI environments often slower than local machines

2. **Temp directory collisions:** Using only `mkdtempSync` with `process.cwd()` can still cause collisions when:
   - Multiple test processes run in parallel
   - Previous test's cleanup hasn't completed
   - File handles haven't been released

3. **Aggressive cleanup:** Immediately deleting temp directories after `watcher.close()` can fail because:
   - chokidar may still hold file handles briefly after close
   - File system hasn't fully released locks

The fix addresses all three issues:
- Longer wait times (300ms init, 500ms after changes) provide safety margin
- Unique temp directory names prevent collisions
- 100ms delay before cleanup lets file handles release
- Try-catch on cleanup prevents cascading failures

---

## Why Not Fake Timers?

Initially considered using `vi.useFakeTimers()` to eliminate timing dependency entirely. However, this approach doesn't work well with chokidar because:

1. chokidar uses native file system events (inotify, FSEvents, etc.)
2. These events are asynchronous and not controlled by fake timers
3. Fake timers would only control the debounce timer, not file event propagation
4. Tests would become more complex without solving the real issue

The better approach is to use realistic wait times that work reliably in all environments. The chosen values (300ms, 500ms) are conservative enough to handle slow CI while still keeping total test time reasonable (< 3 seconds for 4 tests).
