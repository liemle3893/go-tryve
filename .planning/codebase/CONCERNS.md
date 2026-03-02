# Codebase Concerns

**Analysis Date:** 2026-03-02

## Tech Debt

**Incomplete Assertion Engine in StepExecutor:**
- Issue: `validateAssertions` in `StepExecutor` is a stub that does nothing. It logs "assertions pending validation for Phase 5" but never validates. Actual assertion validation is handled inside each adapter's `execute()` method only — assertions passed via `step.assert` at the step level bypass the shared assertion engine entirely unless the adapter also receives them through `interpolatedParams.assert`.
- Files: `src/core/step-executor.ts` (lines 240-250)
- Impact: Silent test failures — assertions on generic adapter results from non-HTTP/non-PostgreSQL paths may silently pass when they should fail. Comments reference "Phase 5" integration that has never been completed.
- Fix approach: Replace `validateAssertions` stub with actual calls to `src/assertions/assertion-runner.ts`'s `runAssertion`. Remove the "Phase 5" comments once implemented.

**`continueOnError` Reports Failures as `passed`:**
- Issue: In `src/core/step-executor.ts` (lines 137-148), when a step fails and `continueOnError=true`, the step result is created with `status: 'passed'` even though an error is attached. This silently hides failures in reporting.
- Files: `src/core/step-executor.ts` (lines 137-148)
- Impact: Reporters see a "passed" step with an attached error, which is logically inconsistent. A consumer of test results cannot distinguish between a genuine pass and a forgiven failure.
- Fix approach: Introduce a distinct status value such as `'warned'` or `'passedWithError'`, or keep `'failed'` status but flag it as non-blocking in a separate field (e.g., `nonBlocking: true`).

**`TestExecutionResult.retryCount` Always Zero:**
- Issue: `buildTestResult` in `src/core/test-orchestrator.ts` (line 371) hardcodes `retryCount: 0`. Step-level retry counts are tracked in `StepResult` but the test-level count is never aggregated.
- Files: `src/core/test-orchestrator.ts` (line 371)
- Impact: Retry information is invisible at the test result level in reporters and consumers.
- Fix approach: Derive `retryCount` by summing `stepResult.retryCount` across all phases, or track it separately during test execution.

**`depends` Field Parsed but Never Enforced:**
- Issue: The `depends` field is defined in `src/types.ts`, parsed in both `src/core/yaml-loader.ts` and `src/core/ts-loader.ts`, and a `sortTestsByDependencies` utility exists in `src/core/test-discovery.ts`. However, `src/cli/run.command.ts` never calls `sortTestsByDependencies`. Tests run in discovery order regardless of declared dependencies.
- Files: `src/core/test-discovery.ts` (lines 277-311), `src/cli/run.command.ts`
- Impact: A test declaring `depends: [OtherTest]` may execute before its dependency, causing flaky failures.
- Fix approach: Call `sortTestsByDependencies` on the loaded definitions before passing them to the orchestrator in `runCommand`.

**TS Test Functions Stored as Function References in `params`:**
- Issue: TypeScript test functions are stored in `step.params.__function` with `adapter: 'http'` as a placeholder (see `src/core/ts-loader.ts` lines 285-296). This conflates two different execution paths under one adapter type and leaks internal implementation details into the public type system.
- Files: `src/core/ts-loader.ts` (lines 285-296), `src/core/step-executor.ts` (lines 164-166)
- Impact: Fragile — any code that iterates over steps and inspects adapter type will misidentify TypeScript function steps as HTTP steps. The placeholder `adapter: 'http'` is misleading for reporting and filtering.
- Fix approach: Introduce a dedicated `AdapterType` value (e.g., `'typescript'`) to represent function-backed steps, and update validators and reporters accordingly.

**`captureValues` Method in StepExecutor is Defined but Never Called:**
- Issue: `StepExecutor.captureValues()` exists at `src/core/step-executor.ts` (lines 255-263) and is never invoked anywhere in the class. Capture is instead handled inline within each adapter. Additionally, `captureValues` is also exported from `src/adapters/base.adapter.ts` (line 141), creating two parallel but disconnected implementations.
- Files: `src/core/step-executor.ts` (lines 255-263), `src/adapters/base.adapter.ts` (line 141)
- Impact: Dead code increases maintenance surface. Future developers may try to use the unreachable method.
- Fix approach: Remove the unreachable `captureValues` from `StepExecutor`, or consolidate capture logic to use it consistently.

