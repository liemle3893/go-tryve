# Add Unit Tests for Core Modules — Implementation Plan

**Goal:** Add comprehensive unit tests for all core modules to achieve ≥70% code coverage.

**Architecture:** Create unit test files for each core module in `src/core/`. Each test file covers the module's public API, edge cases, and error paths.

**Tech Stack:** TypeScript, vitest (existing), @vitest/coverage-v8 (existing).

**Status:** Ready for Backend Developer
**Task:** Task 1.5 from Phase 1 (P1 High)
**Dependencies:** Task 1.1 (strict mode), Task 1.2 (coverage reporting)

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/core/test-orchestrator.ts` — Orchestrates test execution
  - `src/core/test-discovery.ts` — Discovers test files
  - `src/core/variable-interpolator.ts` — Interpolates variables
  - `src/core/yaml-loader.ts` — Loads YAML test files
  - `src/core/step-executor.ts` — Executes test steps
  - `src/core/config-loader.ts` — Loads configuration
  - `tests/unit/core/step-executor.test.ts` — Existing test (2 tests)
  - `tests/unit/core/step-executor-continueOnError.test.ts` — Existing test (3 tests)
- **Assumptions:** Phase 0 complete. Strict mode enabled (Task 1.1). Coverage configured (Task 1.2).
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 1, Task 1.5)

## Constraints

- Each test file should cover ≥70% of its target module
- Tests must pass in strict mode
- No changes to existing module behavior
- Follow existing test patterns (describe/it/expect)
- Use mocks for external dependencies (adapters, file system)

## Rollback

```bash
git revert HEAD~N  # Where N is number of commits for this task
```

---

## Task 1: Add tests for test-orchestrator.ts

**Files:**
- Create: `tests/unit/core/test-orchestrator.test.ts`

### Step 1: Write failing test

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { TestOrchestrator } from '../../../src/core/test-orchestrator'
import type { AdapterRegistry, Logger, TestConfig, TestExecutionResult } from '../../../src/types'

describe('TestOrchestrator', () => {
  let orchestrator: TestOrchestrator
  let mockAdapters: AdapterRegistry
  let mockLogger: Logger

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    }

    mockAdapters = {
      getAdapter: vi.fn(),
    } as any

    orchestrator = new TestOrchestrator(mockAdapters, mockLogger)
  })

  it('should run a simple test successfully', async () => {
    const config: TestConfig = {
      tests: [{
        name: 'simple-test',
        phases: [{
          name: 'execute',
          steps: [{
            id: 'step-1',
            adapter: 'http',
            action: 'get',
            params: { url: 'http://example.com' },
          }],
        }],
      }],
    }

    const result = await orchestrator.runTests(config)

    expect(result).toBeDefined()
    expect(result.status).toBe('passed')
  })

  it('should report test failures correctly', async () => {
    const config: TestConfig = {
      tests: [{
        name: 'failing-test',
        phases: [{
          name: 'execute',
          steps: [{
            id: 'step-1',
            adapter: 'http',
            action: 'get',
            params: { url: 'http://example.com' },
          }],
        }],
      }],
    }

    // Mock adapter to fail
    vi.mocked(mockAdapters.getAdapter).mockReturnValue({
      execute: vi.fn().mockRejectedValue(new Error('Connection failed')),
    } as any)

    const result = await orchestrator.runTests(config)

    expect(result.status).toBe('failed')
  })

  it('should handle multiple phases in order', async () => {
    const executionOrder: string[] = []

    const config: TestConfig = {
      tests: [{
        name: 'multi-phase-test',
        phases: [
          {
            name: 'setup',
            steps: [{ id: 'setup-1', adapter: 'http', action: 'get', params: { url: '/setup' } }],
          },
          {
            name: 'execute',
            steps: [{ id: 'exec-1', adapter: 'http', action: 'get', params: { url: '/exec' } }],
          },
          {
            name: 'teardown',
            steps: [{ id: 'teardown-1', adapter: 'http', action: 'get', params: { url: '/teardown' } }],
          },
        ],
      }],
    }

    await orchestrator.runTests(config)

    // Verify phases ran in order
    expect(executionOrder).toEqual(['setup', 'execute', 'teardown'])
  })
})
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/core/test-orchestrator.test.ts`
Expected: Test file doesn't exist yet, or tests fail due to missing implementation.

### Step 3: Implement

Run: `npm test -- tests/unit/core/test-orchestrator.test.ts`
Expected: Tests pass. Fix any failures by adjusting test mocks.

### Step 4: Verify coverage

Run: `npm run test:coverage`
Expected: Coverage for test-orchestrator.ts ≥70%

### Step 5: Commit

