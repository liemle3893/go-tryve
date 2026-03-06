# Fix MongoDB ObjectId Import Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Import `ObjectId` from the `mongodb` package once at connect time rather than re-importing it on every `normalizeFilter` call.

**Architecture:** `MongoDBAdapter.normalizeFilter` is `async` only because it does `const { ObjectId } = await import('mongodb')` on every invocation. Moving the import into `connect()` and storing `ObjectId` as a class property lets `normalizeFilter` become synchronous and eliminates repeated dynamic imports. The public interface of the adapter is unchanged.

**Tech Stack:** TypeScript, Vitest, `src/adapters/mongodb.adapter.ts` (existing), `mongodb` driver

---

### Task 1: Write the failing unit test

**Files:**
- Create: `tests/unit/adapters/mongodb.test.ts`

**Step 1: Write the failing test**

```typescript
import { describe, it, expect, vi } from 'vitest'

describe('MongoDBAdapter ObjectId import', () => {
  it('imports mongodb module once (at connect time), not once per normalizeFilter call', async () => {
    // Track how many times the mongodb module is dynamically imported
    let importCount = 0
    const RealObjectId = (await import('mongodb')).ObjectId

    vi.doMock('mongodb', () => {
      importCount++
      return {
        MongoClient: vi.fn(() => ({
          connect: vi.fn().mockResolvedValue(undefined),
          close: vi.fn().mockResolvedValue(undefined),
          db: vi.fn(() => ({
            command: vi.fn().mockResolvedValue({ ok: 1 }),
            collection: vi.fn(() => ({
              findOne: vi.fn().mockResolvedValue({ _id: 'abc', name: 'test' }),
            })),
          })),
        })),
        ObjectId: RealObjectId,
      }
    })

    const { MongoDBAdapter } = await import('../../../src/adapters/mongodb.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new MongoDBAdapter(
      { connectionString: 'mongodb://localhost:27017', database: 'test' },
      logger
    )
    await adapter.connect()
    const importCountAfterConnect = importCount

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    // Execute findOne three times — each currently re-imports mongodb
    await adapter.execute('findOne', { collection: 'users', filter: { name: 'Alice' } }, ctx)
    await adapter.execute('findOne', { collection: 'users', filter: { name: 'Bob' } }, ctx)
    await adapter.execute('findOne', { collection: 'users', filter: { name: 'Carol' } }, ctx)

    // After the fix, importCount should not increase beyond connect time
    expect(importCount).toBe(importCountAfterConnect)

    vi.doUnmock('mongodb')
  })
})
```

**Step 2: Run test to verify it fails**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/adapters/mongodb.test.ts 2>&1 | tail -30
```

Expected: FAIL — `importCount` increases with each `findOne` call because `normalizeFilter` currently re-imports `mongodb` every time.

**Step 3: Implement the fix**

Modify `src/adapters/mongodb.adapter.ts`:

**3a. Add a class property to hold `ObjectId`** (after line 25, inside the class declaration):

```typescript
export class MongoDBAdapter extends BaseAdapter {
  private client: import('mongodb').MongoClient | null = null;
  private db: import('mongodb').Db | null = null;
  private ObjectId: typeof import('mongodb').ObjectId | null = null;  // ADD THIS
```

**3b. Update `connect()` to capture `ObjectId`** — replace the import line inside `connect()`:

```typescript
// BEFORE:
const { MongoClient } = await import('mongodb');

// AFTER:
const { MongoClient, ObjectId } = await import('mongodb');
this.ObjectId = ObjectId;
```

**3c. Make `normalizeFilter` synchronous and use `this.ObjectId`**:

```typescript
// BEFORE:
private async normalizeFilter(
  filter: Record<string, unknown>
): Promise<Record<string, unknown>> {
  const { ObjectId } = await import('mongodb');
  const normalized = { ...filter };
  // ...
}
```

```typescript
// AFTER:
private normalizeFilter(
  filter: Record<string, unknown>
): Record<string, unknown> {
  const ObjectId = this.ObjectId!;
  const normalized = { ...filter };

  // Handle _id field
  if (typeof normalized._id === 'string' && ObjectId.isValid(normalized._id)) {
    normalized._id = new ObjectId(normalized._id);
  }

  // Handle $oid notation (from YAML)
  if (normalized._id && typeof normalized._id === 'object') {
    const idObj = normalized._id as Record<string, unknown>;
    if ('$oid' in idObj && typeof idObj.$oid === 'string') {
      normalized._id = new ObjectId(idObj.$oid);
    }
  }

  return normalized;
}
```

**3d. Update callers of `normalizeFilter` in the `execute` method** — remove all `await` keywords before `this.normalizeFilter(...)` calls (there are 6 of them in the switch statement):

```typescript
// BEFORE (example):
const filter = await this.normalizeFilter(params.filter as Record<string, unknown>);

// AFTER (example):
const filter = this.normalizeFilter(params.filter as Record<string, unknown>);
```

Apply this change to all 6 `case` blocks: `findOne`, `find`, `updateOne`, `updateMany`, `deleteOne`, `deleteMany`.

**Step 4: Run test to verify it passes**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/adapters/mongodb.test.ts 2>&1 | tail -20
```

Expected: PASS.

**Step 5: Commit**

```bash
cd /tmp/e2e-runner && git add src/adapters/mongodb.adapter.ts tests/unit/adapters/mongodb.test.ts && git commit -m "fix(adapters): import ObjectId once at connect time in MongoDBAdapter

normalizeFilter previously did await import('mongodb') on every call,
re-importing the module for each findOne/find/updateOne/etc operation.
Now ObjectId is captured during connect() and stored as a class field,
making normalizeFilter synchronous and eliminating repeated imports.

Closes Task 0.7"
```