**Mixed `require()` and `import()` in the Same Module:**
- Issue: `src/core/test-discovery.ts` uses synchronous `require('minimatch')` (lines 53, 109) inside an async function while the rest of the codebase uses `await import(...)`. This is an ESM/CJS interop inconsistency.
- Files: `src/core/test-discovery.ts` (lines 53, 109)
- Impact: May break in strict ESM environments. Inconsistent module loading strategy makes future ESM migration harder.
- Fix approach: Replace `require()` calls with `await import('minimatch')` consistently.

**Hook Paths Loaded with Unrestricted `dynamic import`:**
- Issue: `src/core/test-orchestrator.ts` (line 506) executes `await import(hookPath)` where `hookPath` comes directly from the config file. There is no path validation, sandboxing, or allowlist check before loading hook modules.
- Files: `src/core/test-orchestrator.ts` (lines 496-518)
- Impact: Low severity in its current use case (local config files), but represents an arbitrary code execution risk if config files can be tampered with. Any path accepted without validation.
- Fix approach: Validate that `hookPath` resolves to a path within the project root before importing, or document the security boundary clearly.

## Security Considerations

**Unvalidated RegExp Construction from User Strings:**
- Risk: Several locations construct `new RegExp(userProvidedString)` directly from test YAML or config values without escaping or validation.
- Files: `src/adapters/http.adapter.ts` (line 435), `src/adapters/eventhub.adapter.ts` (line 387), `src/assertions/assertion-runner.ts` (line 114), `src/assertions/matchers.ts` (line 326)
- Current mitigation: None. ReDoS (Regular Expression Denial of Service) via a crafted YAML test file is possible.
- Recommendations: Wrap `new RegExp()` in try/catch for each usage. Consider adding pattern length limits or a regex timeout wrapper if test YAML can come from untrusted sources.

**`$file()` Interpolation Function Reads Arbitrary Filesystem Paths:**
- Risk: The `$file(path)` built-in in `src/core/variable-interpolator.ts` (lines 59-68) reads any file path passed as a template argument, including paths outside the project directory (e.g., `$file(/etc/passwd)`).
- Files: `src/core/variable-interpolator.ts` (lines 59-68)
- Current mitigation: None. Any YAML test file can read arbitrary files from the filesystem.
- Recommendations: Restrict `$file()` to paths within the config root or test directory. Resolve the path and verify it starts with an allowed prefix.

**`process.env` Exposed as Full Interpolation Context:**
- Risk: `createInterpolationContext` at `src/core/variable-interpolator.ts` (line 371) passes the entire `process.env` into every interpolation context. Any `{{envVarName}}` reference in a YAML test or assertion message will resolve environment variables.
- Files: `src/core/variable-interpolator.ts` (line 371)
- Current mitigation: Variables must be explicitly referenced in test YAML, so casual exposure requires knowledge of env var names.
- Recommendations: Consider using an explicit allowlist of env vars rather than passing the full `process.env`. This avoids accidental exposure in assertion error messages or logs.

## Performance Bottlenecks

**Sequential Test Metadata Loading During Filtering:**
- Problem: `loadTestMetadata` in `src/cli/run.command.ts` is called per-test inside `filterTestsByTags` and `filterTestsByPriority`. Each call does a full YAML parse or TypeScript module load just to extract `tags` and `priority`.
- Files: `src/cli/run.command.ts` (lines 311-324), `src/core/yaml-loader.ts` (lines 404-416)
- Cause: No caching between the metadata-load phase and the full-load phase. Tests are parsed twice: once for metadata filtering, once for full definition loading.
- Improvement path: Cache metadata after the first parse and reuse when loading full definitions. Alternatively, extract metadata from the file header with a lightweight regex scan rather than full YAML parsing.

**Redis `keys` / `flushPattern` Operations on Large Keyspaces:**
- Problem: `flushPattern` in `src/adapters/redis.adapter.ts` (lines 201-207) uses `KEYS pattern` which blocks the Redis server and loads all matching keys into memory.
- Files: `src/adapters/redis.adapter.ts` (lines 201-207)
- Cause: `KEYS` is a blocking O(N) Redis command, not recommended for production use.
- Improvement path: Replace with `SCAN`-based iteration (cursor loop) to avoid blocking the server.

