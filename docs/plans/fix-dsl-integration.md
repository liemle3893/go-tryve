# Fix DSL Integration Gap — Implementation Plan

**Goal:** Developers can import `e2e-runner/dsl` and write declarative tests using the fluent DSL that execute correctly with the e2e-runner.

**Architecture:** 
- Package exports expose the DSL via `e2e-runner/dsl` submodule import path.
- ts-loader detects DSL-produced `UnifiedTestDefinition` objects (declarative steps) vs function-based tests and handles them directly without conversion.

**Tech Stack:** TypeScript, Node.js, vitest (existing), no new dependencies.

---

## Current State

> Read every file you plan to modify BEFORE writing this section. Do not assume file contents.

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/core/ts-loader.ts` — Loads TypeScript test files. Currently expects `execute` to be a function and wraps it in `__typescript_function__` steps.
  - `src/dsl/builder.ts` — Fluent API builder (exists on integration branch, not on main). Produces `UnifiedTestDefinition` with declarative step arrays.
  - `src/dsl/index.ts` — DSL public exports (exists on integration branch).
  - `src/dsl/types.ts` — DSL type definitions (exists on integration branch).
  - `package.json` — No `exports` field. DSL import path `e2e-runner/dsl` does not resolve.
  - `src/types.ts` — Defines `UnifiedTestDefinition`, `UnifiedStep`, `TestPriority`.
- **Assumptions:** 
  - DSL code (src/dsl/) has been implemented and exists on the integration branch `integration/phase3-developer-experience` (commit dc846b0).
  - The DSL produces valid `UnifiedTestDefinition` objects with `execute` as an array of `UnifiedStep` objects.
  - Main branch does NOT have src/dsl/ directory yet.
- **Design doc:** `docs/plans/phase3-developer-experience.md` (original Phase 3 plan)

## Constraints

- No breaking changes to existing function-based TypeScript tests.
- DSL tests and function-based tests must coexist in the same project.
- All new code must have unit tests.
- Package must remain compatible with Node.js ESM and CommonJS.
- DSL detection must be reliable (no false positives/negatives).

## Rollback

> How to undo everything this plan creates. Must be copy-pasteable.

```bash
git revert HEAD~2  # Adjust based on actual commit count
# Remove exports field from package.json manually if needed
# No new dependencies to uninstall
```

---

## Task 1: Add Package Exports for DSL Submodule

**Files:**
- Modify: `package.json` — Add `exports` field with DSL submodule entry

### Step 1: Write failing test

Create test to verify DSL import path resolves:

```typescript
// tests/unit/package/dsl-import.test.ts
import { describe, it, expect } from 'vitest'

describe('DSL Package Exports', () => {
  it('exports test builder function', async () => {
    const { test } = await import('e2e-runner/dsl')
    expect(typeof test).toBe('function')
  })

  it('exports http step builders', async () => {
    const { http } = await import('e2e-runner/dsl')
    expect(typeof http.get).toBe('function')
    expect(typeof http.post).toBe('function')
    expect(typeof http.put).toBe('function')
    expect(typeof http.patch).toBe('function')
    expect(typeof http.delete).toBe('function')
  })

  it('exports shell step builder', async () => {
    const { shell } = await import('e2e-runner/dsl')
    expect(typeof shell.run).toBe('function')
  })

  it('exports step alias', async () => {
    const { step } = await import('e2e-runner/dsl')
    expect(step.http).toBeDefined()
    expect(step.shell).toBeDefined()
  })
})
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/package/dsl-import.test.ts`
Expected: `FAIL — Cannot find module 'e2e-runner/dsl'`

### Step 3: Implement

Modify `package.json`. Find the closing brace of the file. Insert before the closing brace:

```json
,
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "require": "./dist/index.js",
      "import": "./dist/index.js"
    },
    "./dsl": {
      "types": "./dist/dsl/index.d.ts",
      "require": "./dist/dsl/index.js",
      "import": "./dist/dsl/index.js"
    }
  }
