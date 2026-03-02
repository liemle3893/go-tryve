# Coding Conventions

**Analysis Date:** 2026-03-02

## Naming Patterns

**Files:**
- kebab-case for all source files: `test-orchestrator.ts`, `step-executor.ts`, `base.adapter.ts`
- Suffix indicates role: `.adapter.ts`, `.command.ts`, `.reporter.ts`, `.loader.ts`
- Test files: `<TEST-ID>.test.yaml` (uppercase kebab with numeric suffix, e.g., `TC-HTTP-ASSERTIONS-001.test.yaml`)

**Classes:**
- PascalCase: `TestOrchestrator`, `StepExecutor`, `ContextFactory`, `HTTPAdapter`, `BaseAdapter`
- Abstract base classes prefixed: `BaseAdapter`, `BaseReporter`

**Functions:**
- camelCase for standalone functions: `createOrchestrator`, `loadConfig`, `withRetry`, `interpolate`
- Factory functions always prefixed with `create`: `createLogger`, `createStepExecutor`, `createAdapterRegistry`, `createContextFactory`

**Variables / Constants:**
- camelCase for variables and parameters: `testDir`, `exitCode`, `logLevel`
- SCREAMING_SNAKE_CASE for module-level constants: `EXIT_CODES`, `DEFAULT_RETRY_OPTIONS`, `BUILT_IN_FUNCTIONS`, `TYPESCRIPT_FUNCTION_ACTION`
- Boolean flags use `is`/`should`/`has` prefix: `isConnected`, `shouldBail`, `hasInterpolation`

**Types / Interfaces:**
- PascalCase for all type definitions: `UnifiedTestDefinition`, `AdapterContext`, `RetryOptions`
- Type union aliases use descriptive names: `TestStatus`, `PhaseStatus`, `StepStatus`, `LogLevel`
- Option interfaces suffixed with `Options`: `OrchestratorOptions`, `StepExecutorOptions`, `RetryOptions`
- Result interfaces suffixed with `Result`: `TestExecutionResult`, `AdapterStepResult`, `MatcherResult`
- Config interfaces suffixed with `Config`: `E2EConfig`, `LoadedConfig`, `AdapterConfig`
- Data interfaces for event payloads suffixed with `Data`: `SuiteStartData`, `TestEndData`

**Enum-like Types:**
- String literal unions instead of enums: `type TestStatus = 'passed' | 'failed' | 'skipped' | 'error'`
- `const` objects for flag groups: `const EXIT_CODES = { SUCCESS: 0, TEST_FAILURE: 1, ... } as const`

## Code Style

