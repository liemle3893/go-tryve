# Implement Test Dependency Ordering — Implementation Plan

**Goal:** Execute tests in topological order based on their dependencies, ensuring prerequisite tests run before dependent tests.

**Architecture:** Test files can declare dependencies using the `depends` field. The test orchestrator builds a dependency graph, performs topological sort, and executes tests in the correct order. Circular dependencies are detected and reported as errors.

**Tech Stack:** TypeScript, vitest (existing).

**Status:** Ready for Backend Developer
**Task:** Task 1.4 from Phase 1 (P0 Critical)
**Dependencies:** Phase 0 complete

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/core/test-orchestrator.ts` — Orchestrates test execution, needs dependency ordering
  - `src/core/test-discovery.ts` — Discovers test files, needs dependency extraction
  - `src/types.ts` — Test definition types (has `depends` field)
- **Assumptions:** Tests currently run in discovery order. Dependencies field exists but is not enforced.
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 1, Task 1.4)

## Constraints

- Topological sort must be stable (consistent ordering for same input)
- Circular dependencies must be detected before execution
- Dependency tests that fail should mark dependent tests as skipped
- No changes to existing test file format
- Performance: O(N + E) where N = tests, E = dependencies

## Rollback

```bash
git revert HEAD~2  # Reverts the 2 commits for this task
```

---

## Task 1: Implement topological sort

**Files:**
- Create: `src/core/topological-sort.ts`
- Create: `tests/unit/core/topological-sort.test.ts`

### Step 1: Write failing test

```typescript
import { describe, it, expect } from 'vitest'
import { topologicalSort, detectCircularDependencies } from '../../../src/core/topological-sort'

describe('Topological Sort', () => {
  it('should sort nodes with no dependencies', () => {
    const nodes = ['a', 'b', 'c']
    const dependencies = new Map<string, string[]>([
      ['a', []],
      ['b', []],
      ['c', []],
    ])

    const sorted = topologicalSort(nodes, dependencies)

    expect(sorted).toEqual(['a', 'b', 'c'])
  })

  it('should sort nodes with simple dependencies', () => {
    const nodes = ['a', 'b', 'c']
    const dependencies = new Map<string, string[]>([
      ['a', []],
      ['b', ['a']],
      ['c', ['b']],
    ])

    const sorted = topologicalSort(nodes, dependencies)

    expect(sorted.indexOf('a')).toBeLessThan(sorted.indexOf('b'))
    expect(sorted.indexOf('b')).toBeLessThan(sorted.indexOf('c'))
  })

  it('should handle multiple dependencies', () => {
    const nodes = ['a', 'b', 'c', 'd']
    const dependencies = new Map<string, string[]>([
      ['a', []],
      ['b', ['a']],
      ['c', ['a']],
      ['d', ['b', 'c']],
    ])

    const sorted = topologicalSort(nodes, dependencies)

    expect(sorted.indexOf('a')).toBeLessThan(sorted.indexOf('b'))
    expect(sorted.indexOf('a')).toBeLessThan(sorted.indexOf('c'))
    expect(sorted.indexOf('b')).toBeLessThan(sorted.indexOf('d'))
    expect(sorted.indexOf('c')).toBeLessThan(sorted.indexOf('d'))
  })

  it('should detect circular dependencies', () => {
    const nodes = ['a', 'b', 'c']
    const dependencies = new Map<string, string[]>([
      ['a', ['c']],
      ['b', ['a']],
      ['c', ['b']],
    ])

    expect(() => topologicalSort(nodes, dependencies)).toThrow('Circular dependency detected')
  })

  it('should detect self-dependencies', () => {
    const nodes = ['a']
    const dependencies = new Map<string, string[]>([
      ['a', ['a']],
    ])

    expect(() => topologicalSort(nodes, dependencies)).toThrow('Circular dependency detected')
  })

  it('should handle empty input', () => {
    const nodes: string[] = []
    const dependencies = new Map<string, string[]>()

    const sorted = topologicalSort(nodes, dependencies)

    expect(sorted).toEqual([])
  })
})
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/core/topological-sort.test.ts`
Expected: Module doesn't exist, tests fail.

### Step 3: Implement

Create `src/core/topological-sort.ts`:

```typescript
/**
 * Topological sort with circular dependency detection
 */

export class CircularDependencyError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'CircularDependencyError'
  }
}

/**
 * Perform topological sort on nodes with dependencies
 * @param nodes List of node identifiers
 * @param dependencies Map of node -> list of nodes it depends on
 * @returns Nodes in topological order (dependencies first)
 * @throws CircularDependencyError if circular dependencies detected
 */
export function topologicalSort<T>(
  nodes: T[],
  dependencies: Map<T, T[]>
): T[] {
  const sorted: T[] = []
  const visited = new Set<T>()
  const visiting = new Set<T>()

  function visit(node: T): void {
    if (visited.has(node)) {
      return
    }

    if (visiting.has(node)) {
      throw new CircularDependencyError(
        `Circular dependency detected involving node: ${String(node)}`
      )
    }

    visiting.add(node)

    const deps = dependencies.get(node) || []
    for (const dep of deps) {
      visit(dep)
    }

    visiting.delete(node)
    visited.add(node)
    sorted.push(node)
  }

  for (const node of nodes) {
    visit(node)
  }

  return sorted
}