```

After modification, the end of package.json should look like:

```json
  "repository": {
    "type": "git",
    "url": "https://github.com/liemle3893/go-autoflow.git"
  },
  "homepage": "https://github.com/liemle3893/go-autoflow#readme",
  "bugs": {
    "url": "https://github.com/liemle3893/go-autoflow/issues"
  },
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "require": "./dist/index.js",
      "import": "./dist/index.js"
    },
    "./dsl": {
      "types": "./dist/dsl/index.d.ts",
      "require": "./dist/dsl/index.js",
      "import": "./dist/dsl/index.js"
    }
  }
}
```

Note: The DSL files (src/dsl/) must exist on the branch where this is being implemented. If they don't exist, the DSL implementation from the integration branch must be merged first.

### Step 4: Run test to verify it passes

Run: 
```bash
npm run build
npm test -- tests/unit/package/dsl-import.test.ts
```
Expected: `4 tests PASS`

### Step 5: Commit

```bash
git add package.json tests/unit/package/dsl-import.test.ts
git commit -m "feat(package): add exports field for e2e-runner/dsl submodule"
```

---

## Task 2: Add DSL Detection to ts-loader

**Files:**
- Modify: `src/core/ts-loader.ts` — Add DSL detection logic and early return for DSL-produced definitions
- Test: `tests/unit/core/ts-loader-dsl.test.ts`

### Step 1: Write failing test

Create comprehensive test for DSL detection and loading:

```typescript
// tests/unit/core/ts-loader-dsl.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { loadTSTest } from '../../../src/core/ts-loader'
import type { UnifiedTestDefinition } from '../../../src/types'

describe('ts-loader DSL Integration', () => {
  const tempDir = path.join(__dirname, 'temp-dsl-tests')
  
  beforeEach(() => {
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true })
    }
  })

  afterEach(() => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true })
    }
  })

  it('loads DSL test with declarative execute steps', async () => {
    const testFile = path.join(tempDir, 'dsl-basic.test.ts')
    fs.writeFileSync(testFile, `
import { test, http } from '../../../src/dsl'

export default test('Health Check')
  .description('Verify health endpoint')
  .execute([
    http.get('/health').expectStatus(200)
  ])
  .build()
`)

    const definition = await loadTSTest(testFile)
    
    expect(definition.name).toBe('Health Check')
    expect(definition.description).toBe('Verify health endpoint')
    expect(definition.execute).toBeDefined()
    expect(Array.isArray(definition.execute)).toBe(true)
    expect(definition.execute.length).toBeGreaterThan(0)
    expect(definition.execute[0].adapter).toBe('http')
    expect(definition.execute[0].action).toBe('GET')
  })

  it('preserves DSL test tags and priority', async () => {
    const testFile = path.join(tempDir, 'dsl-metadata.test.ts')
    fs.writeFileSync(testFile, `
import { test, http } from '../../../src/dsl'

export default test('API Smoke Test')
  .tags('smoke', 'api', 'critical')
  .priority('P0')
  .execute([
    http.get('/api/ping').expectStatus(200)
  ])
  .build()
`)

    const definition = await loadTSTest(testFile)
    
    expect(definition.tags).toEqual(['smoke', 'api', 'critical'])
    expect(definition.priority).toBe('P0')
  })

  it('loads DSL test with multiple phases', async () => {
    const testFile = path.join(tempDir, 'dsl-phases.test.ts')
    fs.writeFileSync(testFile, `
import { test, http, shell } from '../../../src/dsl'

export default test('Full E2E Test')
  .setup([
    shell.run('npm run seed')
  ])
  .execute([
    http.post('/api/users').body({ name: 'Test' }).expectStatus(201)
  ])
  .verify([
    http.get('/api/users/1').expectStatus(200)
  ])
  .teardown([
    shell.run('npm run cleanup')
  ])
  .build()
`)

    const definition = await loadTSTest(testFile)
    
    expect(definition.setup).toBeDefined()
    expect(definition.setup!.length).toBe(1)
    expect(definition.setup![0].adapter).toBe('shell')
    
    expect(definition.execute).toBeDefined()
    expect(definition.execute[0].adapter).toBe('http')
    expect(definition.execute[0].action).toBe('POST')
    
    expect(definition.verify).toBeDefined()
    expect(definition.verify!.length).toBe(1)
    
    expect(definition.teardown).toBeDefined()
    expect(definition.teardown!.length).toBe(1)
  })

  it('sets sourceFile to actual file path for DSL tests', async () => {
    const testFile = path.join(tempDir, 'dsl-source.test.ts')
    fs.writeFileSync(testFile, `
import { test, http } from '../../../src/dsl'

export default test('Source Test')
  .execute([http.get('/test').expectStatus(200)])
  .build()
`)

    const definition = await loadTSTest(testFile)
    
    expect(definition.sourceFile).toBe(path.resolve(testFile))
    expect(definition.sourceType).toBe('typescript')
  })

  it('still loads function-based tests correctly', async () => {
    const testFile = path.join(tempDir, 'function-test.test.ts')
    fs.writeFileSync(testFile, `
export default {
  execute: async (ctx: unknown) => {
    console.log('Function-based test')
  }
}
`)

    const definition = await loadTSTest(testFile)
    
    expect(definition.name).toBe('function-test')
    expect(definition.execute).toBeDefined()
    expect(Array.isArray(definition.execute)).toBe(true)
    // Function-based tests get wrapped in special steps
    expect(definition.execute[0].action).toBe('__typescript_function__')
  })
})
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/core/ts-loader-dsl.test.ts`
Expected: Tests fail because ts-loader validation expects `execute` to be a function, but DSL provides an array.

### Step 3: Implement

Modify `src/core/ts-loader.ts`. 

First, add the DSL detection helper function. Find the comment line (around line 30):

```typescript
// ============================================================================
// Loader Functions
// ============================================================================
```

Insert before that comment:

```typescript
// ============================================================================
// DSL Detection
// ============================================================================

