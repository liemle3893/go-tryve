# Fix Redis KEYS Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the blocking `KEYS` command in `RedisAdapter.flushPattern` with a non-blocking `SCAN` cursor loop so the adapter is safe to use against production-sized Redis instances.

**Architecture:** `flushPattern` currently calls `this.client!.keys(pattern)` which is O(N) and blocks the Redis server during execution. The fix replaces it with a `SCAN` loop: iterate with `SCAN cursor MATCH pattern COUNT 100` until the cursor wraps back to `'0'`, collecting and deleting keys in batches. The public interface (`flushPattern(pattern)`) is unchanged.

**Tech Stack:** TypeScript, Vitest, `src/adapters/redis.adapter.ts` (existing), ioredis

---

### Task 1: Write the failing unit test

**Files:**
- Create: `tests/unit/adapters/redis.test.ts`

**Step 1: Write the failing test**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('RedisAdapter.flushPattern uses SCAN not KEYS', () => {
  it('calls scan() instead of keys() when flushing by pattern', async () => {
    const mockScan = vi.fn()
      .mockResolvedValueOnce(['5', ['key:1', 'key:2']])  // cursor 5, 2 keys
      .mockResolvedValueOnce(['0', ['key:3']])             // cursor 0, 1 key (done)
    const mockDel = vi.fn().mockResolvedValue(1)
    const mockKeys = vi.fn()  // should NOT be called

    const fakeClient = {
      scan: mockScan,
      del: mockDel,
      keys: mockKeys,
      ping: vi.fn().mockResolvedValue('PONG'),
      quit: vi.fn().mockResolvedValue(undefined),
      connect: vi.fn().mockResolvedValue(undefined),
    }

    vi.doMock('ioredis', () => ({
      default: vi.fn(() => fakeClient),
    }))

    const { RedisAdapter } = await import('../../../src/adapters/redis.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new RedisAdapter({ connectionString: 'redis://localhost:6379' }, logger)

    // Bypass connect() — inject fake client directly
    ;(adapter as any).client = fakeClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    await adapter.execute('flushPattern', { pattern: 'key:*' }, ctx)

    // SCAN must have been called; KEYS must NOT have been called
    expect(mockScan).toHaveBeenCalled()
    expect(mockKeys).not.toHaveBeenCalled()

    // All 3 keys should have been deleted across two batches
    const deletedKeys = mockDel.mock.calls.flatMap((call) => call)
    expect(deletedKeys).toEqual(expect.arrayContaining(['key:1', 'key:2', 'key:3']))

    vi.doUnmock('ioredis')
  })

  it('returns 0 and makes no del calls when no keys match the pattern', async () => {
    const mockScan = vi.fn().mockResolvedValue(['0', []])  // empty, done immediately
    const mockDel = vi.fn()

    const fakeClient = {
      scan: mockScan,
      del: mockDel,
      keys: vi.fn(),
      ping: vi.fn().mockResolvedValue('PONG'),
    }

    vi.doMock('ioredis', () => ({
      default: vi.fn(() => fakeClient),
    }))

    const { RedisAdapter } = await import('../../../src/adapters/redis.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new RedisAdapter({ connectionString: 'redis://localhost:6379' }, logger)
    ;(adapter as any).client = fakeClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    const result = await adapter.execute('flushPattern', { pattern: 'no-match:*' }, ctx)

    expect(mockDel).not.toHaveBeenCalled()
    expect(result.success).toBe(true)

    vi.doUnmock('ioredis')
  })
})
```

**Step 2: Run test to verify it fails**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/adapters/redis.test.ts 2>&1 | tail -30
```

Expected: FAIL — `flushPattern` calls `keys()` not `scan()`, so `mockKeys` gets called and the scan-based assertion fails.

**Step 3: Implement the fix**

Modify `src/adapters/redis.adapter.ts`.

Replace the `flushPattern` method (lines 201-207):

```typescript
// BEFORE:
private async flushPattern(pattern: string): Promise<number> {
  const keys = await this.client!.keys(pattern);
  if (keys.length === 0) {
    return 0;
  }
  return await this.client!.del(...keys);
}
```

```typescript
// AFTER:
private async flushPattern(pattern: string): Promise<number> {
  let cursor = '0';
  let totalDeleted = 0;

  do {
    const [nextCursor, keys] = await this.client!.scan(cursor, 'MATCH', pattern, 'COUNT', 100);
    cursor = nextCursor;
    if (keys.length > 0) {
      totalDeleted += await this.client!.del(...keys);
    }
  } while (cursor !== '0');

  return totalDeleted;
}
```

**Step 4: Run test to verify it passes**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/adapters/redis.test.ts 2>&1 | tail -20
```

Expected: PASS — both tests pass.

**Step 5: Commit**

```bash
cd /tmp/e2e-runner && git add src/adapters/redis.adapter.ts tests/unit/adapters/redis.test.ts && git commit -m "fix(adapters): replace blocking KEYS with SCAN loop in flushPattern

The KEYS command blocks the Redis server for O(N) on large keyspaces.
Replace with a SCAN cursor loop that processes keys in batches of 100,
safe for production use.

Closes Task 0.6"
```
