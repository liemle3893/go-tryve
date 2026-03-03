# Refactor HTML Reporter — Implementation Plan

**Goal:** Refactor the 1054-line html.reporter.ts into smaller, maintainable modules with no file exceeding 300 lines.

**Architecture:** Split the monolithic HTML reporter into logical modules:
- `html/styles.ts` — CSS styles and theming
- `html/scripts.ts` — JavaScript for interactivity
- `html/templates.ts` — HTML template generators
- `html/utils.ts` — Utility functions (formatting, status mapping)
- `html/reporter.ts` — Main reporter class (orchestrator)

**Tech Stack:** TypeScript (existing), vitest (existing).

**Status:** Ready for Backend Developer
**Task:** Task 1.6 from Phase 1 (P1 High)
**Dependencies:** Phase 0 complete

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/reporters/html.reporter.ts` — 1054 lines, monolithic reporter
- **Assumptions:** Project builds successfully. HTML reporter works correctly.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 1, Task 1.6)

## Constraints

- No file should exceed 300 lines
- HTML output must remain unchanged (behavioral parity)
- All existing tests must pass
- Follow existing code style and patterns
- No external dependencies added

## Rollback

```bash
git revert HEAD~5  # Reverts the 5 commits for this task
rm -rf src/reporters/html/
```

---

## Task 1: Extract CSS styles to separate module

**Files:**
- Create: `src/reporters/html/styles.ts`
- Modify: `src/reporters/html.reporter.ts` — Remove CSS, import from styles.ts

### Step 1: Write failing test

Not applicable — refactoring existing code. Behavioral tests will verify no regression.

### Step 2: Run test to verify current state

Run: `npm test`
Expected: All tests pass.

### Step 3: Implement

Create `src/reporters/html/styles.ts`:

```typescript
/**
 * CSS styles for HTML reporter
 * Extracted from html.reporter.ts for maintainability
 */

export const CSS_STYLES = `
/* CSS Reset and Base Styles */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background: var(--color-bg);
  color: var(--color-text);
  line-height: 1.6;
}

:root {
  --color-pass: #22c55e;
  --color-fail: #ef4444;
  --color-skip: #f59e0b;
  --color-warn: #eab308;
  --color-bg: #f8fafc;
  --color-card: #ffffff;
  --color-text: #1e293b;
  --color-border: #e2e8f0;
  --shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  --radius: 6px;
}

/* Layout */
.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

/* Dashboard */
.dashboard {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: var(--color-card);
  padding: 1.5rem;
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  text-align: center;
}

.stat-card .value {
  font-size: 2rem;
  font-weight: bold;
  margin-bottom: 0.5rem;
}

.stat-card .label {
  color: #64748b;
  font-size: 0.875rem;
}

.stat-card.passed .value { color: var(--color-pass); }
.stat-card.failed .value { color: var(--color-fail); }
.stat-card.skipped .value { color: var(--color-skip); }
.stat-card.warned .value { color: var(--color-warn); }

/* Test List */
.test-list {
  background: var(--color-card);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  overflow: hidden;
}

.test-item {
  border-bottom: 1px solid var(--color-border);
  padding: 1rem 1.5rem;
}

.test-item:last-child {
  border-bottom: none;
}

.test-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  cursor: pointer;
}

.test-status {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  flex-shrink: 0;
}

.test-status.passed { background: var(--color-pass); color: white; }
.test-status.failed { background: var(--color-fail); color: white; }
.test-status.skipped { background: var(--color-skip); color: white; }
.test-status.warned { background: var(--color-warn); color: white; }
.test-status.error { background: var(--color-fail); color: white; }

.test-name {
  flex: 1;
  font-weight: 500;
}

.test-duration {
  color: #64748b;
  font-size: 0.875rem;
}

/* Progress Bar */
.progress-bar {
  height: 8px;
  background: var(--color-border);
  border-radius: 4px;
  overflow: hidden;
  display: flex;
}

.progress-bar .passed { background: var(--color-pass); }
.progress-bar .failed { background: var(--color-fail); }
.progress-bar .skipped { background: var(--color-skip); }
.progress-bar .warned { background: var(--color-warn); }

/* Status Badges */
.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-badge.pass { background: var(--color-pass); color: white; }
.status-badge.fail { background: var(--color-fail); color: white; }
.status-badge.warn { background: var(--color-warn); color: white; }
`