**Formatting:**
- No Prettier or ESLint config present. Formatting is applied manually and consistently by convention.
- 4-space indentation in most source files (e.g., `src/index.ts`, `src/core/*.ts`)
- 2-space indentation in some utility files (e.g., `src/adapters/base.adapter.ts`, `src/utils/*.ts`)
- Single quotes for strings throughout
- Semicolons at end of statements (mixed — some files use them, some don't, based on origin)

**TypeScript:**
- Compiled with `tsconfig.json`: `"strict": false`, `"noImplicitAny": false`
- `import type` used for type-only imports: `import type { Logger } from '../types'`
- `unknown` used as the safe base type (not `any`): `data: unknown`, `error: unknown`
- `as const` used on literal objects to freeze type inference
- No path aliases configured; relative paths only

## Import Organization

**Order (observed):**
1. Node built-in modules with `node:` prefix: `import { randomUUID } from 'node:crypto'`, `import * as fs from 'node:fs'`
2. Third-party packages: `import pLimit from 'p-limit'`
3. Internal imports by layer: errors → types → utils → adapters → core

**Style:**
- Named imports preferred: `import { createLogger, type LogLevel } from '../utils/logger'`
- Namespace imports for Node built-ins: `import * as fs from 'node:fs'`, `import * as path from 'node:path'`
- No path aliases; all imports use relative paths

**Barrel Files:**
- Every directory has an `index.ts` that re-exports its public API
- `src/core/index.ts`, `src/adapters/index.ts`, `src/assertions/index.ts`, `src/reporters/index.ts`
- Consumers import from the barrel: `import { loadConfig } from '../core'`

## File Organization

**Section Dividers:**
- All files use `// ============================================================================` section headers to divide logical sections
- Sections always labeled: `// Types`, `// Constants`, `// Factory Functions`, `// Helper Functions`

**Section Order within a file (standardized):**
1. File-level JSDoc comment block
2. Import statements
3. `// Types` section (local type definitions)
4. `// Constants` section (module-level constants)
5. Class or main implementation
6. `// Factory Functions` section (create* functions)
7. `// Helper Functions` or utility exports

## Error Handling

**Custom Error Hierarchy:**
- All errors extend `E2ERunnerError` (from `src/errors.ts`) which extends native `Error`
- Each error class stores a machine-readable `code: string` property: `'CONFIG_ERROR'`, `'ASSERTION_ERROR'`, etc.
- Specific error classes: `ConfigurationError`, `ValidationError`, `ConnectionError`, `ExecutionError`, `AssertionError`, `TimeoutError`, `InterpolationError`, `LoaderError`, `AdapterError`

**Error wrapping pattern:**
```typescript
// Wrap unknown errors before re-throwing
const wrapped = wrapError(error, 'Failed to connect adapters')

// Distinguish between known and unknown errors at boundaries
if (isE2ERunnerError(error)) {
    printError(error.message, error.hint)
    return { exitCode: errorCodeToExitCode(error.code) }
}
const wrapped = wrapError(error, 'Unexpected error during test run')
```

**Error propagation:**
- Errors thrown inside adapters, steps, and orchestrator propagate upward
- CLI command handlers are the terminal catch boundary; they convert errors to exit codes
- `finally` blocks always used for cleanup (disconnect adapters, teardown phases)

**Teardown always runs:**
```typescript
try {
    result = await withTimeout(...)
} catch (error) {
    testStatus = 'failed'
} finally {
    await this.runTeardownPhase(test, context, phases)
    await this.runHook('afterEach')
}
```

## Logging

**Framework:** Custom logger from `src/utils/logger.ts`

**Logger interface** (`src/types.ts`):
```typescript
export interface Logger {
    debug(message: string, ...args: unknown[]): void;
    info(message: string, ...args: unknown[]): void;
    warn(message: string, ...args: unknown[]): void;
    error(message: string, ...args: unknown[]): void;
}
```

**Log level conventions:**
- `debug`: Step execution details, variable captures, adapter actions, phase transitions
- `info`: Test start/end, suite start/end, high-level progress
- `warn`: Non-fatal failures, skipped tests, retry attempts, missing hooks
- `error`: Step failures, teardown failures, adapter disconnections

**Logger creation and injection:**
- Created once per command with `createLogger({ level, useColors, timestamp })`
- Injected into all constructors and factory functions as a parameter (never global/singleton)
- Child loggers via `createChildLogger(parent, 'prefix')` for scoped output
- Silent logger `createSilentLogger()` available for programmatic API usage where output is suppressed

## Comments

**File-level JSDoc:**
Every file begins with a top-level JSDoc comment block describing its module:
```typescript
/**
 * E2E Test Runner - Test Orchestrator
 *
 * Manages test lifecycle: setup -> execute -> verify -> teardown
 */
```

**Function-level JSDoc:**
- All exported functions have JSDoc with `@param` and `@returns` tags
- Private/protected methods use shorter single-line or block comments
- Example pattern:
```typescript
/**
 * Execute a function with retry logic
 *
 * @param fn - The async function to retry
 * @param options - Retry configuration
 * @returns Result of the first successful invocation
 */
export async function withRetry<T>(fn: () => Promise<T>, options: Partial<RetryOptions> = {}): Promise<T>
```

**Inline comments:**
- Used sparingly for non-obvious logic: complex regex, protocol edge cases
- Bug-fix comments explain WHY a workaround exists (e.g., in `step-executor.ts` lines 121-126)

## Function Design

**Size:** Most functions are short and focused (10-30 lines). Larger classes split logic into many small private methods.

**Parameters:**
- Options/config always passed as an object (not positional args for 3+ params)
- Optional parameters use `Partial<T>` at call site with defaults applied inside the function
- Callbacks use typed function signatures: `onRetry?: (error: Error, attempt: number, delay: number) => void`

**Return Values:**
- Async functions always return a typed `Promise<T>` — never implicit returns
- Result objects used instead of throwing for recoverable outcomes: `{ exitCode, result }`, `{ pass, message }`
- Void returns for side-effect-only methods (logging, captures)

## Module Design

**Exports:**
- Named exports only — no default exports except test files loaded via dynamic import
- Types exported alongside implementations in same file
- `export type` used for type-only re-exports from barrel files

**Barrel Files:**
- All directories export via `index.ts`
- Barrel re-exports are explicit (not `export * from`) for tree-shaking and discoverability
- Example: `src/adapters/index.ts` explicitly re-exports each class and interface

**Abstract base classes:**
- Used for `BaseAdapter` and `BaseReporter` to enforce interface contracts
- Protected helper methods (`measureDuration`, `successResult`, `failResult`, `logAction`, `logResult`) provided by base classes for subclass reuse

---

*Convention analysis: 2026-03-02*
