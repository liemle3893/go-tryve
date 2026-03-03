# Fix continueOnError Status — Implementation Plan

**Goal:** Steps that fail with `continueOnError: true` display status `warned` instead of `passed`, making it clear to users that the step encountered an error.

**Architecture:** StepExecutor catches errors and checks `continueOnError` flag. When true, returns new `warned` status instead of `passed`. Reporters (console, HTML) display `warned` status with yellow color and warning symbol.

**Tech Stack:** TypeScript, vitest (testing).

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/core/step-executor.ts` — Executes test steps with retry logic; currently returns `'passed'` status when `continueOnError: true` (line ~215)
  - `src/types.ts` — Defines `StepStatus = 'passed' | 'failed' | 'skipped'` (line ~84); needs `'warned'` added
  - `src/reporters/console.reporter.ts` — Displays step status with symbols and colors; needs `warned` case added
  - `src/reporters/html.reporter.ts` — HTML report with status styling; needs `warned` CSS and logic
- **Assumptions:** StepExecutor, types, and reporters exist and compile. No existing tests rely on `continueOnError` behavior.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Task 0.2)

## Constraints

- No breaking changes to existing YAML test files
- New status must be clearly distinguishable from `passed` and `failed`
- All reporters must display `warned` status consistently
- Must maintain backward compatibility with test result JSON schema

## Rollback

```bash
git revert HEAD~1  # Reverts the single commit for this fix
```

---

## Task 1: Add warned status to StepStatus type

**Files:**
- Modify: `src/types.ts` — Add `'warned'` to `StepStatus` union type

### Step 1: Write failing test

Not applicable — this is a type definition change with no runtime logic to test in isolation.

### Step 2: Run test to verify it fails

Not applicable — type definition only.

### Step 3: Implement

In `src/types.ts`, find:
```typescript
export type StepStatus = 'passed' | 'failed' | 'skipped';
```

Replace with:
```typescript
export type StepStatus = 'passed' | 'failed' | 'skipped' | 'warned';
```

### Step 4: Run test to verify it passes

Run: `npm run build`
Expected: Build succeeds with no errors.

### Step 5: Commit

```bash
git add src/types.ts
git commit -m "feat: add 'warned' status to StepStatus type"
```

---

## Task 2: Update StepExecutor to return warned status

**Files:**
- Modify: `src/core/step-executor.ts` — Return `'warned'` when step fails with `continueOnError: true`
- Create: `tests/unit/core/step-executor-continueOnError.test.ts` — Test continueOnError behavior

### Step 1: Write failing test

Create `tests/unit/core/step-executor-continueOnError.test.ts`:
```typescript
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { StepExecutor, createStepExecutor } from '../../../src/core/step-executor';
import { AdapterRegistry } from '../../../src/adapters';
import type { AdapterContext, Logger, UnifiedStep } from '../../../src/types';

