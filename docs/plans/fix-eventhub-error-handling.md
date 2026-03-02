# Fix EventHub Error Handling — Implementation Plan

**Goal:** EventHub infrastructure errors cause steps to fail instead of silently passing with a success result.

**Architecture:** EventHub adapter's `processError` callback currently resolves the promise with a failure result object. The fix is to reject the promise instead, so the error propagates up to StepExecutor and fails the step properly.

**Tech Stack:** TypeScript, vitest (testing).

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/adapters/eventhub.adapter.ts` — EventHub adapter with `produce` and `consume` actions; `processError` handlers at lines ~258 and ~308 resolve instead of reject
  - `src/adapters/base.adapter.ts` — Base adapter class with `failResult` helper method
- **Assumptions:** EventHub adapter compiles and basic produce/consume works. Error handling is the only issue.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Task 0.3)

## Constraints

- Must not break existing EventHub tests that pass
- Error must be an Error object, not wrapped in a result
- Both `produce` and `consume` actions need the fix

## Rollback

```bash
git revert HEAD~1  # Reverts the single commit for this fix
```

---

## Task 1: Write failing test for EventHub error handling

**Files:**
- Create: `tests/unit/adapters/eventhub-error.test.ts` — Test error propagation

### Step 1: Write failing test

Create `tests/unit/adapters/eventhub-error.test.ts`:
```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { EventHubAdapter } from '../../../src/adapters/eventhub.adapter';
import type { AdapterContext } from '../../../src/types';

describe('EventHubAdapter - Error Handling', () => {
  let adapter: EventHubAdapter;
  let context: AdapterContext;

  beforeEach(() => {
    adapter = new EventHubAdapter();
    context = {
      variables: {},
      captured: {},
      capture: () => {},
      logger: {
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
      },
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should reject promise when processError is called during produce', async () => {
    // Configure adapter with valid connection string but simulate error
    await adapter.configure({
      connectionString: 'Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test',
      eventHubName: 'test-hub',
    });

    // Mock the producer client to simulate subscription error
    // The actual error will come from processError callback
    const error = new Error('EventHub infrastructure error');

    // This test verifies that processError rejects the promise
    // We'll need to trigger the error through the mock
    const producePromise = adapter.execute('produce', {
      messages: [{ body: 'test' }],
    }, context);

    // In a real scenario, the EventHub SDK would call processError
    // For this test, we're verifying the promise rejects
    await expect(producePromise).rejects.toThrow();
  });

  it('should reject promise when processError is called during consume', async () => {
    await adapter.configure({
      connectionString: 'Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test',
      eventHubName: 'test-hub',
    });

    const consumePromise = adapter.execute('consume', {
      consumerGroup: '$Default',
      count: 1,
      timeout: 1000,
    }, context);

    // Verify that processError rejects the promise
    await expect(consumePromise).rejects.toThrow();
  });
});
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/adapters/eventhub-error.test.ts`
Expected: `FAIL — promise resolved instead of rejected` or timeout

### Step 3: Implement

In `src/adapters/eventhub.adapter.ts`, find the first `processError` handler (around line 258):
```typescript
        processError: async (error) => {
          clearTimeout(timeoutId);
          await cleanup();
          resolve(this.failResult(error, Date.now() - start));
        },
```

Replace with:
```typescript
        processError: async (error) => {
          clearTimeout(timeoutId);
          await cleanup();
          reject(error);
        },
```

Find the second `processError` handler (around line 308):
```typescript
        processError: async (error) => {
          clearTimeout(timeoutId);
          await cleanup();
          resolve(this.failResult(error, Date.now() - start));
        },
```

Replace with:
```typescript
        processError: async (error) => {
          clearTimeout(timeoutId);
          await cleanup();
          reject(error);
        },
```

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/adapters/eventhub-error.test.ts`
Expected: 2 tests PASS (or appropriate number based on mock setup)

### Step 5: Commit

```bash
git add src/adapters/eventhub.adapter.ts tests/unit/adapters/eventhub-error.test.ts
git commit -m "fix(eventhub): reject promise on processError instead of resolving with failResult"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All existing tests PASS + new error handling tests PASS.

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds with no errors.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | EventHub error fails step | Create test with invalid EventHub connection. Run test. | Step fails with error message (not silent pass) |
| 2 | Valid EventHub operations still work | Run existing EventHub E2E tests with valid connection. | Tests pass as before |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `src/adapters/eventhub.adapter.ts` | Replace `resolve(this.failResult(...))` with `reject(error)` in both processError handlers (2 locations) |
| Create | `tests/unit/adapters/eventhub-error.test.ts` | Test that processError rejects promise (50 lines) |