**HTML Report Generation Loads Entire Suite into Memory:**
- Problem: `src/reporters/html.reporter.ts` (1044 lines) builds the entire HTML string in memory via string concatenation before writing. For large test suites, all result data is held in a single string.
- Files: `src/reporters/html.reporter.ts`
- Cause: No streaming — `buildHTML()` returns the full document as a single string.
- Improvement path: Use a streaming write approach (e.g., write HTML in chunks to a writable stream). For most use cases this is acceptable, but could be problematic for very large suites.

**MongoDB `normalizeFilter` Imports ObjectId on Every Operation:**
- Problem: `normalizeFilter` in `src/adapters/mongodb.adapter.ts` (lines 216-236) calls `await import('mongodb')` to get `ObjectId` on every filter normalization, for every query. Dynamic imports are cached by Node.js after the first call, but this is still unnecessary repeated overhead.
- Files: `src/adapters/mongodb.adapter.ts` (lines 216-236)
- Cause: Lazy import used inside a frequently-called private method rather than at connection time.
- Improvement path: Import `ObjectId` once at connect time when the MongoDB module is already loaded.

## Fragile Areas

**EventHub `waitFor` Promise Never Rejects on Error:**
- Files: `src/adapters/eventhub.adapter.ts` (lines 187-265)
- Why fragile: The `waitFor` implementation (lines 195-265) uses `resolve()` for both success AND failure paths (via `this.failResult()`). If `processError` fires, it calls `resolve(this.failResult(...))` — the promise resolves, not rejects. Error handling in `StepExecutor` catches thrown errors only; a resolved `failResult` is treated as a successful step.
- Safe modification: Change `processError` to call `reject(error)` and handle the rejected promise in the `execute()` method. Alternatively, check `adapterResult.success` after the call returns.
- Test coverage: No unit tests exist for the EventHub adapter.

**TypeScript Test Module Cache Clearing:**
- Files: `src/core/ts-loader.ts` (lines 180-185)
- Why fragile: `clearModuleCache` only works for CommonJS (`require.cache`). In ESM environments, clearing the module cache is not possible via this mechanism. Re-running a TypeScript test multiple times in the same process may load stale module instances.
- Safe modification: Treat this as a known limitation for hot-reload scenarios. Document that TS test file re-execution in the same process may use cached definitions.
- Test coverage: None.

**Parallel Test Execution with Shared `currentTest` / `currentPhase` State:**
- Files: `src/core/test-orchestrator.ts` (lines 88-93, 227-228, 396, 430-431)
- Why fragile: `this.currentTest`, `this.currentPhase`, and `this.currentTestIndex` are instance-level mutable state on `TestOrchestrator`. When `parallel > 1`, multiple tests run concurrently via `pLimit`, and these fields are read/written by concurrent test executions. Event emissions using `this.currentTest?.name` may report the wrong test name.
- Safe modification: Pass test/phase context explicitly to helper methods rather than storing it on `this`. The `emit()` call for events should receive the context directly from the local execution scope.
- Test coverage: None.

**Hook Path Resolution Uses Raw Config String:**
- Files: `src/core/test-orchestrator.ts` (line 506)
- Why fragile: `await import(hookPath)` where `hookPath` is a raw string from config. If the path is relative, Node.js resolves it relative to the current working directory at the time of import, which may differ from the config file's directory.
- Safe modification: Resolve `hookPath` relative to the config file's directory using `path.resolve(path.dirname(configFilePath), hookPath)` before importing.
- Test coverage: None.

## Scaling Limits

**`parallel` Configuration Cap Not Enforced:**
- Current capacity: `pLimit` enforces the configured parallelism level.
- Limit: No upper bound is validated on the `parallel` option. A user could set `parallel: 1000` and attempt 1000 concurrent database connections, exhausting connection pool limits.
- Scaling path: Add a maximum cap for `parallel` (e.g., 50) and emit a warning when exceeded. Document recommended values per adapter type.

