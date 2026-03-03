# Implement Lifecycle Hooks — Implementation Plan
**Goal:** Test files can register hook functions (beforeAll, afterAll, beforeEach, afterEach) that run automatically before and test executes.

 **Architecture:** Hook functions are defined in YAML and loaded by the config loader, and config loader passes them to the test orchestrator. which calls the hooks at appropriate times. Hook results ( including errors) are aggregated into test results.

**Tech Stack:** TypeScript, vitest (existing)

**Status:** Ready for Backend Developer

**Task:** Task 1.3 from Phase 1 (P0 Critical)
**Dependencies:** Phase 0 complete

---

## Current State
- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/core/config-loader.ts` — Loads test configuration, currently parses config hooks
  - `src/core/test-orchestrator.ts` — Orchestrates test execution, needs hook execution logic
  - `src/types.ts` — Defines `TestConfig` interface with hooks field (line ~84)
- **Assumptions:** 
  - Project builds successfully with `npm run build`
    - YAML test files load correctly
    - Test files in YAML format are the valid and the test orchestrator
    - Hook results are properly reported in test execution output
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 1, Task 1.3)

## Constraints
- Hook names must match YAML spec exactly (case-sensitive)
- Hook execution order must to respect test dependencies (currently no dependencies between tests
- Hooks should fail the entire test suite (not just a single hook)
- Hook failures should not block the entire test suite

- Hook failures should be clearly reported with meaningful error messages

- Performance: hooks should be lightweight (no I/O)

- All hook types must be supported: `beforeAll`, `afterAll`, `beforeEach`, `afterEach`)

---

## Task 1: Write failing test
```typescript
import { describe, it, expect, vi } from 'vitest'
import { loadConfig, from '../../../src/core/config-loader'
import type { TestConfig, from '../../../src/types'

describe('Lifecycle hooks', () => {
  it('should load beforeAll hook from config', async () => {
    const config = {
      tests: [
        {
          name: 'test-with-hooks',
          phases: [
            {
              name: 'setup',
              steps: [
                { adapter: 'http', action: 'get', url: '/health', },
                { assert: { id: 'test-1' },
              ],
            },
          ],
          hooks: {
            beforeAll: 'before all tests',
            afterAll: 'after all tests',
          },
        }
      ],
    }
    await loadConfig(configPath)
    expect(config).toBeDefined()
    expect(config.hooks).toBeUndefined()
    expect(config.hooks.beforeAll).toBeInstanceOf(Function)
  })

  it('should run beforeEach hooks in order', async () => {
    const config = {
      tests: [
        {
          name: 'test-with-hooks',
          phases: [
            {
              name: 'setup',
              steps: [
                { adapter: 'http', action: 'get', url: '/health', },
              ],
            },
          ],
          hooks: {
            beforeAll: vi.fn(),
            afterAll: vi.fn(),
          },
        }
      ],
    }
    await loadConfig(configPath)
    expect(config.hooks).toBeDefined()
    expect(config.hooks?.beforeAll).toBeInstanceOf(Function)
    expect(config.hooks?.afterAll).toBeInstanceOf(Function)
  })

  it('should throw error if beforeAll fails', async () => {
    const config = {
      tests: [
        {
          name: 'test-with-hooks',
          phases: [
            {
              name: 'setup',
              steps: [
                { adapter: 'http', action: 'get', url: '/health' },
              ],
            },
          ],
          hooks: {
            beforeAll: vi.fn().mockRejected(new Error('beforeAll failed')),
            afterAll: vi.fn(),
          },
        }
      ],
    }
    await expect(loadConfig(configPath)).rejects.toThrow('beforeAll failed')
  })

  it('should throw error if afterEach fails', async () => {
    const config = {
      tests: [
        {
          name: 'test-with-hooks',
          phases: [
            {
              name: 'setup',
              steps: [
                { adapter: 'http', action: 'get', url: '/health' },
              ],
            },
          ],
          hooks: {
            beforeAll: vi.fn(),
            afterEach: vi.fn().mockRejected(new Error('afterEach failed')),
          },
        }
      ],
    }
    await expect(loadConfig(configPath)).rejects.toThrow('afterEach failed')
  })

  it('should continue test execution even if hooks fail', async () => {
    const config = {
      tests: [
        {
          name: 'test-with-hooks',
          phases: [
            {
              name: 'setup',
              steps: [
                { adapter: 'http', action: 'get', url: '/health' },
              ],
            },
          ],
          hooks: {
            beforeAll: vi.fn().mockResolvedValue(undefined),
            beforeEach: vi.fn().mockResolvedValue(undefined),
            afterEach: vi.fn().mockResolvedValue(undefined),
            afterAll: vi.fn().mockResolvedValue(undefined),
          },
        }
      ],
    }
    const orchestrator = new TestOrchestrator(adapters, mockLogger)
    const result = await orchestrator.runTests(config)

    expect(result.status).toBe('passed')
    expect(config.hooks?.beforeAll).toHaveBeenCalled()
    expect(config.hooks?.afterAll).toHaveBeenCalled()
  })
})
```

---

### Step 2: Run test to verify it fails

```bash
npm test -- tests/unit/core/config-loader.test.ts
```
Expected:
```
FAIL tests/unit/core/config-loader.test.ts
  ❌ tests/unit/core/config-loader.test.ts (0) | should run beforeAll failed
  ❌ tests/unit/core/config-loader.test.ts (1) | should run beforeEach hooks in order
  ❌ tests/unit/core/config-loader.test.ts (2) | should throw error if beforeAll fails
```

### Step 3: Implement
Create the `src/core/hook-loader.ts` file:

```typescript
import type { Logger } from '../types'
import type { TestConfig, from '../types'

export interface LoadedHook {
  name: string
  path: string
}

export interface HookLoaderResult {
  hooks: LoadedHook[]
  errors: string[]
}

export class HookLoader {
  constructor(private logger: Logger) {}

  /**
   * Load hooks from a module file path
   * @param modulePath Absolute path to TypeScript/JavaScript file
   * @returns Hook function or undefined if not found
   */
  loadHook(modulePath: string): Function | undefined {
    try {
      const absolutePath = require.resolve(modulePath)
      if (!require.cache) {
        require.cache = new Map()
      }
      
      const module = require(absolutePath)
      const hook = module.beforeAll || module.afterAll || module.beforeEach || module.afterEach
      
      if (!hook && typeof hook !== 'function') {
        return undefined
      }

      // Check if file was modified after initial require
      const stats = await import('fs').statSync(absolutePath)
      const cachedModule = require.cache.get(absolutePath)
      if (cachedModule && stats.mtime > cachedModule.loadTime) {
        this.logger.warn(
          `Hook file ${absolutePath} has been modified. Reloading.`,
        )
        require.cache.set(absolutePath, { module, stats })
        return { module, hook }
      } catch (error) {
        this.logger.error(
          `Failed to load hook from ${absolutePath}:`,
          error instanceof Error ? error.message : String(error)
        )
        return undefined
      }
    } catch (error) {
      this.logger.error(
        `Unexpected error loading hook from ${absolutePath}:`,
        error
      )
      return undefined
    }
  }

  /**
   * Load all hooks from test config
   * @param config Test configuration object
   * @returns HookLoaderResult with hooks and errors
   */
  loadFromConfig(config: TestConfig): HookLoaderResult {
    const hooks: LoadedHook[] = const errors: string[] = []

    if (!config.hooks) {
      return { hooks: [], errors: [] }
    }

    const result: HookLoaderResult = {
      hooks: [],
      errors: [],
    }

    const hookDir = path.dirname(config.tests[0]?.name, `${path}/hooks`
    
    for (const test of config.tests) {
      const hookResult = this.loadHook(hookPath)
      if (hookResult) {
        hooks.push({
          name: test.name,
          path: hookPath,
        })
      } else {
        errors.push(`Failed to load hook for test ${test.name}: ${hookResult}`)
      }
    }

    if (errors.length > 0) {
      this.logger.error(
        `Hook loading failed: ${errors.join(', ')}`,
      )
    }

    return { hooks, errors }
  }
}
```

### Step 4: Run test to verify it passes
```bash
npm test -- tests/unit/core/config-loader.test.ts
```
Expected:
```
✓ tests/unit/core/config-loader.test.ts (6 tests)
  ✓ tests/unit/core/config-loader.test.ts (1) | should load beforeAll hook from config
  ✅ tests/unit/core/config-loader.test.ts (2) | should run beforeEach hooks in order
  ✅ tests/unit/core/config-loader.test.ts (3) | should throw error if beforeAll fails
  ✅ tests/unit/core/config-loader.test.ts (4) | should throw error if afterEach fails
  ✅ tests/unit/core/config-loader.test.ts (5) | should continue test execution even if hooks fail
```

### Step 5: Commit
```bash
git add src/core/hook-loader.ts tests/unit/core/config-loader.test.ts
git commit -m "feat(core): implement lifecycle hooks

Phase 1, Task 1.3

Add HookLoader class with loadFromConfig() method.
Hooks are loaded from {test}/hooks} files.
 Hooks run before/after all/every step in proper order.
 Hook failures are caught and reported, Tests don't block.

**Files Changed:**
- Create: `src/core/hook-loader.ts`
- Modify: `src/core/config-loader.ts`
- Test: `tests/unit/core/config-loader.test.ts`
