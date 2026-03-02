# Architecture

**Analysis Date:** 2026-03-02

## Pattern Overview

**Overall:** Layered Plugin Architecture with Event-Driven Reporting

**Key Characteristics:**
- Strict separation between CLI orchestration, core execution engine, and adapter plugins
- Dual entry point: CLI (`bin/e2e.js` → `src/index.ts`) and programmatic API (exported functions from `src/index.ts`)
- Observer/Event pattern for reporter integration — orchestrator emits events, reporters subscribe
- Strategy pattern for adapters — all implement `BaseAdapter` abstract class; registry selects by type
- Test execution is phase-based and sequential within a test, parallel across tests via `p-limit`

## Layers

**CLI Layer:**
- Purpose: Parse arguments, route commands, output help/errors
- Location: `src/cli/`
- Contains: Command handlers (`run.command.ts`, `validate.command.ts`, `list.command.ts`, `health.command.ts`, `init.command.ts`, `test.command.ts`), argument parser (`index.ts`)
- Depends on: Core layer, Adapters layer, Reporters layer, Utils
- Used by: `src/index.ts` main() function

**Core Layer:**
- Purpose: Test lifecycle management — config loading, discovery, loading, orchestration, execution
- Location: `src/core/`
- Contains: `config-loader.ts`, `test-discovery.ts`, `yaml-loader.ts`, `ts-loader.ts`, `context-factory.ts`, `variable-interpolator.ts`, `step-executor.ts`, `test-orchestrator.ts`
- Depends on: Adapters layer (via `AdapterRegistry`), Assertions layer, Types, Errors, Utils
- Used by: CLI layer and programmatic API callers

**Adapters Layer:**
- Purpose: Provide uniform `execute(action, params, ctx)` interface to varied backends
- Location: `src/adapters/`
- Contains: `base.adapter.ts`, `adapter-registry.ts`, `http.adapter.ts`, `postgresql.adapter.ts`, `mongodb.adapter.ts`, `redis.adapter.ts`, `eventhub.adapter.ts`
- Depends on: Types, Errors
- Used by: Core layer (StepExecutor dispatches to adapters via registry)

**Assertions Layer:**
- Purpose: Evaluate assertion expressions against adapter results
- Location: `src/assertions/`
- Contains: `assertion-runner.ts`, `matchers.ts`, `jsonpath.ts`, `expect.ts`
- Depends on: Types, Errors
- Used by: Adapters (each adapter calls assertion-runner directly when processing step results)

**Reporters Layer:**
- Purpose: Format and output test results in various formats
- Location: `src/reporters/`
- Contains: `base.reporter.ts`, `console.reporter.ts`, `junit.reporter.ts`, `html.reporter.ts`, `json.reporter.ts`, `index.ts` (ReporterManager)
- Depends on: Types
- Used by: CLI layer (run.command.ts wires orchestrator events to reporter manager)

**Types & Errors (Foundation):**
- Purpose: Shared contracts and typed error hierarchy
- Location: `src/types.ts`, `src/errors.ts`
- Contains: All interfaces (`E2EConfig`, `UnifiedTestDefinition`, `UnifiedStep`, `TestExecutionResult`, etc.); error classes (`E2ERunnerError`, `ConfigurationError`, `ValidationError`, `AdapterError`, `AssertionError`, `TimeoutError`, etc.)
- Depends on: Nothing (no internal imports)
- Used by: All other layers

**Utils:**
- Purpose: Shared cross-cutting utilities
- Location: `src/utils/`
- Contains: `logger.ts` (leveled logger with ANSI colors), `retry.ts` (withRetry, withTimeout, pollUntil), `exit-codes.ts`
- Depends on: Types, Errors
- Used by: All layers

## Data Flow

**CLI Run Command Flow:**

1. `bin/e2e.js` requires `dist/index.js`, calls `main()`
2. `src/index.ts` parses CLI args → routes to `runCommand()` in `src/cli/run.command.ts`
3. `run.command.ts` calls `loadConfig()` from `src/core/config-loader.ts` → returns `LoadedConfig`
4. `discoverTests()` from `src/core/test-discovery.ts` walks filesystem, finds `*.test.yaml` / `*.test.ts`
5. Filter functions (`filterTestsByTags`, `filterTestsByPriority`, `filterTestsByGrep`) narrow the set
6. `loadYAMLTest()` / `loadTSTest()` parse files into `UnifiedTestDefinition[]`
7. `getRequiredAdapters()` inspects all steps to determine which adapter types are needed
8. `createAdapterRegistry()` instantiates only required adapters; `connectAll()` runs in parallel
9. `createReporterManager()` instantiates configured reporters
10. `createOrchestrator()` creates `TestOrchestrator`; event listener bridges orchestrator events to reporter manager
11. `orchestrator.runSuite(definitions)` executes tests with `p-limit` concurrency control
12. Per-test: `contextFactory.createTestContext()` → phases run sequentially (setup → execute → verify → teardown)
13. Per-step: `stepExecutor.executeStep()` interpolates variables → dispatches to adapter → asserts
14. Events emitted at each lifecycle point: `suite:start`, `test:start`, `phase:start`, `step:start`, `step:end`, `phase:end`, `test:end`, `suite:end`
15. `reporterManager.generateReports(result)` writes final output files
16. Exit code mapped from `TestSuiteResult.success` via `EXIT_CODES`

**Variable Interpolation Flow:**

