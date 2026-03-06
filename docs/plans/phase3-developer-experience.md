# Phase 3: Developer Experience — Implementation Plan

**Goal:** Developers can run tests with `--watch` for auto-reexecution on file changes, and write tests using a fluent TypeScript DSL with full type inference.

**Architecture:** 
- Watch mode adds a file watcher (using chokidar with debouncing) that monitors test directories and re-runs the test suite when changes are detected.
- TypeScript DSL provides a fluent builder API (`test('name').description('...').execute(...)`) that produces `UnifiedTestDefinition` objects.

**Tech Stack:** TypeScript, Node.js, vitest (existing), chokidar (new), minimatch (existing optional dep).

---

## Current State

- **Project root:** `~/e2e-runner`
- **Existing files that matter:**
  - `src/cli/run.command.ts` — Main run command, handles test discovery, loading, execution. Currently ignores `options.watch`.
  - `src/cli/index.ts` — CLI parser, already defines `--watch` option and sets `options.watch: boolean`.
  - `src/types.ts` — Defines `CLIOptions.watch: boolean`, `UnifiedTestDefinition`, `UnifiedStep`.
  - `src/core/ts-loader.ts` — Loads TypeScript test files with `createE2EFunction()`.
  - `src/assertions/expect.ts` — Example of fluent API pattern with chainable `.not` modifier.
  - `package.json` — No file watcher library currently.

- **Assumptions:**
  - The `--watch` flag is already parsed and available as `options.watch` in `runCommand()`.
  - Test discovery and execution logic in `runCommand()` is complete and working.
  - vitest is available for unit tests.

- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 3 section)

## Constraints

- No breaking changes to existing YAML or TypeScript test files.
- Watch mode must be opt-in via `--watch` flag (default: off).
- Watch mode must debounce file changes (300ms minimum) to avoid rapid re-runs.
- DSL must produce `UnifiedTestDefinition` objects compatible with existing loader.
- DSL must provide full TypeScript type inference (no `any` types).
- All new code must have unit tests.
- Follow existing code style: ESM imports, JSDoc comments, explicit return types.

## Rollback

```bash
# Revert Phase 3 commits
git revert HEAD~4  # Adjust count based on actual commits

# Remove new files
rm -f src/core/watcher.ts
rm -rf src/dsl/

# Uninstall chokidar
npm uninstall chokidar @types/chokidar
```

---

## Task 3.1: Add Watch Mode [independent]

**Files:**
- Create: `src/core/watcher.ts`
- Modify: `src/cli/run.command.ts`
- Test: `tests/unit/core/watcher.test.ts`

### Step 1: Write failing test

```typescript
// tests/unit/core/watcher.test.ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { createWatcher, type WatcherOptions } from '../../../src/core/watcher'

describe('createWatcher', () => {
  let tempDir: string
  let watcher: ReturnType<typeof createWatcher> | null = null

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(process.cwd(), 'watcher-test-'))
  })

  afterEach(() => {
    if (watcher) {
      watcher.close()
      watcher = null
    }
    fs.rmSync(tempDir, { recursive: true, force: true })
  })

  it('calls onChange callback when a .test.yaml file is modified', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'example.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    await new Promise(resolve => setTimeout(resolve, 200))
    fs.writeFileSync(testFile, 'name: test-modified')
    await new Promise(resolve => setTimeout(resolve, 300))

    expect(onChange).toHaveBeenCalled()
  })

  it('debounces multiple rapid changes into a single callback', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'debounce.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    await new Promise(resolve => setTimeout(resolve, 200))
    fs.writeFileSync(testFile, 'name: change1')
    fs.writeFileSync(testFile, 'name: change2')
    fs.writeFileSync(testFile, 'name: change3')
    await new Promise(resolve => setTimeout(resolve, 300))

    expect(onChange).toHaveBeenCalledTimes(1)
  })

  it('ignores files that do not match test patterns', async () => {
    const onChange = vi.fn()
    const configFile = path.join(tempDir, 'config.yaml')
    fs.writeFileSync(configFile, 'setting: value')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    await new Promise(resolve => setTimeout(resolve, 200))
    fs.writeFileSync(configFile, 'setting: new-value')
    await new Promise(resolve => setTimeout(resolve, 300))

    expect(onChange).not.toHaveBeenCalled()
  })

  it('close() stops the watcher and prevents further callbacks', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'close.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    await new Promise(resolve => setTimeout(resolve, 200))
    watcher.close()
    watcher = null

    fs.writeFileSync(testFile, 'name: after-close')
    await new Promise(resolve => setTimeout(resolve, 300))

    expect(onChange).not.toHaveBeenCalled()
  })
})
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/core/watcher.test.ts`
Expected: `FAIL — Cannot find module '../../../src/core/watcher'`

### Step 3: Implement

First install chokidar:

```bash
npm install chokidar
npm install -D @types/chokidar
```

Create `src/core/watcher.ts`:

```typescript
/**
 * E2E Test Runner - File Watcher
 *
 * Monitors test directories for changes and triggers re-runs.
 */

import * as path from 'node:path'
import chokidar from 'chokidar'
import type { FSWatcher } from 'chokidar'
import { minimatch } from 'minimatch'

export interface WatcherOptions {
  paths: string[]
  patterns: string[]
  debounceMs: number
  onChange: (changedPath: string) => void
  onError?: (error: Error) => void
}

export interface Watcher {
  close(): void
}

export function createWatcher(options: WatcherOptions): Watcher {
  const { paths, patterns, debounceMs, onChange, onError } = options

  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let lastChangedPath: string | null = null

  function matchesPattern(filePath: string): boolean {
    return patterns.some(pattern => minimatch(filePath, pattern))
  }

  function handleChange(eventPath: string): void {
    if (!matchesPattern(eventPath)) return

    lastChangedPath = eventPath
    if (debounceTimer) clearTimeout(debounceTimer)

    debounceTimer = setTimeout(() => {
      if (lastChangedPath) onChange(lastChangedPath)
      debounceTimer = null
      lastChangedPath = null
    }, debounceMs)
  }

  const internalWatcher = chokidar.watch(paths, {
    ignored: /(node_modules|\.git)/,
    ignoreInitial: true,
    awaitWriteFinish: { stabilityThreshold: 100, pollInterval: 50 },
  })

  internalWatcher
    .on('add', handleChange)
    .on('change', handleChange)
    .on('unlink', handleChange)

  if (onError) internalWatcher.on('error', onError)

  return {
    close: () => {
      if (debounceTimer) clearTimeout(debounceTimer)
      internalWatcher.close()
    },
  }
}
```

Modify `src/cli/run.command.ts` - add import at top and wrap with watch mode logic. The key change is to check `options.watch` early and set up a watcher after the initial run.

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/core/watcher.test.ts`
Expected: `4 tests PASS`

### Step 5: Commit

```bash
git add src/core/watcher.ts src/cli/run.command.ts tests/unit/core/watcher.test.ts package.json
git commit -m "feat(watch): add --watch mode for auto-rerunning tests on file changes"
```

---

## Task 3.2: Add TypeScript Test DSL [independent]

**Files:**
- Create: `src/dsl/types.ts`
- Create: `src/dsl/builder.ts`
- Create: `src/dsl/index.ts`
- Test: `tests/unit/dsl/builder.test.ts`

### Step 1: Write failing test

```typescript
// tests/unit/dsl/builder.test.ts
import { describe, it, expect } from 'vitest'
import { test, step, http, shell } from '../../../src/dsl'

describe('TypeScript DSL Builder', () => {
  describe('test() builder', () => {
    it('creates a test definition with name and execute phase', () => {
      const definition = test('my-test')
        .execute([
          http.get('/api/health').expectStatus(200)
        ])
        .build()

      expect(definition.name).toBe('my-test')
      expect(definition.execute).toHaveLength(1)
      expect(definition.execute[0].adapter).toBe('http')
      expect(definition.execute[0].action).toBe('GET')
    })

    it('adds optional description', () => {
      const definition = test('my-test')
        .description('A sample test')
        .execute([])
        .build()

      expect(definition.description).toBe('A sample test')
    })

    it('adds optional tags', () => {
      const definition = test('my-test')
        .tags('smoke', 'api')
        .execute([])
        .build()

      expect(definition.tags).toEqual(['smoke', 'api'])
    })

    it('adds optional priority', () => {
      const definition = test('my-test')
        .priority('P0')
        .execute([])
        .build()

      expect(definition.priority).toBe('P0')
    })

    it('adds setup phase', () => {
      const definition = test('my-test')
        .setup([
          shell.run('npm run seed')
        ])
        .execute([])
        .build()

      expect(definition.setup).toHaveLength(1)
      expect(definition.setup![0].adapter).toBe('shell')
    })

    it('adds verify phase', () => {
      const definition = test('my-test')
        .execute([])
        .verify([
          http.get('/api/users').expectStatus(200)
        ])
        .build()

      expect(definition.verify).toHaveLength(1)
    })

    it('adds teardown phase', () => {
      const definition = test('my-test')
        .execute([])
        .teardown([
          shell.run('npm run cleanup')
        ])
        .build()

      expect(definition.teardown).toHaveLength(1)
    })
  })

  describe('http step builder', () => {
    it('creates GET request step', () => {
      const s = http.get('/api/users').expectStatus(200).build()

      expect(s.adapter).toBe('http')
      expect(s.action).toBe('GET')
      expect(s.params.url).toBe('/api/users')
      expect(s.assert).toEqual({ status: 200 })
    })

    it('creates POST request step with body', () => {
      const s = http.post('/api/users')
        .body({ name: 'John' })
        .expectStatus(201)
        .build()

      expect(s.action).toBe('POST')
      expect(s.params.body).toEqual({ name: 'John' })
    })

    it('creates request with headers', () => {
      const s = http.get('/api/private')
        .header('Authorization', 'Bearer token')
        .expectStatus(200)
        .build()

      expect(s.params.headers).toEqual({ Authorization: 'Bearer token' })
    })

    it('captures response value', () => {
      const s = http.get('/api/users/1')
        .capture('userId', '$.id')
        .expectStatus(200)
        .build()

      expect(s.capture).toEqual({ userId: '$.id' })
    })
  })

  describe('shell step builder', () => {
    it('creates shell command step', () => {
      const s = shell.run('echo hello').build()

      expect(s.adapter).toBe('shell')
      expect(s.action).toBe('execute')
      expect(s.params.command).toBe('echo hello')
    })

    it('adds timeout', () => {
      const s = shell.run('sleep 10').timeout(5000).build()

      expect(s.params.timeout).toBe(5000)
    })
  })
})
```

### Step 2: Run test to verify it fails

Run: `npm test -- tests/unit/dsl/builder.test.ts`
Expected: `FAIL — Cannot find module '../../../src/dsl'`

### Step 3: Implement

Create `src/dsl/types.ts`:

```typescript
/**
 * DSL Type Definitions
 */