describe('StepExecutor - continueOnError', () => {
  let executor: StepExecutor;
  let mockRegistry: AdapterRegistry;
  let mockLogger: Logger;
  let context: AdapterContext;

  beforeEach(() => {
    mockLogger = {
      debug: () => {},
      info: () => {},
      warn: () => {},
      error: () => {},
    };

    mockRegistry = new AdapterRegistry();
    context = {
      variables: {},
      captured: {},
      capture: (name: string, value: unknown) => { context.captured[name] = value; },
      logger: mockLogger,
    };
  });

  afterEach(() => {
    // Cleanup if needed
  });

  it('should return warned status when step fails with continueOnError=true', async () => {
    // Create a step that will fail
    const failingStep: UnifiedStep = {
      id: 'failing-step',
      adapter: 'http',
      action: 'GET',
      params: { url: 'http://nonexistent.invalid/test' },
      continueOnError: true,
    };

    // Create executor with mock registry
    executor = createStepExecutor(mockRegistry, {
      defaultRetries: 0,
      retryDelay: 100,
      logger: mockLogger,
    });

    // Mock the adapter to throw an error
    mockRegistry.get = (name: string) => ({
      execute: async () => {
        throw new Error('Connection refused');
      },
    });

    const interpolationContext = {
      env: {},
      variables: {},
      captured: {},
      builtinFunctions: {},
    };

    const result = await executor.executeStep(failingStep, context, interpolationContext);

    expect(result.status).toBe('warned');
    expect(result.error).toBeDefined();
    expect(result.error?.message).toContain('Connection refused');
  });

  it('should return failed status when step fails without continueOnError', async () => {
    const failingStep: UnifiedStep = {
      id: 'failing-step',
      adapter: 'http',
      action: 'GET',
      params: { url: 'http://nonexistent.invalid/test' },
      continueOnError: false,
    };

    executor = createStepExecutor(mockRegistry, {
      defaultRetries: 0,
      retryDelay: 100,
      logger: mockLogger,
    });

    mockRegistry.get = (name: string) => ({
      execute: async () => {
        throw new Error('Connection refused');
      },
    });

    const interpolationContext = {
      env: {},
      variables: {},
      captured: {},
      builtinFunctions: {},
    };

    const result = await executor.executeStep(failingStep, context, interpolationContext);

    expect(result.status).toBe('failed');
    expect(result.error).toBeDefined();
  });
});
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/core/step-executor-continueOnError.test.ts`
Expected: `FAIL — expected 'passed' to be 'warned'` (test will fail because status is currently `'passed'`)

### Step 3: Implement

In `src/core/step-executor.ts`, find (around line 210-218):
```typescript
            if (step.continueOnError) {
                this.logger.warn(
                    `Step ${step.id} failed but continueOnError=true: ${errorObj.message}`,
                )
                return this.createStepResult(
                    step,
                    'passed',
                    duration,
                    undefined,
                    retryCount,
                    errorObj,
                )
            }
```

Replace with:
```typescript
            if (step.continueOnError) {
                this.logger.warn(
                    `Step ${step.id} failed but continueOnError=true: ${errorObj.message}`,
                )
                return this.createStepResult(
                    step,
                    'warned',
                    duration,
                    undefined,
                    retryCount,
                    errorObj,
                )
            }
```

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/core/step-executor-continueOnError.test.ts`
Expected: 2 tests PASS

### Step 5: Commit

```bash
git add src/core/step-executor.ts tests/unit/core/step-executor-continueOnError.test.ts
git commit -m "fix: return warned status for steps that fail with continueOnError"
```

---

## Task 3: Update console reporter to display warned status

**Files:**
- Modify: `src/reporters/console.reporter.ts` — Add `warned` case to status display methods

### Step 1: Write failing test

Not applicable — UI display changes verified manually in Final Task.

### Step 2: Run test to verify it fails

Not applicable — UI changes only.

### Step 3: Implement

In `src/reporters/console.reporter.ts`, find (around line 83-88):
```typescript
  private getStatusSymbol(status: TestStatus | PhaseStatus | StepStatus): string {
    switch (status) {
      case 'passed':
        return this.colorize(SYMBOLS.pass, 'green');
      case 'failed':
        return this.colorize(SYMBOLS.fail, 'red');
      case 'skipped':
        return this.colorize(SYMBOLS.skip, 'yellow');
```

Insert after the `case 'skipped':` line:
```typescript
      case 'warned':
        return this.colorize(SYMBOLS.warn, 'yellow');
```

Then find (around line 101-107):
```typescript
  private getStatusText(status: TestStatus | PhaseStatus | StepStatus): string {
    switch (status) {
      case 'passed':
        return this.colorize('PASSED', 'green');
      case 'failed':
        return this.colorize('FAILED', 'red');
      case 'skipped':
        return this.colorize('SKIPPED', 'yellow');
```

Insert after the `case 'skipped':` line:
```typescript
      case 'warned':
        return this.colorize('WARNED', 'yellow');
```

Then find the SYMBOLS constant at the top of the file (around line 10-15):
```typescript
const SYMBOLS = {
  pass: '✓',
  fail: '✗',
  skip: '⊙',
```