1. `ContextFactory.createTestContext()` merges `config.variables` + `test.variables` → `variables` map
2. During step execution, `StepExecutor` calls `interpolateObject(step.params, interpolationContext)`
3. `variable-interpolator.ts` resolves `{{varName}}`, `{{captured.field}}`, `{{$uuid}}`, `{{$env(VAR)}}` etc.
4. Captured values from prior steps flow into subsequent step params via shared `captured` object

**Test Loading Flow:**

- YAML: `yaml-loader.ts` parses YAML → validates schema → normalizes to `UnifiedTestDefinition`
- TypeScript: `ts-loader.ts` dynamically `require()`s the TS test file → calls the exported test factory function → gets `UnifiedTestDefinition`

**State Management:**
- Per-test state lives in `TestContext` (created fresh per test by `ContextFactory`)
- `captured` is a plain `Record<string, unknown>` mutated in-place via `context.capture(name, value)`
- No global mutable state between tests (each test gets independent context)

## Key Abstractions

**BaseAdapter:**
- Purpose: Contract all service connectors must implement
- Examples: `src/adapters/http.adapter.ts`, `src/adapters/postgresql.adapter.ts`, `src/adapters/mongodb.adapter.ts`, `src/adapters/redis.adapter.ts`, `src/adapters/eventhub.adapter.ts`
- Pattern: Abstract class with `connect()`, `disconnect()`, `execute(action, params, ctx)`, `healthCheck()` — plus protected helpers `measureDuration()`, `successResult()`, `failResult()`

**AdapterRegistry:**
- Purpose: Lifecycle manager and lookup for adapter instances; only initializes adapters used by discovered tests
- Examples: `src/adapters/adapter-registry.ts`
- Pattern: Map-based registry keyed by `AdapterType`; factory function `createAdapterRegistry()`; `getRequiredAdapters()` analyzes test steps before initialization

**UnifiedTestDefinition:**
- Purpose: Single normalized representation for both YAML and TypeScript tests
- Examples: Defined in `src/types.ts`; populated by `src/core/yaml-loader.ts` and `src/core/ts-loader.ts`
- Pattern: Phases (`setup`, `execute`, `verify`, `teardown`) each contain `UnifiedStep[]`

**BaseReporter:**
- Purpose: Event-driven reporting contract
- Examples: `src/reporters/base.reporter.ts`, extended by `console.reporter.ts`, `junit.reporter.ts`, `html.reporter.ts`, `json.reporter.ts`
- Pattern: Abstract class with optional `onSuiteStart()` / `onTestStart()` etc. event hooks; required `generateReport(result)` for final output

**TestOrchestrator:**
- Purpose: Central coordinator of test lifecycle, parallelism, and event emission
- Examples: `src/core/test-orchestrator.ts`
- Pattern: Class with `addEventListener()` / `removeEventListener()` for observer pattern; `runSuite()` uses `p-limit` for bounded concurrency; teardown runs in `finally` block regardless of test outcome

## Entry Points

**CLI Entry Point:**
- Location: `bin/e2e.js` → requires `dist/index.js`
- Triggers: Direct execution (`./bin/e2e.js run`) or via npm `bin` field
- Responsibilities: Bootstraps Node.js process, calls `main()`, catches fatal errors

**Main Source Entry:**
- Location: `src/index.ts`
- Triggers: `bin/e2e.js` (CLI) or `import { runTests } from 'e2e-runner'` (programmatic)
- Responsibilities: Parses CLI args, routes to command handlers, exports programmatic API (`runTests`, `validateTests`, `listTests`, `checkHealth`)

**Programmatic API:**
- Location: `src/index.ts` exported functions
- Triggers: `import` from consuming project
- Responsibilities: Wraps CLI commands with sensible defaults; re-exports core types and factory functions

## Error Handling

**Strategy:** Typed error hierarchy rooted at `E2ERunnerError` in `src/errors.ts`; errors carry machine-readable `code` strings mapped to exit codes

**Patterns:**
- `E2ERunnerError` base → `ConfigurationError`, `ValidationError`, `ConnectionError`, `ExecutionError`, `AssertionError`, `TimeoutError`, `InterpolationError`, `LoaderError`, `AdapterError`
- CLI catches `isE2ERunnerError()` → calls `errorCodeToExitCode(error.code)` for structured exit
- `wrapError()` converts unknown `catch` values to `E2ERunnerError` preserving context
- Teardown always runs in `finally` block in `TestOrchestrator.runTest()` regardless of test outcome
- Adapter disconnects always run in `finally` block in `run.command.ts` regardless of suite outcome
- Step `continueOnError: true` allows test phases to continue past individual step failures

## Cross-Cutting Concerns

**Logging:** `createLogger()` in `src/utils/logger.ts` returns a `Logger` interface; leveled (`debug`/`info`/`warn`/`error`/`silent`); ANSI colors optional; timestamp optional. Passed by dependency injection to all major components.

**Validation:** Configuration validated with JSON Schema via `ajv` (optional dep) in `src/core/config-loader.ts`; YAML tests validated against embedded schema in `src/core/yaml-loader.ts`; adapter connection strings validated lazily at startup only for required adapters

**Authentication:** No built-in auth — HTTP adapter sends credentials via request headers/params configured in test YAML; database adapters receive connection strings from `e2e.config.yaml` with `${ENV_VAR}` substitution

---

*Architecture analysis: 2026-03-02*