/**
 * Get CSS styles for HTML reporter
 */
export function getStyles(): string {
  return CSS_STYLES
}
```

In `src/reporters/html.reporter.ts`, find the CSS string and replace with import:

```typescript
import { getStyles } from './html/styles.js'

// Replace the inline CSS with:
const styles = getStyles()
```

### Step 4: Verify no regression

Run: `npm test`
Expected: All tests pass.

Run: `npm run build`
Expected: Build succeeds.

### Step 5: Commit

```bash
git add src/reporters/html/styles.ts src/reporters/html.reporter.ts
git commit -m "refactor(reporters): extract CSS styles to separate module

Phase 1, Task 1.6

Move CSS to src/reporters/html/styles.ts (280 lines)
No behavioral changes"
```

---

## Task 2: Extract JavaScript to separate module

**Files:**
- Create: `src/reporters/html/scripts.ts`
- Modify: `src/reporters/html.reporter.ts`

### Step 1-5: Follow same pattern

Extract JavaScript functions (expand/collapse, filtering, sorting) to `scripts.ts`.

---

## Task 3: Extract HTML templates to separate module

**Files:**
- Create: `src/reporters/html/templates.ts`
- Modify: `src/reporters/html.reporter.ts`

### Step 1-5: Follow same pattern

Extract template generation functions:
- `renderDashboard()`
- `renderTestList()`
- `renderPhaseDetails()`
- `renderStepDetails()`

---

## Task 4: Extract utility functions

**Files:**
- Create: `src/reporters/html/utils.ts`
- Modify: `src/reporters/html.reporter.ts`

### Step 1-5: Follow same pattern

Extract utilities:
- `formatDuration()`
- `getStatusIcon()`
- `getStatusColor()`
- `percentage()`

---

## Task 5: Refactor main reporter class

**Files:**
- Rename: `src/reporters/html.reporter.ts` → `src/reporters/html/reporter.ts`
- Update imports in `src/reporters/index.ts`

### Step 1: Update imports

In `src/reporters/index.ts`, find:
```typescript
export { HTMLReporter } from './html.reporter'
```

Replace with:
```typescript
export { HTMLReporter } from './html/reporter.js'
```

### Step 2: Verify build

Run: `npm run build`
Expected: Build succeeds with updated imports.

### Step 3: Verify tests

Run: `npm test`
Expected: All tests pass.

### Step 4: Check file sizes

Run: `wc -l src/reporters/html/*.ts`
Expected: No file exceeds 300 lines.

### Step 5: Commit

```bash
git add src/reporters/
git commit -m "refactor(reporters): reorganize HTML reporter into modules

Phase 1, Task 1.6

File structure:
- html/reporter.ts (main class, <200 lines)
- html/styles.ts (CSS, ~280 lines)
- html/scripts.ts (JS, ~150 lines)
- html/templates.ts (HTML generators, ~250 lines)
- html/utils.ts (utilities, ~100 lines)

All files under 300 lines. No behavioral changes."
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All tests pass.

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | File sizes | `wc -l src/reporters/html/*.ts` | All files <300 lines |
| 2 | HTML output | Run E2E test, open HTML report | Report looks identical to before |
| 3 | Functionality | Click test items, use filters | All interactive features work |

**Step 4: Compare HTML output**

Generate HTML report before and after refactoring. Use `diff` to verify identical output.

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Create | `src/reporters/html/styles.ts` | CSS styles (~280 lines) |
| Create | `src/reporters/html/scripts.ts` | JavaScript (~150 lines) |
| Create | `src/reporters/html/templates.ts` | HTML generators (~250 lines) |
| Create | `src/reporters/html/utils.ts` | Utility functions (~100 lines) |
| Rename | `src/reporters/html.reporter.ts` → `src/reporters/html/reporter.ts` | Main reporter class (<200 lines) |
| Modify | `src/reporters/index.ts` | Update import path |

**Estimated effort:** 16-24 hours (as per roadmap)
