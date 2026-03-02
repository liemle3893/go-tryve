# Add TypeScript Adapter Type — Implementation Plan

**Goal:** TypeScript function-backed steps declare `adapter: typescript` instead of using the misleading `http` adapter type.

**Architecture:** Add `typescript` to the `AdapterType` union. Update YAML loader and test orchestrator to handle the new adapter type. Existing TypeScript function steps will work without changes since they use a special action identifier.

**Tech Stack:** TypeScript, vitest (testing).

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/types.ts` — Defines `AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell'` (line 84)
  - `src/core/yaml-loader.ts` — Loads YAML test files, needs to accept `typescript` adapter
  - `src/core/step-executor.ts` — Already handles TypeScript functions via `TYPESCRIPT_FUNCTION_ACTION` constant
  - `src/core/test-orchestrator.ts` — Orchestrates test execution, may need adapter type handling
  - `tests/e2e/` — Existing E2E tests that use TypeScript functions
- **Assumptions:** TypeScript function steps currently work with `adapter: http` as a workaround. The function execution logic in StepExecutor is correct and won't change.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Task 0.5)

## Constraints

- Must not break existing TypeScript function tests
- Must not break existing YAML tests using other adapters
- The `typescript` adapter type is for documentation/type safety only — no actual adapter implementation needed

## Rollback

```bash
git revert HEAD~1  # Reverts the single commit for this fix
```

---

## Task 1: Add typescript to AdapterType union

**Files:**
- Modify: `src/types.ts` — Add `'typescript'` to `AdapterType` union

### Step 1: Write failing test

Not applicable — type definition only.

### Step 2: Run test to verify it fails

Not applicable — type definition only.

### Step 3: Implement

In `src/types.ts`, find (line 84):
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell';
```

Replace with:
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell' | 'typescript';
```

### Step 4: Run test to verify it passes

Run: `npm run build`
Expected: Build succeeds with no errors.

### Step 5: Commit

```bash
git add src/types.ts
git commit -m "feat(types): add 'typescript' to AdapterType union"
```

---

## Task 2: Update YAML loader to accept typescript adapter

**Files:**
- Modify: `src/core/yaml-loader.ts` — Add validation/acceptance for `typescript` adapter

### Step 1: Write failing test

Create `tests/unit/core/yaml-loader-typescript.test.ts`:
```typescript
import { describe, it, expect } from 'vitest';
import { loadTestFromYaml } from '../../../src/core/yaml-loader';

describe('YAML Loader - TypeScript Adapter', () => {
  it('should accept adapter: typescript', () => {
    const yamlContent = `
name: Test with TypeScript function
execute:
  - id: custom-function
    adapter: typescript
    action: run
    params:
      code: "async (ctx) => { return { success: true }; }"
`;

    const result = loadTestFromYaml(yamlContent);
    expect(result).toBeDefined();
    expect(result.execute).toBeDefined();
    expect(result.execute![0].adapter).toBe('typescript');
  });

  it('should work with TypeScript function steps', () => {
    const yamlContent = `
name: TypeScript function test
execute:
  - id: typescript-step
    adapter: typescript
    action: customAction
    params:
      __function: "placeholder"
`;

    const result = loadTestFromYaml(yamlContent);
    expect(result.execute![0].adapter).toBe('typescript');
  });
});
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/core/yaml-loader-typescript.test.ts`
Expected: `FAIL — Type '"typescript"' is not assignable to type 'AdapterType'` (if validation exists) or tests may pass if no validation

### Step 3: Implement

In `src/core/yaml-loader.ts`, find any adapter type validation logic. If it validates against a hardcoded list, add `'typescript'` to the allowed values.

If no explicit validation exists (the type system handles it), no changes needed here.

Search for patterns like:
```typescript
const validAdapters = ['postgresql', 'redis', 'mongodb', 'eventhub', 'http', 'shell'];
```

And add `'typescript'` to the array.

If no such validation exists, this step requires no code changes.

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/core/yaml-loader-typescript.test.ts`
Expected: 2 tests PASS

### Step 5: Commit

```bash
git add src/core/yaml-loader.ts tests/unit/core/yaml-loader-typescript.test.ts
git commit -m "feat(yaml-loader): support typescript adapter type"
```

---

## Task 3: Update documentation and examples

**Files:**
- Modify: `docs/03-yaml-tests.md` — Add example with `adapter: typescript`
- Modify: `tests/e2e/` — Update any TypeScript function tests to use `adapter: typescript`

### Step 1: Write failing test

Not applicable — documentation update.

### Step 2: Run test to verify it fails

Not applicable — documentation update.

### Step 3: Implement

In `docs/03-yaml-tests.md`, find the section about TypeScript function steps. Update the example to use `adapter: typescript`:

Find:
```yaml
adapter: http
```

In TypeScript function examples, replace with:
```yaml
adapter: typescript
```

Search for any TypeScript test files in `tests/e2e/` that use function-backed steps with `adapter: http` and update them to `adapter: typescript`.

### Step 4: Run test to verify it passes

Run: `npm test`
Expected: All tests PASS (no behavior change expected).

### Step 5: Commit

```bash
git add docs/03-yaml-tests.md tests/e2e/
git commit -m "docs: update examples to use typescript adapter type"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All existing tests PASS + new YAML loader tests PASS.

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds with no errors.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | TypeScript adapter accepted in YAML | Create test file with `adapter: typescript`. Run `e2e validate`. | Validation passes with no errors |
| 2 | TypeScript function tests still work | Run existing TypeScript function E2E tests. | Tests execute successfully |
| 3 | Documentation is correct | Read `docs/03-yaml-tests.md`. | Example shows `adapter: typescript` |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `src/types.ts` | Add `'typescript'` to `AdapterType` union (1 word) |
| Modify | `src/core/yaml-loader.ts` | Accept `typescript` adapter (1 line if validation exists) |
| Create | `tests/unit/core/yaml-loader-typescript.test.ts` | Test typescript adapter loading (40 lines) |
| Modify | `docs/03-yaml-tests.md` | Update examples (1-3 lines) |
| Modify | `tests/e2e/*.test.yaml` | Update TypeScript tests (if any use wrong adapter) |
