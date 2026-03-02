# Fix Redis KEYS Command — Implementation Plan

**Goal:** `flushPattern` action uses non-blocking `SCAN` command instead of blocking `KEYS` command, preventing Redis performance degradation in production.

**Architecture:** RedisAdapter's `flushPattern` method currently uses `KEYS pattern` which blocks all other Redis operations. Replace with `SCAN` cursor-based iteration that yields control to other operations.

**Tech Stack:** TypeScript, vitest (testing), ioredis (existing).

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/adapters/redis.adapter.ts` — Redis adapter with `flushPattern` method at line 201-207; uses blocking `KEYS` command
- **Assumptions:** ioredis supports `SCAN` command with cursor-based iteration. RedisAdapter exists and compiles.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Task 0.6)

## Constraints

- Must delete ALL keys matching pattern (same behavior as `KEYS`)
- Must not block Redis for extended periods
- Must handle large keysets (1000+ keys) efficiently
- Must maintain existing error handling
- Cannot test against real Redis without infrastructure — unit test with mock

## Rollback

```bash
git revert HEAD~1  # Reverts the single commit for this fix
```

---

## Task 1: Replace KEYS with SCAN loop in flushPattern

**Files:**
- Modify: `src/adapters/redis.adapter.ts` — Replace `KEYS` + `del` with `SCAN` loop
- Create: `tests/unit/adapters/redis-scan.test.ts` — Test SCAN-based flushPattern

### Step 1: Write failing test

Create `tests/unit/adapters/redis-scan.test.ts`:
```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { RedisAdapter } from '../../../src/adapters/redis.adapter';
import type { AdapterContext, Logger } from '../../../src/types';

describe('RedisAdapter - SCAN-based flushPattern', () => {
  let adapter: RedisAdapter;
  let mockRedis: any;
  let mockLogger: Logger;
  let context: AdapterContext;

  beforeEach(async () => {
    mockLogger = {
      debug: () => {},
      info: () => {},
      warn: () => {},
      error: () => {},
    };

    context = {
      variables: {},
      captured: {},
      capture: () => {},
      logger: mockLogger,
    };

    // Create mock Redis client with SCAN support
    mockRedis = {
      keys: vi.fn(), // Should NOT be called
      scan: vi.fn(),
      del: vi.fn(),
      ping: vi.fn(() => Promise.resolve('PONG')),
      connect: vi.fn(() => Promise.resolve()),
      quit: vi.fn(() => Promise.resolve()),
    };

    adapter = new RedisAdapter({ connectionString: 'redis://localhost:6379' }, mockLogger);

    // Inject mock client
    (adapter as any).client = mockRedis;
    (adapter as any).connected = true;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should use SCAN instead of KEYS for flushPattern', async () => {
    // Mock SCAN to return keys in 2 batches, then end
    mockRedis.scan
      .mockResolvedValueOnce(['10', ['test:key1', 'test:key2']]) // Cursor 10, 2 keys
      .mockResolvedValueOnce(['20', ['test:key3']]) // Cursor 20, 1 key
      .mockResolvedValueOnce(['0', []]); // Cursor 0 = done

    mockRedis.del.mockResolvedValue(3);

    const result = await adapter.execute('flushPattern', { pattern: 'test:*' }, context);

    // KEYS should NOT be called
    expect(mockRedis.keys).not.toHaveBeenCalled();

    // SCAN should be called 3 times (until cursor returns 0)
    expect(mockRedis.scan).toHaveBeenCalledTimes(3);

    // First SCAN call should start with cursor 0
    expect(mockRedis.scan).toHaveBeenNthCalledWith(1, 0, 'MATCH', 'test:*', 'COUNT', 100);

    // DEL should be called with all collected keys
    expect(mockRedis.del).toHaveBeenCalledWith('test:key1', 'test:key2', 'test:key3');

    // Result should show 3 keys deleted
    expect(result.success).toBe(true);
    expect((result.data as any).result).toBe(3);
  });

  it('should return 0 when no keys match pattern', async () => {
    mockRedis.scan.mockResolvedValue(['0', []]); // Cursor 0 = done immediately, no keys

    const result = await adapter.execute('flushPattern', { pattern: 'nomatch:*' }, context);

    expect(mockRedis.keys).not.toHaveBeenCalled();
    expect(mockRedis.del).not.toHaveBeenCalled();
    expect(result.success).toBe(true);
    expect((result.data as any).result).toBe(0);
  });

  it('should handle large keysets efficiently', async () => {
    // Simulate 500 keys across 5 SCAN iterations
    const keys1 = Array.from({ length: 100 }, (_, i) => `large:key${i}`);
    const keys2 = Array.from({ length: 100 }, (_, i) => `large:key${i + 100}`);
    const keys3 = Array.from({ length: 100 }, (_, i) => `large:key${i + 200}`);
    const keys4 = Array.from({ length: 100 }, (_, i) => `large:key${i + 300}`);
    const keys5 = Array.from({ length: 100 }, (_, i) => `large:key${i + 400}`);

    mockRedis.scan
      .mockResolvedValueOnce(['100', keys1])
      .mockResolvedValueOnce(['200', keys2])
      .mockResolvedValueOnce(['300', keys3])
      .mockResolvedValueOnce(['400', keys4])
      .mockResolvedValueOnce(['0', keys5]);

    mockRedis.del.mockResolvedValue(500);

    const result = await adapter.execute('flushPattern', { pattern: 'large:*' }, context);

    expect(mockRedis.scan).toHaveBeenCalledTimes(5);
    expect(mockRedis.del).toHaveBeenCalledTimes(1);
    expect((result.data as any).result).toBe(500);
  });
});
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/adapters/redis-scan.test.ts`
Expected: `FAIL — mockRedis.keys should not be called` (current implementation uses KEYS)

### Step 3: Implement

In `src/adapters/redis.adapter.ts`, find the `flushPattern` method (lines 201-207):

```typescript
  private async flushPattern(pattern: string): Promise<number> {
    const keys = await this.client!.keys(pattern);
    if (keys.length === 0) {
      return 0;
    }
    return await this.client!.del(...keys);
  }
