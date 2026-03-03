# Implement Test Coverage Reporting — Implementation Plan

**Goal:** Configure vitest coverage reporting to track test coverage and establish baseline metrics.

**Architecture:** Enable vitest's built-in coverage provider (v8), add npm scripts for coverage generation, document baseline percentage in README with badge.

**Tech Stack:** vitest (existing), @vitest/coverage-v8 (existing dev dependency).

**Status:** Ready for Backend Developer
**Task:** Task 1.2 from Phase 1 (P0 Critical)
**Dependencies:** Phase 0 complete

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `vitest.config.ts` — Test configuration, currently no coverage setup
  - `package.json` — npm scripts, needs coverage commands
  - `README.md` — Project documentation, needs coverage badge
- **Assumptions:** Vitest installed and working. Tests pass with `npm test`.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 1, Task 1.2)

## Constraints

- Use vitest's built-in coverage provider (no external tools)
- No changes to existing test behavior
- Coverage thresholds are informational (not CI gates yet)
- Badge should show baseline percentage

## Rollback

```bash
git revert HEAD~3  # Reverts the 3 commits for this task
```

---

## Task 1: Configure vitest coverage

**Files:**
- Modify: `vitest.config.ts` — Add coverage configuration

### Step 1: Write failing test

Not applicable — configuration change only.

### Step 2: Run test to verify current state

Run: `npm test`
Expected: All tests pass, no coverage report generated.

### Step 3: Implement

In `vitest.config.ts`, find:
```typescript
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    include: ['tests/**/*.test.ts'],
  },
});
```

Replace with:
```typescript
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    include: ['tests/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'text-summary'],
      reportsDirectory: './coverage',
      include: ['src/**/*'],
      exclude: [
        'node_modules',
        'dist',
        'src/**/*.d.ts',
        'src/**/*.test.ts',
      ],
    },
  },
});
```

### Step 4: Verify coverage runs

Run: `npm test -- --coverage`
Expected: Coverage report generated in `./coverage/` directory with text and HTML output.

### Step 5: Commit

```bash
git add vitest.config.ts
git commit -m "feat(testing): configure vitest coverage reporting

Phase 1, Task 1.2

Add coverage provider v8 with text, JSON, and HTML reporters"
```

---

## Task 2: Add coverage scripts to package.json

**Files:**
- Modify: `package.json` — Add coverage scripts

### Step 1: Write failing test

Not applicable — npm scripts addition.

### Step 2: Verify current scripts

Run: `npm run | grep test`
Expected: Shows existing test script.

### Step 3: Implement

In `package.json`, find:
```json
  "scripts": {
    "build": "tsc",
    "clean": "rm -rf dist",
    "prepublishOnly": "npm run clean && npm run build",
    "local:install": "npm run build && npm install --global --prefix ~/.local .",
    "test": "vitest run",
```

Add after:
```json
    "test:coverage": "vitest run --coverage",
    "test:coverage:open": "vitest run --coverage && open coverage/index.html",
```

### Step 4: Verify new scripts

Run: `npm run test:coverage`
Expected: Tests run with coverage, report generated.

### Step 5: Commit

```bash
git add package.json
git commit -m "feat(testing): add test coverage scripts

Phase 1, Task 1.2

Add test:coverage and test:coverage:open scripts"
```

---

## Task 3: Document baseline coverage in README

**Files:**
- Modify: `README.md` — Add coverage badge

### Step 1: Run coverage to get baseline

Run: `npm run test:coverage`
Expected: Coverage report shows baseline percentage (likely ~15% based on roadmap).

### Step 2: Add coverage section to README

In `README.md`, find an appropriate location after the project description.

Insert:
```markdown
## Test Coverage

[![Coverage](https://img.shields.io/badge/coverage-15%25-yellow)](./coverage/)

**Baseline coverage:** ~15% (as of Phase 0 completion)

Run `npm run test:coverage` to generate coverage report.
```

Note: Update the percentage based on actual baseline from Step 1.

### Step 3: Verify badge renders

Open `README.md` in browser or markdown viewer.
Expected: Badge displays with coverage percentage.

### Step 4: Commit

```bash
git add README.md
git commit -m "docs: add test coverage badge with baseline percentage

Phase 1, Task 1.2

Baseline coverage: ~15% (based on current test suite)"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run coverage report**

Run: `npm run test:coverage`
Expected: All tests pass, coverage report generated in `./coverage/` directory.

**Step 2: Verify coverage files exist**

Run: `ls -la coverage/`
Expected: `coverage-final.json`, `index.html`, and other report files exist.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Coverage runs | `npm run test:coverage` | Report generated successfully |
| 2 | Badge displays | Open README.md | Badge shows ~15% coverage |
| 3 | HTML report works | Open `coverage/index.html` | Interactive coverage report displays |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `vitest.config.ts` | Add coverage configuration (provider, reporter, include/exclude) |
| Modify | `package.json` | Add coverage scripts (test:coverage, test:coverage:open) |
| Modify | `README.md` | Add coverage badge and baseline documentation (5 lines) |
