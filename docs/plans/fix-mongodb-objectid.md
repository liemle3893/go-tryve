# Fix MongoDB ObjectId Import — Implementation Plan

**Goal:** Import `ObjectId` once at module level instead of dynamically importing it on every MongoDB operation, improving performance.

**Architecture:** Move `ObjectId` import from inside the `normalizeFilter` method to the top of the file. This reduces overhead from repeated dynamic imports.

**Tech Stack:** TypeScript, mongodb (existing dependency), vitest (testing).

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/adapters/mongodb.adapter.ts` — MongoDB adapter; dynamically imports `ObjectId` in `normalizeFilter` method (line ~219)
- **Assumptions:** MongoDB adapter compiles and basic operations work. The dynamic import is a performance optimization issue, not a functional bug.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Task 0.7)

## Constraints

- Must not break existing MongoDB functionality
- Must work with mongodb package's ES module exports
- Performance improvement should be measurable in high-throughput scenarios

## Rollback

```bash
git revert HEAD~1  # Reverts the single commit for this fix
```

---

## Task 1: Move ObjectId import to module level

**Files:**
- Modify: `src/adapters/mongodb.adapter.ts` — Import `ObjectId` at top of file, remove dynamic import

### Step 1: Write failing test

Not applicable — this is a performance optimization with no functional change. Existing tests will verify behavior.

### Step 2: Run test to verify it fails

Not applicable — performance optimization only.

### Step 3: Implement

In `src/adapters/mongodb.adapter.ts`, find the top of the file (imports section):

```typescript
import type { Collection, Document, MongoClient, MongoClientOptions } from 'mongodb';
```

Replace with:
```typescript
import { ObjectId } from 'mongodb';
import type { Collection, Document, MongoClient, MongoClientOptions } from 'mongodb';
```

Then find the `normalizeFilter` method (around line 217-219):

```typescript
  private async normalizeFilter(
    filter: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    const { ObjectId } = await import('mongodb');
    const normalized = { ...filter };
```

Remove the dynamic import line and change the method from `async` to sync:

```typescript
  private normalizeFilter(
    filter: Record<string, unknown>
  ): Record<string, unknown> {
    const normalized = { ...filter };
```

**Note:** Since we're removing the `await import()`, the method no longer needs to be async. All callers that `await` this method will still work (awaiting a non-promise just returns the value immediately).

### Step 4: Run test to verify it passes

Run: `npm test`
Expected: All existing tests PASS (no behavior change expected).

Run: `npm run build`
Expected: Build succeeds with no errors.

### Step 5: Commit

```bash
git add src/adapters/mongodb.adapter.ts
git commit -m "perf(mongodb): import ObjectId once at module level"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All existing tests PASS (no behavior change).

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds with no errors.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | MongoDB operations still work | Run existing MongoDB E2E tests. | All tests PASS |
| 2 | ObjectId conversion works | Create test with `_id` filter. Run test. | ObjectId correctly converted and query succeeds |
| 3 | Build succeeds | Run `npm run build`. | No compilation errors |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `src/adapters/mongodb.adapter.ts` | Import `ObjectId` at module level (1 line added), remove dynamic import (1 line removed), make `normalizeFilter` sync instead of async (2 lines changed) |