/**
 * Detect circular dependencies without sorting
 * @param nodes List of node identifiers
 * @param dependencies Map of node -> list of nodes it depends on
 * @returns Array of cycles found (empty if no cycles)
 */
export function detectCircularDependencies<T>(
  nodes: T[],
  dependencies: Map<T, T[]>
): T[][] {
  const cycles: T[][] = []
  const visited = new Set<T>()
  const path: T[] = []

  function dfs(node: T): void {
    if (visited.has(node)) {
      return
    }

    const cycleStart = path.indexOf(node)
    if (cycleStart !== -1) {
      cycles.push([...path.slice(cycleStart), node])
      return
    }

    path.push(node)

    const deps = dependencies.get(node) || []
    for (const dep of deps) {
      dfs(dep)
    }

    path.pop()
    visited.add(node)
  }

  for (const node of nodes) {
    dfs(node)
  }

  return cycles
}
```

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/core/topological-sort.test.ts`
Expected: All 6 tests pass.

### Step 5: Commit

```bash
git add src/core/topological-sort.ts tests/unit/core/topological-sort.test.ts
git commit -m "feat(core): implement topological sort with cycle detection

Phase 1, Task 1.4

Add topologicalSort() and detectCircularDependencies() functions.
Tests: 6 unit tests covering all edge cases."
```

---

## Task 2: Integrate dependency ordering into orchestrator

**Files:**
- Modify: `src/core/test-orchestrator.ts` — Sort tests before execution
- Modify: `src/core/test-discovery.ts` — Extract dependencies from test files

### Step 1: Write failing test

Not applicable — integration testing.

### Step 2: Run test to verify current state

Run: `npm test`
Expected: All tests pass.

### Step 3: Implement

In `src/core/test-orchestrator.ts`, find the test execution logic and add:

```typescript
import { topologicalSort, CircularDependencyError } from './topological-sort.js'

export class TestOrchestrator {
  async runTests(config: TestConfig): Promise<TestExecutionResult> {
    // Extract dependencies from tests
    const testNames = config.tests.map(t => t.name)
    const dependencies = new Map<string, string[]>()
    for (const test of config.tests) {
      dependencies.set(test.name, test.depends || [])
    }

    // Sort tests topologically
    let sortedTestNames: string[]
    try {
      sortedTestNames = topologicalSort(testNames, dependencies)
    } catch (error) {
      if (error instanceof CircularDependencyError) {
        this.logger.error(`Circular dependency detected: ${error.message}`)
        return {
          status: 'error',
          error: error.message,
        }
      }
      throw error
    }

    // Execute tests in sorted order
    const testResults: TestResult[] = []
    const failedTests = new Set<string>()

    for (const testName of sortedTestNames) {
      const test = config.tests.find(t => t.name === testName)!
      const deps = dependencies.get(testName) || []

      // Check if any dependency failed
      const failedDeps = deps.filter(dep => failedTests.has(dep))
      if (failedDeps.length > 0) {
        this.logger.warn(
          `Skipping test ${testName} because dependencies failed: ${failedDeps.join(', ')}`
        )
        testResults.push({
          name: testName,
          status: 'skipped',
          skipReason: `Dependencies failed: ${failedDeps.join(', ')}`,
        })
        continue
      }

      // Execute test
      const result = await this.runTest(test)
      testResults.push(result)

      if (result.status === 'failed') {
        failedTests.add(testName)
      }
    }

    // ... rest of result aggregation
  }
}
```

### Step 4: Verify tests pass

Run: `npm test`
Expected: All tests pass, including new dependency ordering.

### Step 5: Commit

```bash
git add src/core/test-orchestrator.ts src/core/test-discovery.ts
git commit -m "feat(core): integrate dependency ordering into test orchestrator

Phase 1, Task 1.4

Tests now execute in topological order.
Failed dependencies cause dependent tests to skip.
Circular dependencies are detected and reported."
```

---

## Final Task: Verification

**Files:** None — verification only.

**Step 1: Run full test suite**

Run: `npm test`
Expected: All tests pass.

**Step 2: Manual smoke test**

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Independent tests | Run tests with no dependencies | Tests execute in discovery order |
| 2 | Simple dependency | Test B depends on A | A runs before B |
| 3 | Failed dependency | Test A fails, B depends on A | A runs, B skips with reason |
| 4 | Circular dependency | A depends on B, B depends on A | Error reported, execution stops |
| 5 | Multiple dependencies | D depends on B and C | B and C run before D |

**Step 3: Create test YAML files**

Create test files demonstrating dependencies:
- `tests/e2e/dependencies/TC-DEP-001.yaml` (no dependencies)
- `tests/e2e/dependencies/TC-DEP-002.yaml` (depends on TC-DEP-001)
- `tests/e2e/dependencies/TC-DEP-003.yaml` (depends on TC-DEP-001, TC-DEP-002)

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Create | `src/core/topological-sort.ts` | Topological sort implementation (~80 lines) |
| Create | `tests/unit/core/topological-sort.test.ts` | Unit tests (~100 lines) |
| Modify | `src/core/test-orchestrator.ts` | Add dependency ordering logic (~30 lines added) |
| Modify | `src/core/test-discovery.ts` | Extract dependencies from test files (~10 lines added) |

**Estimated effort:** 3-4 hours (as per roadmap)