```bash
git add tests/unit/core/test-orchestrator.test.ts
git commit -m "test(core): add unit tests for TestOrchestrator

Phase 1, Task 1.5

Coverage: ≥70% for test-orchestrator.ts"
```

---

## Task 2: Add tests for test-discovery.ts

**Files:**
- Create: `tests/unit/core/test-discovery.test.ts`

### Step 1: Write failing test

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { discoverTests } from '../../../src/core/test-discovery'
import type { Logger } from '../../../src/types'
import * as fs from 'fs'
import * as path from 'path'

vi.mock('fs')
vi.mock('path')

describe('TestDiscovery', () => {
  let mockLogger: Logger

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    }

    vi.clearAllMocks()
  })

  it('should discover YAML test files in directory', async () => {
    vi.mocked(fs.readdirSync).mockReturnValue(['test1.yaml', 'test2.yml', 'readme.md'] as any)
    vi.mocked(path.join).mockImplementation((...args) => args.join('/'))
    vi.mocked(path.extname).mockImplementation((file) => '.' + file.split('.').pop())

    const tests = await discoverTests('/test/dir', mockLogger)

    expect(tests).toHaveLength(2)
    expect(tests[0]).toContain('test1.yaml')
    expect(tests[1]).toContain('test2.yml')
  })

  it('should discover TypeScript test files', async () => {
    vi.mocked(fs.readdirSync).mockReturnValue(['TC-API-001.test.ts', 'helper.ts'] as any)

    const tests = await discoverTests('/test/dir', mockLogger)

    expect(tests).toHaveLength(1)
    expect(tests[0]).toContain('TC-API-001.test.ts')
  })

  it('should handle empty directories', async () => {
    vi.mocked(fs.readdirSync).mockReturnValue([] as any)

    const tests = await discoverTests('/empty/dir', mockLogger)

    expect(tests).toHaveLength(0)
  })

  it('should log discovered tests', async () => {
    vi.mocked(fs.readdirSync).mockReturnValue(['test1.yaml'] as any)

    await discoverTests('/test/dir', mockLogger)

    expect(mockLogger.info).toHaveBeenCalledWith(
      expect.stringContaining('Discovered 1 test file(s)')
    )
  })
})
```

### Step 2-5: Follow same pattern as Task 1

(Run, implement, verify coverage, commit)

---

## Task 3: Add tests for variable-interpolator.ts

**Files:**
- Create: `tests/unit/core/variable-interpolator.test.ts` (note: this file already exists with 42 tests)

### Step 1: Verify existing tests

Run: `npm test -- tests/unit/core/variable-interpolator.test.ts`
Expected: All 42 tests pass.

### Step 2: Check coverage

Run: `npm run test:coverage`
Expected: Coverage for variable-interpolator.ts already ≥70%

### Step 3: Commit (if needed)

If tests already exist and coverage is good, skip commit. Otherwise add missing tests.

---

## Task 4: Add tests for yaml-loader.ts

**Files:**
- Create: `tests/unit/core/yaml-loader.test.ts`

### Step 1-5: Follow same pattern

Write tests for:
- Loading valid YAML files
- Handling malformed YAML
- Validating test structure
- Resolving relative paths
- Error reporting

---

## Task 5: Add tests for config-loader.ts

**Files:**
- Create: `tests/unit/core/config-loader.test.ts`

### Step 1-5: Follow same pattern

Write tests for:
- Loading config files
- Merging CLI arguments
- Validating config schema
- Error handling

---

## Task 6: Enhance existing tests for step-executor.ts

**Files:**
- Modify: `tests/unit/core/step-executor.test.ts`

### Step 1: Review existing tests

The file already has 2 tests. Add more to reach ≥70% coverage.

### Step 2: Add missing test cases

Add tests for:
- Retry logic
- Error handling
- Variable interpolation
- Assertion validation
- Different adapter types

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite with coverage**

Run: `npm run test:coverage`
Expected: All tests pass. Core module coverage ≥70%.

**Step 2: Review coverage report**

Open: `coverage/index.html`
Expected: All files in `src/core/` show ≥70% coverage.

**Step 3: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Coverage meets threshold | `npm run test:coverage` | All core modules ≥70% |
| 2 | Tests pass | `npm test` | All tests pass |
| 3 | No regressions | Run existing E2E tests | E2E tests still pass |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Create | `tests/unit/core/test-orchestrator.test.ts` | Tests for test orchestrator |
| Create | `tests/unit/core/test-discovery.test.ts` | Tests for test discovery |
| Create | `tests/unit/core/yaml-loader.test.ts` | Tests for YAML loader |
| Create | `tests/unit/core/config-loader.test.ts` | Tests for config loader |
| Modify | `tests/unit/core/step-executor.test.ts` | Enhanced tests for step executor |

**Estimated effort:** 40-80 hours (as per roadmap)