```

Replace with:
```typescript
  private async flushPattern(pattern: string): Promise<number> {
    const allKeys: string[] = [];
    let cursor = '0';

    // Use SCAN to iterate through keys matching pattern
    // This is non-blocking and won't degrade Redis performance
    do {
      const [nextCursor, keys] = await this.client!.scan(
        cursor,
        'MATCH',
        pattern,
        'COUNT',
        100
      );
      cursor = nextCursor;
      allKeys.push(...keys);
    } while (cursor !== '0');

    if (allKeys.length === 0) {
      return 0;
    }

    // Delete all collected keys
    return await this.client!.del(...allKeys);
  }
```

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/adapters/redis-scan.test.ts`
Expected: 3 tests PASS

### Step 5: Commit

```bash
git add src/adapters/redis.adapter.ts tests/unit/adapters/redis-scan.test.ts
git commit -m "perf(redis): replace KEYS with SCAN in flushPattern"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All existing tests PASS + 3 new Redis SCAN tests PASS.

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds with no errors.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | flushPattern deletes matching keys | Cannot test without real Redis | Unit test validates behavior |
| 2 | flushPattern returns 0 for no matches | Cannot test without real Redis | Unit test validates behavior |
| 3 | No KEYS command issued | Cannot test without real Redis monitoring | Unit test validates KEYS not called |
| 4 | Existing Redis E2E tests pass | Run existing Redis E2E tests if infrastructure available | All E2E tests PASS |

**Note:** Full verification requires Redis infrastructure. Unit tests validate the fix with mocked client. Integration testing deferred until infrastructure available.

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `src/adapters/redis.adapter.ts` | Replace `KEYS` with `SCAN` loop in `flushPattern` (replace 6 lines with 17 lines) |
| Create | `tests/unit/adapters/redis-scan.test.ts` | Test SCAN-based flushPattern (120 lines) |