**No Connection Pool Size Validation:**
- Current capacity: PostgreSQL pool defaults to `max: 5` (`src/adapters/adapter-registry.ts`). Redis and MongoDB do not configure explicit pool limits.
- Limit: If `parallel` exceeds the PostgreSQL pool max, tests will queue waiting for connections, causing timeout failures with no clear error message.
- Scaling path: Validate that `parallel <= poolMax` for PostgreSQL. Log a warning if the configured parallelism may exceed connection pool capacity.

## Dependencies at Risk

**`minimatch` is an `optionalDependency`:**
- Risk: `minimatch` is listed as optional (`package.json`), but test discovery silently falls back to a simple regex-based `simpleMatch` function when it is absent. The fallback does not support full glob syntax (`**`, character classes, etc.).
- Impact: Tests may not be discovered correctly if `minimatch` is not installed and users use advanced glob patterns in `e2e.config.yaml`.
- Migration plan: Either make `minimatch` a required `dependency` or document the fallback limitations explicitly. Alternatively, use Node.js built-in `fs.glob` (available from Node 22+) to eliminate the dependency.

**`ajv` is Optional but Config Validation Silently Skips on Absence:**
- Risk: Schema validation in `src/core/config-loader.ts` (lines 127-156) is silently skipped if `ajv` is not installed. An invalid config file will proceed past schema validation and fail later with a less helpful error.
- Impact: Users who skip installing optional dependencies lose all config/test schema validation silently.
- Migration plan: Log a warning when `ajv` is absent so users are aware validation was skipped.

**`ts-node` Not Listed as a Dependency:**
- Risk: TypeScript test loading via `src/core/ts-loader.ts` attempts to register `ts-node` at runtime. `ts-node` is in `devDependencies` only and not in `peerDependencies`, so library consumers who want TypeScript tests must discover this requirement independently.
- Impact: TypeScript test files will fail to load in environments without `ts-node` or `tsx`, with no informative error pointing to the missing peer.
- Migration plan: Add `ts-node` and/or `tsx` to `peerDependenciesMeta` as optional peer dependencies with a clear documentation note.

## Missing Critical Features

**No Unit or Integration Test Suite:**
- Problem: `package.json` `test` script is `echo "No tests yet"`. There are no unit tests for core logic (`variable-interpolator`, `step-executor`, `test-orchestrator`, assertion engines) and no integration tests for adapters beyond the YAML e2e test files.
- Blocks: Regression safety for refactoring any core module. CI/CD cannot verify correctness of changes before release.

**Test Dependency Ordering Not Implemented:**
- Problem: `depends` field is accepted in test definitions and parsed, but `sortTestsByDependencies` is never called in the execution pipeline. Tests always execute in file-system discovery order.
- Blocks: Tests that share state across test files via setup/teardown cannot be reliably ordered.

## Test Coverage Gaps

**All Core Logic:**
- What's not tested: `src/core/variable-interpolator.ts`, `src/core/step-executor.ts`, `src/core/test-orchestrator.ts`, `src/core/context-factory.ts`, `src/core/config-loader.ts`, `src/core/yaml-loader.ts`, `src/core/ts-loader.ts`
- Files: Entire `src/core/` directory
- Risk: Any refactoring or bug fix in core execution logic has no safety net.
- Priority: High

**All Adapters:**
- What's not tested: `src/adapters/http.adapter.ts`, `src/adapters/postgresql.adapter.ts`, `src/adapters/mongodb.adapter.ts`, `src/adapters/redis.adapter.ts`, `src/adapters/eventhub.adapter.ts`
- Files: Entire `src/adapters/` directory
- Risk: Adapter-specific bugs (e.g., EventHub failResult issue, Redis KEYS blocking) cannot be caught by CI.
- Priority: High

**Assertion Engine:**
- What's not tested: `src/assertions/matchers.ts`, `src/assertions/assertion-runner.ts`, `src/assertions/jsonpath.ts`
- Files: Entire `src/assertions/` directory
- Risk: JSONPath evaluation edge cases, regex assertions, and numeric comparisons may silently produce incorrect results.
- Priority: High

**CLI Commands:**
- What's not tested: `src/cli/run.command.ts`, `src/cli/validate.command.ts`, `src/cli/list.command.ts`, `src/cli/health.command.ts`
- Files: Entire `src/cli/` directory
- Risk: CLI option parsing, filter logic, and error handling paths are unverified.
- Priority: Medium

---

*Concerns audit: 2026-03-02*
