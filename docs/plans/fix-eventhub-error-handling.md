# Fix EventHub Error Handling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix `processError` handlers in the EventHub adapter so that Azure infrastructure errors cause the test step to fail (reject) rather than silently resolve.

**Architecture:** `EventHubAdapter.waitFor` and `EventHubAdapter.consume` both return a `new Promise`. Their `processError` callbacks currently call `resolve(this.failResult(error, ...))` which resolves the promise with a "failed" payload — but `StepExecutor` treats any resolved return as a passed step. The fix is to call `reject(error)` instead, so the error propagates up through `withRetry` in `StepExecutor` and the step is marked `failed`.

**Tech Stack:** TypeScript, Vitest, `src/adapters/eventhub.adapter.ts` (existing)

---

### Task 1: Write the failing unit test

**Files:**
- Create: `tests/unit/adapters/eventhub.test.ts`

**Step 1: Write the failing test**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'

// We test the processError path by constructing a minimal mock of the
// EventHubConsumerClient and injecting an error into processError.

describe('EventHubAdapter processError handling', () => {
  it('waitFor rejects when processError is called with an infrastructure error', async () => {
    // Build a fake consumerClient that immediately calls processError
    const infraError = new Error('AMQP link detached')

    const fakeConsumerClient = {
      subscribe: vi.fn().mockImplementation(({ processError }) => {
        // Simulate infrastructure error fired immediately
        setImmediate(() => processError(infraError))
        return { close: vi.fn().mockResolvedValue(undefined) }
      }),
      close: vi.fn().mockResolvedValue(undefined),
    }

    const fakeProducerClient = {
      getEventHubProperties: vi.fn().mockResolvedValue({}),
      createBatch: vi.fn(),
      sendBatch: vi.fn(),
      close: vi.fn().mockResolvedValue(undefined),
    }

    // Dynamically import the adapter after setting up mocks
    vi.doMock('@azure/event-hubs', () => ({
      EventHubProducerClient: vi.fn(() => fakeProducerClient),
      EventHubConsumerClient: vi.fn(() => fakeConsumerClient),
    }))

    const { EventHubAdapter } = await import('../../../src/adapters/eventhub.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new EventHubAdapter(
      { connectionString: 'fake-connection-string', consumerGroup: '$Default' },
      logger
    )

    // Manually set the internal clients (bypassing connect() which needs real Azure)
    ;(adapter as any).producerClient = fakeProducerClient
    ;(adapter as any).consumerClient = fakeConsumerClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    // waitFor should reject, not resolve, when processError fires
    await expect(
      adapter.execute('waitFor', { topic: 'test-topic', timeout: 5000 }, ctx)
    ).rejects.toThrow()

    vi.doUnmock('@azure/event-hubs')
  })

  it('consume rejects when processError is called with an infrastructure error', async () => {
    const infraError = new Error('Connection reset')

    const fakeConsumerClient = {
      subscribe: vi.fn().mockImplementation(({ processError }) => {
        setImmediate(() => processError(infraError))
        return { close: vi.fn().mockResolvedValue(undefined) }
      }),
      close: vi.fn().mockResolvedValue(undefined),
    }

    const fakeProducerClient = {
      getEventHubProperties: vi.fn().mockResolvedValue({}),
      close: vi.fn().mockResolvedValue(undefined),
    }

    vi.doMock('@azure/event-hubs', () => ({
      EventHubProducerClient: vi.fn(() => fakeProducerClient),
      EventHubConsumerClient: vi.fn(() => fakeConsumerClient),
    }))

    const { EventHubAdapter } = await import('../../../src/adapters/eventhub.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new EventHubAdapter(
      { connectionString: 'fake-connection-string', consumerGroup: '$Default' },
      logger
    )
    ;(adapter as any).producerClient = fakeProducerClient
    ;(adapter as any).consumerClient = fakeConsumerClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    await expect(
      adapter.execute('consume', { topic: 'test-topic', count: 1, timeout: 5000 }, ctx)
    ).rejects.toThrow()

    vi.doUnmock('@azure/event-hubs')
  })
})
```

**Step 2: Run test to verify it fails**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/adapters/eventhub.test.ts 2>&1 | tail -30
```

Expected: FAIL — both tests fail because `processError` currently resolves, not rejects.

**Step 3: Implement the fix**

Modify `src/adapters/eventhub.adapter.ts`.

In the `waitFor` method, change the `processError` callback (around line 258):

```typescript
// BEFORE:
processError: async (error) => {
  clearTimeout(timeoutId);
  await cleanup();
  resolve(this.failResult(error, Date.now() - start));
},

// AFTER:
processError: async (error) => {
  clearTimeout(timeoutId);
  await cleanup();
  reject(error instanceof Error ? error : new Error(String(error)));
},
```

Also change the Promise signature from `new Promise<AdapterStepResult>((resolve) => {` to `new Promise<AdapterStepResult>((resolve, reject) => {` in the `waitFor` method.

In the `consume` method, apply the same fix to its `processError` callback (around line 308):

```typescript
// BEFORE:
processError: async (error) => {
  clearTimeout(timeoutId);
  await cleanup();
  resolve(this.failResult(error, Date.now() - start));
},

// AFTER:
processError: async (error) => {
  clearTimeout(timeoutId);
  await cleanup();
  reject(error instanceof Error ? error : new Error(String(error)));
},
```

Also change the consume Promise signature from `new Promise<AdapterStepResult>((resolve) => {` to `new Promise<AdapterStepResult>((resolve, reject) => {`.

**Step 4: Run test to verify it passes**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/adapters/eventhub.test.ts 2>&1 | tail -20
```

Expected: PASS — both tests pass.

**Step 5: Commit**

```bash
cd /tmp/e2e-runner && git add src/adapters/eventhub.adapter.ts tests/unit/adapters/eventhub.test.ts && git commit -m "fix(adapters): reject on EventHub processError instead of resolving

processError callbacks in waitFor and consume were calling resolve()
with a failResult, silently masking infrastructure errors. Now they
call reject() so the step correctly fails in StepExecutor.

Closes Task 0.3"
```