/**
 * Check if a definition was produced by the DSL builder.
 * DSL definitions have declarative steps with adapters, not functions.
 */
function isDSLDefinition(def: unknown): def is UnifiedTestDefinition {
  if (!def || typeof def !== 'object') {
    return false
  }

  const obj = def as Record<string, unknown>

  // DSL-produced definitions have:
  // 1. name (string)
  // 2. execute array with step objects (not functions)
  if (typeof obj.name !== 'string') {
    return false
  }

  if (!Array.isArray(obj.execute) || obj.execute.length === 0) {
    return false
  }

  // Check if execute[0] is a step object (has adapter, action, params)
  const firstStep = obj.execute[0] as Record<string, unknown>
  if (
    typeof firstStep === 'object' &&
    firstStep !== null &&
    typeof firstStep.adapter === 'string' &&
    typeof firstStep.action === 'string' &&
    firstStep.action !== '__typescript_function__'
  ) {
    return true
  }

  return false
}

```

Now modify the `loadTSTest` function. Find this block (around line 77):

```typescript
    const definition = module.default;

    // Extract test name from module or filename
    const testName = extractTestName(module, filePath);

    // Validate the definition
    validateTSDefinition(definition, filePath);

    // Convert to unified format
    return convertToUnified(testName, definition, filePath);
```

Replace with:

```typescript
    const definition = module.default;

    // Check if this is a DSL-produced definition (already in UnifiedTestDefinition format)
    if (isDSLDefinition(definition)) {
      // DSL definitions are already in the correct format
      // Just ensure sourceFile is set correctly
      return {
        ...definition,
        sourceFile: path.resolve(filePath),
      }
    }

    // Extract test name from module or filename
    const testName = extractTestName(module, filePath);

    // Validate the definition (for function-based tests)
    validateTSDefinition(definition, filePath);

    // Convert to unified format
    return convertToUnified(testName, definition, filePath);
```

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/core/ts-loader-dsl.test.ts`
Expected: `5 tests PASS`

### Step 5: Commit

```bash
git add src/core/ts-loader.ts tests/unit/core/ts-loader-dsl.test.ts
git commit -m "feat(loader): integrate DSL with ts-loader for declarative test definitions"
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All existing tests + new tests PASS (report total count).

**Step 2: Run build**

Run: `npm run build`
Expected: Build succeeds with no TypeScript errors.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | DSL import resolves | Create file importing `import { test, http } from 'e2e-runner/dsl'` and run `npx tsc --noEmit` | No TypeScript errors, types resolve correctly |
| 2 | DSL test loads | Create test file using DSL builder, load via ts-loader programmatically | Returns UnifiedTestDefinition with correct structure |
| 3 | Function test still works | Create traditional function-based test file, load via ts-loader | Returns UnifiedTestDefinition with `__typescript_function__` steps |
| 4 | Mixed tests coexist | Project with both DSL and function-based tests runs successfully | Both test types execute correctly |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `package.json` | Add exports field for e2e-runner/dsl submodule |
| Modify | `src/core/ts-loader.ts` | Add isDSLDetection helper + early return for DSL definitions |
| Create | `tests/unit/package/dsl-import.test.ts` | Verify DSL import path resolves |
| Create | `tests/unit/core/ts-loader-dsl.test.ts` | Test DSL detection and loading |

---

## Dependency Graph

```
Task 1 (Package Exports) [independent]
Task 2 (Loader Integration) [independent]
```

Both tasks can be assigned in parallel. Task 1 enables the import path, Task 2 enables the loader to handle DSL tests. Both are required for end-to-end DSL functionality.

---

## Prerequisites

**CRITICAL:** This plan assumes the DSL code (src/dsl/) exists in the repository. If it does not exist on the current branch:

1. First merge the DSL implementation from the integration branch:
   ```bash
   git merge origin/integration/phase3-developer-experience --no-ff -m "Merge: DSL implementation from Phase 3"
   ```

2. Or cherry-pick the DSL commit:
   ```bash
   git cherry-pick ab2c078  # feat(dsl): add fluent TypeScript DSL for test definitions
   ```

After the DSL code is present, proceed with Task 1 and Task 2 above.