Add after the `skip` line:
```typescript
  warn: '⚠',
```

### Step 4: Run test to verify it passes

Run: `npm run build`
Expected: Build succeeds with no errors.

### Step 5: Commit

```bash
git add src/reporters/console.reporter.ts
git commit -m "feat(console-reporter): display warned status with warning symbol"
```

---

## Task 4: Update HTML reporter to display warned status

**Files:**
- Modify: `src/reporters/html.reporter.ts` — Add `warned` CSS classes and display logic

### Step 1: Write failing test

Not applicable — UI display changes verified manually in Final Task.

### Step 2: Run test to verify it fails

Not applicable — UI changes only.

### Step 3: Implement

In `src/reporters/html.reporter.ts`, find the CSS section (around line 335-340):
```typescript
  .step-status.passed { background: var(--color-pass); }
  .step-status.failed { background: var(--color-fail); }
  .step-status.skipped { background: var(--color-skip); }
```

Add after the `skipped` line:
```typescript
  .step-status.warned { background: var(--color-warn); }
```

Find the CSS color variables (around line 10-15):
```typescript
    --color-pass: #10b981;
    --color-fail: #ef4444;
    --color-skip: #f59e0b;
```

Add after the `--color-skip` line:
```typescript
    --color-warn: #f59e0b;
```

Find the legend dots (around line 206-210):
```typescript
  .legend-dot.passed { background: var(--color-pass); }
  .legend-dot.failed { background: var(--color-fail); }
  .legend-dot.skipped { background: var(--color-skip); }
```

Add after the `skipped` line:
```typescript
  .legend-dot.warned { background: var(--color-warn); }
```

Find the progress bar segments (around line 182-185):
```typescript
  .progress-bar .passed { background: var(--color-pass); }
  .progress-bar .failed { background: var(--color-fail); }
  .progress-bar .skipped { background: var(--color-skip); }
```

Add after the `skipped` line:
```typescript
  .progress-bar .warned { background: var(--color-warn); }
```

Find the stat cards (around line 145-148):
```typescript
  .stat-card.passed .value { color: var(--color-pass); }
  .stat-card.failed .value { color: var(--color-fail); }
  .stat-card.skipped .value { color: var(--color-skip); }
```

Add after the `skipped` line:
```typescript
  .stat-card.warned .value { color: var(--color-warn); }
```

Now find the HTML template where status badges are rendered (search for `step-status ${step.status}`):
In the step rendering section, add handling for warned status. The existing pattern uses `step-status ${step.status}` which will automatically apply the warned CSS class.

### Step 4: Run test to verify it passes

Run: `npm run build`
Expected: Build succeeds with no errors.

### Step 5: Commit

```bash
git add src/reporters/html.reporter.ts
git commit -m "feat(html-reporter): add warned status styling and display"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All existing tests PASS + 2 new continueOnError tests PASS.

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds with no errors.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Warned status displays in console | Create test with `continueOnError: true` step that fails. Run `e2e run`. | Console shows `⚠ WARNED` in yellow for the step |
| 2 | Warned status displays in HTML | Run `e2e run` with HTML reporter. Open report. | HTML shows yellow badge with "warned" status |
| 3 | Failed status still works | Create test with step that fails without `continueOnError`. Run `e2e run`. | Console shows `✗ FAILED` in red, test stops |
| 4 | Passed status still works | Create test with passing step. Run `e2e run`. | Console shows `✓ PASSED` in green |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `src/types.ts` | Add `'warned'` to `StepStatus` union (1 line) |
| Modify | `src/core/step-executor.ts` | Return `'warned'` instead of `'passed'` when `continueOnError` (1 word change) |
| Create | `tests/unit/core/step-executor-continueOnError.test.ts` | Test continueOnError behavior (60 lines) |
| Modify | `src/reporters/console.reporter.ts` | Add warned symbol and text display (3 additions) |
| Modify | `src/reporters/html.reporter.ts` | Add warned CSS styling (5 additions) |
