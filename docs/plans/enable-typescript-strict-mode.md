# Enable TypeScript Strict Mode — Implementation Plan

**Goal:** Enable all TypeScript strict mode flags and fix resulting compilation errors across the codebase.

**Architecture:** Incrementally enable strict mode flags in tsconfig.json, fix compilation errors in each source file systematically.

**Tech Stack:** TypeScript (existing), vitest (existing).

**Status:** Ready for Backend Developer
**Task:** Task 1.1 from Phase 1 (P0 Critical)
**Dependencies:** Phase 0 complete

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `tsconfig.json` — TypeScript configuration, currently has `"strict": false`
  - All 45 TypeScript source files in `src/` — May have type errors when strict mode enabled
- **Assumptions:** Project builds successfully with current config. All tests pass.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 1, Task 1.1)

## Constraints

- Enable strict flags incrementally (one or few at a time)
- Fix errors in smallest logical units
- One commit per file or logical group of files fixed
- No changes to existing test behavior
- May require temporary type assertions or null checks

## Rollback

```bash
git revert HEAD~N  # Where N is number of commits for this task
```

---

## Task 1: Enable strict mode flags in tsconfig.json

**Files:**
- Modify: `tsconfig.json` — Enable strict flags

### Step 1: Write failing test

Not applicable — configuration change only.

### Step 2: Run build to identify errors

Run: `npm run build`
Expected: Build currently succeeds (strict mode disabled)

### Step 3: Implement

In `tsconfig.json`, find:
```json
{
  "compilerOptions": {
    "strict": false,
    "noImplicitAny": false,
    "noUnusedLocals": false,
    "noUnusedParameters": false,
    "noImplicitReturns": false,
```

Replace with:
```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true,
    "strictBindCallApply": true,
    "strictPropertyInitialization": true,
```

### Step 4: Run build to see errors

Run: `npm run build`
Expected: TypeScript compilation fails with strict mode errors (this is expected — developer will fix in subsequent tasks)

### Step 5: Commit

```bash
git add tsconfig.json
git commit -m "chore(typescript): enable all strict mode flags

Phase 1, Task 1.1: Enable strict mode

Note: Build will fail until type errors are fixed in subsequent commits"
```

---

## Task 2-N: Fix compilation errors in each source file

**Approach:** Run `npm run build`, identify errors, fix systematically by file or module.

**Pattern for each fix:**

### Step 1: Identify errors

Run: `npm run build 2>&1 | grep "error TS"`
Expected: List of TypeScript errors with file locations

### Step 2: Fix errors in one file or module

Common fixes needed:
- Add null checks (`if (x !== null)`, `x?.property`, `x!`)
- Add type annotations for implicit `any`
- Remove unused variables/parameters
- Add return statements for implicit returns

### Step 3: Verify build

Run: `npm run build`
Expected: Fewer errors than before

### Step 4: Verify tests still pass

Run: `npm test`
Expected: All existing tests pass

### Step 5: Commit

```bash
git add <files-fixed>
git commit -m "fix(types): fix strict mode errors in <module-name>

Phase 1, Task 1.1"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full build**

Run: `npm run build`
Expected: Build succeeds with zero TypeScript errors.

**Step 2: Run full test suite**

Run: `npm test`
Expected: All tests pass (no behavioral changes).

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Build succeeds | `npm run build` | Exit code 0, no errors |
| 2 | Tests pass | `npm test` | All tests pass |
| 3 | Type safety | Inspect code | No `any` types, proper null checks |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `tsconfig.json` | Enable strict mode flags |
| Modify | Multiple files in `src/` | Fix strict mode compilation errors |

**Estimated files to modify:** 45 TypeScript source files (exact count depends on errors found)