import type { TestPriority, UnifiedStep } from '../types'

export interface TestBuilder {
  description(desc: string): TestBuilder
  tags(...tags: string[]): TestBuilder
  priority(p: TestPriority): TestBuilder
  setup(steps: StepBuilder[]): TestBuilder
  execute(steps: StepBuilder[]): TestBuilder
  verify(steps: StepBuilder[]): TestBuilder
  teardown(steps: StepBuilder[]): TestBuilder
  build(): import('../types').UnifiedTestDefinition
}

export interface StepBuilder {
  build(): UnifiedStep
}

export interface HttpStepBuilder extends StepBuilder {
  header(name: string, value: string): HttpStepBuilder
  body(data: unknown): HttpStepBuilder
  expectStatus(code: number): HttpStepBuilder
  capture(name: string, jsonPath: string): HttpStepBuilder
}

export interface ShellStepBuilder extends StepBuilder {
  timeout(ms: number): ShellStepBuilder
  capture(name: string, jsonPath: string): ShellStepBuilder
}
```

Create `src/dsl/builder.ts`:

```typescript
/**
 * DSL Builders - Fluent API for defining tests
 */

import type { TestPriority, UnifiedStep, UnifiedTestDefinition } from '../types'
import type { TestBuilder, StepBuilder, HttpStepBuilder, ShellStepBuilder } from './types'

// ============================================================================
// Test Builder
// ============================================================================

class TestBuilderImpl implements TestBuilder {
  private _name: string
  private _description?: string
  private _tags?: string[]
  private _priority?: TestPriority
  private _setup?: UnifiedStep[]
  private _execute?: UnifiedStep[]
  private _verify?: UnifiedStep[]
  private _teardown?: UnifiedStep[]

  constructor(name: string) {
    this._name = name
  }

  description(desc: string): TestBuilder {
    this._description = desc
    return this
  }

  tags(...tags: string[]): TestBuilder {
    this._tags = tags
    return this
  }

  priority(p: TestPriority): TestBuilder {
    this._priority = p
    return this
  }

  setup(steps: StepBuilder[]): TestBuilder {
    this._setup = steps.map(s => s.build())
    return this
  }

  execute(steps: StepBuilder[]): TestBuilder {
    this._execute = steps.map(s => s.build())
    return this
  }

  verify(steps: StepBuilder[]): TestBuilder {
    this._verify = steps.map(s => s.build())
    return this
  }

  teardown(steps: StepBuilder[]): TestBuilder {
    this._teardown = steps.map(s => s.build())
    return this
  }

  build(): UnifiedTestDefinition {
    if (!this._execute || this._execute.length === 0) {
      throw new Error('Test must have at least one execute step')
    }

    return {
      name: this._name,
      description: this._description,
      tags: this._tags,
      priority: this._priority,
      setup: this._setup,
      execute: this._execute,
      verify: this._verify,
      teardown: this._teardown,
      sourceFile: 'dsl',
      sourceType: 'typescript',
    }
  }
}

// ============================================================================
// HTTP Step Builder
// ============================================================================

class HttpStepBuilderImpl implements HttpStepBuilder {
  private _url: string
  private _method: string
  private _headers?: Record<string, string>
  private _body?: unknown
  private _assert?: Record<string, unknown>
  private _capture?: Record<string, string>

  constructor(method: string, url: string) {
    this._method = method
    this._url = url
  }

