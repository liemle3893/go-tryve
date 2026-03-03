# Wire Assertion Engine Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the no-op `validateAssertions` stub in `StepExecutor` with a real call to the shared assertion runner so that step-level assertions actually fire.

**Architecture:** `StepExecutor.validateAssertions` currently logs assertions and does nothing else. The project already has a complete `runAssertion` function in `src/assertions/assertion-runner.ts`. The fix is to import and call it inside `validateAssertions`, handling both single-assertion objects and arrays.

**Tech Stack:** TypeScript, Vitest, `src/assertions/assertion-runner.ts` (existing), `src/core/step-executor.ts` (existing)

---

### Task 1: Write the failing unit test

**Files:**
- Create: `tests/unit/core/step-executor.test.ts`

**Step 1: Write the failing test**

```typescript
import { describe, it, expect, vi } from 'vitest'
import { StepExecutor } from '../../../src/core/step-executor'
import { AdapterRegistry } from '../../../src/adapters'
import type { AdapterContext, InterpolationContext, UnifiedStep } from '../../../src/types'

function makeLogger() {
  return { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
}

function makeContext(): AdapterContext {
  const captured: Record<string, unknown> = {}
  return {
    variables: {},
    captured,
    capture: (name, value) => { captured[name] = value },
    logger: makeLogger(),
    baseUrl: 'http://localhost',
    cookieJar: new Map(),
  }
}

function makeInterpolationContext(): InterpolationContext {
  return { variables: {}, captured: {}, baseUrl: 'http://localhost', env: {} }
}

describe('StepExecutor.validateAssertions', () => {
  it('throws AssertionError when assert.equals does not match adapter result', async () => {
    const mockAdapter = {
      execute: vi.fn().mockResolvedValue({ success: true, data: { value: 'actual' }, duration: 1 }),
      connect: vi.fn(),
      disconnect: vi.fn(),
      healthCheck: vi.fn(),
      isConnected: vi.fn().mockReturnValue(true),
      name: 'mock',
    }
    const registry = new AdapterRegistry()
    registry.register('http' as any, mockAdapter as any)

    const executor = new StepExecutor(registry, {
      defaultRetries: 0,
      retryDelay: 0,
      logger: makeLogger(),
    })

    const step: UnifiedStep = {
      id: 'test-step',
      adapter: 'http' as any,
      action: 'request',
      params: { url: '/test', method: 'GET' },
      assert: { equals: 'expected' },  // value is 'actual', so this should fail
    }

    await expect(
      executor.executeStep(step, makeContext(), makeInterpolationContext())
    ).rejects.toThrow('AssertionError')
  })

  it('passes when assert.equals matches adapter result data', async () => {
    const mockAdapter = {
      execute: vi.fn().mockResolvedValue({ success: true, data: 'expected', duration: 1 }),
      connect: vi.fn(),
      disconnect: vi.fn(),
      healthCheck: vi.fn(),
      isConnected: vi.fn().mockReturnValue(true),
      name: 'mock',
    }
    const registry = new AdapterRegistry()
    registry.register('http' as any, mockAdapter as any)

    const executor = new StepExecutor(registry, {
      defaultRetries: 0,
      retryDelay: 0,
      logger: makeLogger(),
    })

    const step: UnifiedStep = {
      id: 'test-step',
      adapter: 'http' as any,
      action: 'request',
      params: { url: '/test', method: 'GET' },
      assert: { equals: 'expected' },
    }

    const result = await executor.executeStep(step, makeContext(), makeInterpolationContext())
    expect(result.status).toBe('passed')
  })
})
```

**Step 2: Run test to verify it fails**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/core/step-executor.test.ts 2>&1 | tail -30
```

Expected: FAIL — the stub never throws, so the first test fails because it expects a rejection but gets `passed`.

**Step 3: Implement the fix**

Modify `src/core/step-executor.ts`:

Add import at the top of the file (after existing imports):
```typescript
import { runAssertion, type BaseAssertion } from '../assertions/assertion-runner'
```

Replace the `validateAssertions` method body (lines 240-250) with:
```typescript
private validateAssertions(assertions: unknown, data: unknown, stepId: string): void {
    this.logger.debug(`Step ${stepId} validating assertions against data`)

    if (!assertions || typeof assertions !== 'object') {
        return
    }

    // Support both single assertion object and array of assertions
    const items: BaseAssertion[] = Array.isArray(assertions)
        ? (assertions as BaseAssertion[])
        : [assertions as BaseAssertion]

    for (const assertion of items) {
        runAssertion(data, assertion)
    }
}
```

**Step 4: Run test to verify it passes**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/core/step-executor.test.ts 2>&1 | tail -20
```

Expected: PASS — both tests pass.

**Step 5: Commit**

```bash
cd /tmp/e2e-runner && git add src/core/step-executor.ts tests/unit/core/step-executor.test.ts && git commit -m "fix(core): wire assertion engine in StepExecutor.validateAssertions

Replace no-op stub with call to runAssertion() from assertion-runner.
Tests with a failing assert block now correctly throw AssertionError.

Closes Task 0.1"
```