  header(name: string, value: string): HttpStepBuilder {
    if (!this._headers) this._headers = {}
    this._headers[name] = value
    return this
  }

  body(data: unknown): HttpStepBuilder {
    this._body = data
    return this
  }

  expectStatus(code: number): HttpStepBuilder {
    if (!this._assert) this._assert = {}
    this._assert.status = code
    return this
  }

  capture(name: string, jsonPath: string): HttpStepBuilder {
    if (!this._capture) this._capture = {}
    this._capture[name] = jsonPath
    return this
  }

  build(): UnifiedStep {
    return {
      id: `http-${Date.now()}`,
      adapter: 'http',
      action: this._method,
      params: {
        url: this._url,
        headers: this._headers,
        body: this._body,
      },
      assert: this._assert,
      capture: this._capture,
    }
  }
}

// ============================================================================
// Shell Step Builder
// ============================================================================

class ShellStepBuilderImpl implements ShellStepBuilder {
  private _command: string
  private _timeout?: number
  private _capture?: Record<string, string>

  constructor(command: string) {
    this._command = command
  }

  timeout(ms: number): ShellStepBuilder {
    this._timeout = ms
    return this
  }

  capture(name: string, jsonPath: string): ShellStepBuilder {
    if (!this._capture) this._capture = {}
    this._capture[name] = jsonPath
    return this
  }

  build(): UnifiedStep {
    return {
      id: `shell-${Date.now()}`,
      adapter: 'shell',
      action: 'execute',
      params: {
        command: this._command,
        timeout: this._timeout,
      },
      capture: this._capture,
    }
  }
}

// ============================================================================
// Factory Functions
// ============================================================================

export function test(name: string): TestBuilder {
  return new TestBuilderImpl(name)
}

export const http = {
  get: (url: string) => new HttpStepBuilderImpl('GET', url),
  post: (url: string) => new HttpStepBuilderImpl('POST', url),
  put: (url: string) => new HttpStepBuilderImpl('PUT', url),
  patch: (url: string) => new HttpStepBuilderImpl('PATCH', url),
  delete: (url: string) => new HttpStepBuilderImpl('DELETE', url),
}

export const shell = {
  run: (command: string) => new ShellStepBuilderImpl(command),
}

// Alias for step builders
export const step = { http, shell }
```

Create `src/dsl/index.ts`:

```typescript
/**
 * TypeScript Test DSL
 *
 * Fluent API for defining E2E tests with full type inference.
 *
 * @example
 * ```typescript
 * import { test, http, shell } from 'e2e-runner/dsl'
 *
 * export default test('API Health Check')
 *   .description('Verify API is healthy')
 *   .tags('smoke', 'health')
 *   .priority('P0')
 *   .execute([
 *     http.get('/health').expectStatus(200)
 *   ])
 *   .build()
 * ```
 */

export { test, http, shell, step } from './builder'
export type { TestBuilder, StepBuilder, HttpStepBuilder, ShellStepBuilder } from './types'
```

### Step 4: Run test to verify it passes

Run: `npm test -- tests/unit/dsl/builder.test.ts`
Expected: `15 tests PASS`

### Step 5: Commit

```bash
git add src/dsl/ tests/unit/dsl/
git commit -m "feat(dsl): add fluent TypeScript DSL for test definitions"
```

---

## Final Task: Verification

**Files:** None — verification only.

### Step 1: Run full test suite

Run: `npm test`
Expected: All tests PASS (existing + new watcher + new dsl tests)

### Step 2: Run build

Run: `npm run build`
Expected: Build succeeds with no TypeScript errors.

### Step 3: Manual smoke test

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Watch mode starts | `e2e run --watch` in test directory | Shows "Watch mode enabled" message |
| 2 | Watch mode re-runs | Modify a .test.yaml file | Tests re-run automatically |
| 3 | Watch mode ignores | Modify a non-test file | No re-run triggered |
| 4 | DSL test runs | Create test using DSL and run | Test executes correctly |
| 5 | DSL types inferred | Use DSL in TypeScript file | Full autocomplete and type checking |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Create | `src/core/watcher.ts` | File watcher with debouncing |
| Create | `src/dsl/types.ts` | DSL type definitions |
| Create | `src/dsl/builder.ts` | Fluent builder implementations |
| Create | `src/dsl/index.ts` | DSL public exports |
| Modify | `src/cli/run.command.ts` | Integrate watch mode |
| Create | `tests/unit/core/watcher.test.ts` | Watcher unit tests |
| Create | `tests/unit/dsl/builder.test.ts` | DSL unit tests |

---

## Dependency Graph

```
Phase 3 (Developer Experience) — parallel track
    ├── Task 3.1 (independent) — Watch Mode
    └── Task 3.2 (independent) — TypeScript DSL
```

Both tasks can be assigned in parallel to the same or different developers.
