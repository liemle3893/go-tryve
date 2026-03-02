# Pitfalls Research

**Domain:** E2E testing framework — bug-fixing, assertion engine centralization, Kafka adapter, unit test retrofit
**Researched:** 2026-03-02
**Confidence:** HIGH (assertions/parallel/unit test areas grounded in codebase evidence); MEDIUM (Kafka consumer timing grounded in official kafkajs issues)

---

## Critical Pitfalls

### Pitfall 1: Assertion Engine Dual-Implementation Drift

**What goes wrong:**
The codebase already has TWO assertion paths: (1) `assertion-runner.ts` (`runAssertion`) — the shared engine — and (2) per-adapter inline assertion logic duplicated inside `http.adapter.ts`, `eventhub.adapter.ts`, and others. If `StepExecutor.validateAssertions` is wired to `runAssertion` without also removing or delegating to the per-adapter implementations, the system ends up with three paths, and each can diverge on edge cases (e.g., how `null` vs `undefined` is handled in `equals`, whether `matches` wraps regex in try/catch). Bugs fixed in one path silently go unfixed in the others.

**Why it happens:**
The stub at `StepExecutor` lines 240-250 deferred assertion work to "Phase 5" while each adapter kept its own copy. When the stub is now replaced with real calls to `runAssertion`, there is a natural temptation to leave the per-adapter assertions in place "just in case," rather than verifying and removing them. The redundancy feels safe but is not.

**How to avoid:**
Wire `StepExecutor.validateAssertions` to `runAssertion` and then audit each adapter's `execute()` method to determine whether it runs its own assertion logic (`eventhub.adapter.ts` lines 352-398, `http.adapter.ts`). For any adapter that already calls `runAssertion` via `interpolatedParams.assert`, no change is needed. For any adapter with hand-rolled inline assertion logic, migrate it to use `runAssertion` or explicitly delete it with a comment explaining why the centralized path covers it. Do not leave both active.

**Warning signs:**
- A test with `matches: "[invalid regex"` throws from one adapter but silently passes from another
- `equals` comparison against a MongoDB ObjectId returns different results depending on whether the step uses the `mongodb` or `http` adapter
- The `continueOnError` fix lands and assertion failures in EventHub steps are still silently resolved as success (because `failResult` resolves, not rejects — see Pitfall 2)

**Phase to address:** Assertion engine phase (whichever phase centralizes `validateAssertions`)

---

### Pitfall 2: `failResult` Pattern Masks Errors That Must Surface as Failures

**What goes wrong:**
`BaseAdapter.failResult()` returns an `AdapterStepResult` with `success: false`. But in `EventHub.waitFor()` and `EventHub.consume()`, both `processError` and the timeout handler call `resolve(this.failResult(...))` — they resolve the promise with a failure object instead of rejecting. `StepExecutor` only catches thrown errors; it does not inspect `adapterResult.success` after `withRetry` completes. The result is that a network error or subscription error in EventHub causes the step to report `status: 'passed'` with no error attached — because `StepExecutor` calls `this.createStepResult(step, 'passed', ...)` on the returned value.

When a Kafka adapter is built following the EventHub pattern, this same bug will be replicated unless the pattern is corrected first.

**Why it happens:**
The `failResult` helper was designed to return a structured error for cases where the adapter wants to signal failure without throwing (e.g., assertion-level failures that should be reported but not crash the process). The intent is correct but the contract is ambiguous — callers (StepExecutor) never check `adapterResult.success`. Using `resolve(failResult)` for actual infrastructure errors (socket errors, subscription errors) violates the contract of the Promise-based retry wrapper.

**How to avoid:**
Fix the `processError` handlers in EventHub to call `reject(error)` not `resolve(this.failResult(error))`. Then add a check in `StepExecutor.executeStepOnce` (or in `withRetry`) to throw if `adapterResult.success === false` after the adapter returns. This makes the contract explicit: an adapter that wants to signal assertion failure should throw `AssertionError`, not return `failResult`. Build the Kafka adapter after this fix is in place — never after.

**Warning signs:**
- An EventHub health check fails but the subsequent test step shows `status: passed`
- The Kafka adapter is scaffolded by copying `eventhub.adapter.ts` and inheriting the `resolve(failResult)` pattern
- No test exercises a subscription error path because there are no unit tests

**Phase to address:** EventHub bug fix phase (before Kafka adapter phase — Kafka must not inherit this pattern)

---

### Pitfall 3: Parallel Test Execution Corrupts Event Emission via Shared Instance State

**What goes wrong:**
`TestOrchestrator` stores `this.currentTest`, `this.currentPhase`, and `this.currentTestIndex` as mutable instance fields. When `parallel > 1`, `pLimit` runs multiple calls to `this.runTest()` concurrently on the same orchestrator instance. Event emissions inside `runPhase()` read `this.currentTest?.name` — but by the time the `phase:start` or `step:start` event fires, another concurrent test execution may have already overwritten `this.currentTest`. Reporters receive events tagged with the wrong test name, producing misleading output in parallel runs. The `currentTestIndex` increment at line 228 is also a non-atomic read-increment-write on a shared field, which can produce duplicate or skipped index values.

**Why it happens:**
Instance-level state tracking is a natural first implementation when tests run sequentially. The parallel path was added (via `pLimit`) without refactoring state tracking from instance fields to local scope. Because the bug only manifests under parallel execution and produces wrong-test-name events (not crashes), it is invisible in sequential CI runs.

**How to avoid:**
Pass test and phase context explicitly into `runPhase` and all emit calls rather than reading from `this.currentTest`. The local variable `const test = ...` already exists in `runTest` scope — use it directly in the event payload. Replace `this.currentTestIndex++` with a closure over the local index captured at call time. Make `currentTest`/`currentPhase`/`currentTestIndex` local to each invocation of `runTest`, not on `this`. After this fix, add a unit test that runs 5 tests concurrently and verifies each `test:start` event carries its correct test name.

**Warning signs:**
- Console reporter shows "Test A: passed" events interleaved with wrong step names when `parallel: 3` is configured
- `test:start` events show the same test name twice in parallel runs
- CI shows different reporter output when `parallel: 1` vs `parallel: 4` on the same test suite

**Phase to address:** Parallel execution fix phase (before unit test phase — the fix must be verifiable by unit tests written in the next phase)

---

### Pitfall 4: KafkaJS Consumer Startup Race Condition Loses Messages

**What goes wrong:**
In KafkaJS, `consumer.run()` resolves before the consumer has completed Kafka consumer group rebalancing and partition assignment. If a test produces messages to a topic and then immediately awaits consumption, messages sent during the rebalancing window are never delivered to the consumer — they are either missed (if `fromBeginning: false`) or read from a stale offset. This is a documented KafkaJS issue (GitHub issue #1629) that appears reliably in CI where startup is faster and the race window is tighter.

**Why it happens:**
The consumer group coordinator in Kafka requires a join-group / sync-group round trip before the consumer is assigned partitions. KafkaJS does not expose a "ready" event or a way to await partition assignment completion. Developers see `await consumer.run()` return and assume the consumer is ready, then publish test messages and wonder why the assertion for received messages times out.

**How to avoid:**
In the Kafka adapter's `waitFor` action, always set `fromBeginning: true` on subscribe so offset re-calculation is not a factor. Add an explicit delay between `consumer.run()` and the test's publish step, or (better) implement a `seekToBeginning` approach using the `admin` client to reset offsets before subscribing. The cleanest test pattern is: subscribe with `fromBeginning: true`, then publish, then wait. Do not subscribe after publishing. Document this constraint in the Kafka adapter's YAML test examples.

**Warning signs:**
- Tests pass locally (where Docker startup is slower) but fail in CI (where Kafka starts faster and the race window shrinks to zero)
- Increasing the `waitFor` timeout doesn't fix the problem — the messages were never queued for this consumer group
- `fromBeginning: false` (the default) is used in test YAML

**Phase to address:** Kafka adapter phase — encode `fromBeginning: true` as the default for all test-mode subscriptions

---

### Pitfall 5: Kafka Test Isolation Breaks When Consumer Groups or Topics Are Reused

**What goes wrong:**
If multiple test cases use the same Kafka consumer group ID (e.g., `groupId: 'e2e-test'`), Kafka's group coordinator assigns offsets per group. After test A consumes messages and commits offsets, test B (using the same group) starts from the committed offset and misses messages that were published before test B's consumer subscribed. Conversely, uncommitted offsets from test A "leak" into test B. In parallel execution (multiple tests running concurrently), two consumers with the same group ID cause a rebalance storm — each new consumer joining the group triggers partition reassignment, which can cause all consumers to briefly stop processing.

**Why it happens:**
Consumer group IDs are typically set once in a configuration file and reused. When tests run sequentially, offset drift is less obvious because each test fully consumes its messages. In parallel runs, the rebalance cascade makes failures appear random and unreproducible.

**How to avoid:**
Generate a unique consumer group ID per test step execution using `${uuid()}` interpolation (which this framework already supports as a built-in). Example YAML: `groupId: "e2e-${uuid()}"`. Use unique topic names per test or per test run where possible. Add a `deleteGroup` action to the Kafka adapter that calls `admin.deleteGroups([groupId])` in the teardown phase, cleaning up the ephemeral consumer group. Document that using a fixed `groupId` across tests is an anti-pattern.

**Warning signs:**
- Tests that pass in isolation fail when the full suite is run
- Increasing `parallel` from 1 to 2 causes previously-passing Kafka tests to fail intermittently
- Rerunning the suite without restarting Kafka causes tests to pass on the second run but fail on the first

**Phase to address:** Kafka adapter phase — make unique `groupId` the default pattern in scaffolded YAML examples

---

## Moderate Pitfalls

### Pitfall 6: `continueOnError` Steps Report `passed` With an Attached Error Object

**What goes wrong:**
The current code at `StepExecutor` lines 137-148 creates a step result with `status: 'passed'` and an `error` field populated. This is logically incoherent — no downstream consumer (reporter, CI system, programmatic API caller) can distinguish a genuine pass from a forgiven failure. JUnit and HTML reporters may render it as green when it should be amber.

**How to avoid:**
Introduce a `'warned'` status in the `StepStatus` union type (in `types.ts`). Use `'warned'` for `continueOnError` failures. Update `allStepsPassed()` in `step-executor.ts` to treat `'warned'` as non-failing. Update all reporters to render `'warned'` distinctly (amber/yellow). Update `createSuiteResult` to count `'warned'` separately. This is a type-system change that cascades — do it early to avoid reporter rework later.

**Warning signs:**
- CI reports 100% pass rate but individual test HTML report shows errors in step detail
- `getFirstFailedStep()` returns undefined even though a step logged a warning about failure

**Phase to address:** `continueOnError` bug fix phase

---

### Pitfall 7: Retrofitting Unit Tests on Code With Hardcoded Side Effects

**What goes wrong:**
`StepExecutor`, `TestOrchestrator`, and the adapter classes construct their dependencies internally or use module-level imports. When adding Vitest unit tests to these classes, tests cannot inject mock adapters or mock loggers without refactoring the constructor signatures. Additionally, Vitest's `vi.mock()` only intercepts ES `import` statements — it does not intercept `require()` calls. The `require('minimatch')` call in `test-discovery.ts` cannot be mocked via `vi.mock`, requiring either a code change (`await import('minimatch')`) or a workaround using `vi.stubGlobal`.

**How to avoid:**
Before writing tests, verify that each class under test accepts its dependencies via constructor injection rather than constructing them internally. `StepExecutor` and `TestOrchestrator` already do this (adapters, logger, options are injected). Fix the `require('minimatch')` call to `await import(...)` during the code-fix phase so that it can be mocked normally in tests. Never add unit tests for a class that still uses synchronous `require()` for a mocked dependency — fix the `require` first.

**Warning signs:**
- A Vitest test file has `vi.mock('minimatch')` but the mock is not applied because the code path uses `require('minimatch')`
- Test setup creates a full `LoadedConfig` object to test a single method that only needs `config.defaults`

**Phase to address:** Unit test phase — but the `require` → `import` fix must land in the preceding code-fix phase

---

### Pitfall 8: Vitest CJS Interop Breaks `vi.mock` for Optional Peer Dependencies

**What goes wrong:**
The project publishes a CommonJS output (`tsconfig module: commonjs`). When Vitest runs tests against the TypeScript source (pre-compilation), it processes them as ESM via Vite. However, peer dependencies like `pg`, `mongodb`, `ioredis` are loaded via `await import(...)` inside adapter methods. Mocking these dynamic imports with `vi.mock('pg')` may not intercept the dynamic import inside adapter methods because Vitest's static analysis of `vi.mock` hoisting operates differently for dynamic imports vs. top-level static imports. Tests that call `vi.mock('mongodb')` at the top of the file may find the adapter's `await import('mongodb')` inside `connect()` still reaches the real module.

**How to avoid:**
Use `vi.doMock()` (which is not hoisted) immediately before the code path that triggers the dynamic import, combined with `vi.resetModules()` between tests. Alternatively, accept that adapter unit tests require an installed (or mocked at the OS level) peer dependency and use Vitest's `globalSetup` to install a lightweight mock module. Test the adapter logic by passing mock objects directly into the adapter constructor where possible, keeping the dynamic import path as a thin integration seam tested separately.

**Warning signs:**
- `vi.mock('mongodb')` is declared but `jest.fn()` / `vi.fn()` calls inside the mock factory are never invoked during test runs
- Test isolation between adapter tests fails because one test's `vi.mock` bleeds into another test's dynamic import resolution

**Phase to address:** Unit test phase — document the dynamic import mock strategy in the test scaffold before writing adapter tests

---

### Pitfall 9: Hook Path Resolution Silently Breaks Under Monorepo or Different CWD

**What goes wrong:**
`TestOrchestrator.runHook()` calls `await import(hookPath)` where `hookPath` is the raw string from the config. If `hookPath` is relative (e.g., `./hooks/before-all.js`), Node.js resolves it relative to the current working directory at import time — not relative to the config file's location. If `e2e runner` is invoked from a parent directory or from CI with a different `cwd`, the hook fails with `Cannot find module` rather than a helpful error. Worse: if a similarly-named file exists at the resolved path in the different `cwd`, it silently loads the wrong module.

**How to avoid:**
Resolve `hookPath` against the config file directory before importing: `path.resolve(path.dirname(configFilePath), hookPath)`. Pass `configFilePath` into `TestOrchestrator` or use `this.config.configPath` if that is stored. This is a one-line fix but must land before the Kafka adapter phase, since hook-based Kafka consumer group cleanup is a likely pattern.

**Warning signs:**
- Hooks work in local dev but fail in CI (different `cwd`)
- `Cannot find module './hooks/before-all.js'` is the error, not a permissions error

**Phase to address:** Bug fix phase (the same sprint as the other orchestrator fixes)

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Keep per-adapter inline assertions after wiring `runAssertion` | No disruption to existing adapter tests | Three divergent assertion paths; bugs fixed in one not fixed in others | Never — remove duplicates |
| Copy `eventhub.adapter.ts` as the base for `kafka.adapter.ts` | Fast scaffolding | Inherits `resolve(failResult)` bug and custom assertion engine instead of shared `runAssertion` | Never — fix EventHub first |
| Use a fixed `groupId` in Kafka test YAML examples | Simpler initial YAML | Parallel tests fight over consumer group offsets; teardown doesn't clean up | Never in test YAML |
| Add `vi.mock` without fixing `require()` calls first | Can write test file structure immediately | Mocks silently don't apply; tests pass with real modules, hiding bugs | Never — fix `require` first |
| Leave `retryCount: 0` hardcoded after adding unit tests | Tests pass on first attempt | Consumers of test results (CI dashboards) cannot detect retry storms | Acceptable until retry-aware reporting is explicitly required |
| Use `this.currentTest` in event emissions during parallel runs | Simple code | Wrong test names in parallel event stream; corrupts CI reporter output | Never in production code |

---

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Kafka (kafkajs) | Publish messages before consumer is ready (after `consumer.run()` but before partition assignment) | Subscribe with `fromBeginning: true`, then publish; or use admin API to seek before subscribing |
| Kafka (kafkajs) | Reuse the same `groupId` across test cases | Generate `groupId: "test-${uuid()}"` per test; delete group in teardown |
| Kafka (kafkajs) | Set `fromBeginning: false` (default) in test adapter — consumer misses messages published before subscription | Always default to `fromBeginning: true` in the Kafka adapter's `waitFor` action |
| EventHub | `processError` calls `resolve(failResult)` — StepExecutor sees a "passed" result for an infrastructure error | Change `processError` to `reject(error)`; add `adapterResult.success` check in StepExecutor |
| Redis | `KEYS pattern` blocks server on large keysets in `flushPattern` | Replace with `SCAN`-based iteration; applies before any test that does large-scale Redis cleanup |
| MongoDB | `await import('mongodb')` inside `normalizeFilter()` on every call | Import `ObjectId` once at `connect()` time and cache on the adapter instance |
| Optional peer deps (`pg`, `mongodb`) | Test suite fails with `Cannot find module 'pg'` because peer dep isn't installed in test environment | Use `vi.mock()` or install the peer dep in `devDependencies` for the test run |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Sequential metadata loading (YAML parsed twice per test during filtering) | Slow `list` and `run --tag` commands with large test suites | Cache metadata after first parse; reuse when loading full definition | ~50+ test files |
| Uncapped `parallel` config (no upper bound) | PostgreSQL connection pool exhausted; tests timeout with no clear error | Enforce `parallel <= 50` cap; warn when `parallel > pgPoolMax` | When `parallel > 5` with default pg pool |
| HTML reporter builds full HTML string in memory | Out-of-memory for very large suites | Stream HTML output in chunks; acceptable to defer this | ~500+ test results |
| Kafka consumer left running after test timeout | Subsequent tests join the same consumer group and trigger rebalance | Always close consumer in teardown; set `sessionTimeout` to a low value in test config | >1 concurrent Kafka test |

---

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| `new RegExp(userValue)` without try/catch in `assertion-runner.ts` line 114, `matchers.ts` line 326 | ReDoS attack via crafted YAML assertion `matches` value | Wrap `new RegExp(...)` in try/catch; rethrow as `AssertionError` with a helpful message; add pattern length limit |
| `$file(path)` reads any filesystem path including `../../etc/passwd` | Arbitrary file read from test YAML | Restrict `$file()` to paths under `config root` or `test directory`; resolve and validate prefix before reading |
| Full `process.env` passed into interpolation context | Any env var name in test YAML resolves — accidental secret exposure in assertion error messages | Use an explicit allowlist of env vars or a `${env.ALLOWED_VAR}` prefix mechanism |
| Hook path from config loaded via `await import(hookPath)` without validation | Arbitrary module execution if config file is tampered | Resolve hook path to within project root; validate before import |

---

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| `continueOnError` step shows `passed` in reporter | User cannot tell which steps had forgiven failures; false confidence in CI green | Use `'warned'` status with amber color in all reporters |
| `retryCount: 0` always reported at test level | User cannot identify flaky tests that are passing only after retries | Aggregate step retryCount to test level in `buildTestResult` |
| Assertion failure on a `waitFor` Kafka/EventHub step shows `Timeout` not the actual value | User doesn't know what was received vs. what was expected | On timeout, include last-seen event body (if any) in the timeout error message |
| TypeScript test steps shown as adapter type `http` in reporters | User is confused why a TypeScript function shows as an HTTP request in the HTML report | Fix `adapter: 'http'` placeholder to `adapter: 'typescript'` and update reporters to display it correctly |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Assertion engine wired:** `StepExecutor.validateAssertions` calls `runAssertion` — verify that per-adapter inline assertion logic is also removed or consolidated, not left running in parallel
- [ ] **`continueOnError` fix complete:** `'warned'` status is rendered distinctly in ALL four reporters (console, JUnit, HTML, JSON), not just one
- [ ] **EventHub error handling fixed:** `processError` now calls `reject` — verify `StepExecutor` also checks `adapterResult.success` after `withRetry` returns, otherwise the fix only works for thrown errors
- [ ] **Kafka adapter complete:** Consumer group cleanup (`deleteGroups`) is implemented in teardown — verify it runs even when the test fails (i.e., it is in the teardown phase, not execute)
- [ ] **Parallel state fixed:** `this.currentTest` is no longer read in event emissions — verify by running with `parallel: 4` and checking that each event's `testName` matches the test that emitted it
- [ ] **Unit test suite bootstrapped:** Vitest is configured AND at least one test per core module exists — `echo "No tests yet"` is not acceptable as CI signal
- [ ] **Hook path resolution fixed:** `hookPath` is resolved against `config directory` — verify with a relative path and `cwd` set to a parent directory

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Assertion dual-implementation divergence discovered after Kafka phase | HIGH | Audit every adapter's `execute()` for inline assertions; write characterization tests for current behavior before removing duplicates; merge one at a time |
| Kafka consumer group offset drift causes flaky suite in CI | MEDIUM | Add `fromBeginning: true` to all test subscriptions; delete all test consumer groups via admin API; re-run suite |
| Parallel state corruption producing wrong event names in reporters | MEDIUM | Set `parallel: 1` as a temporary workaround; then apply the local-scope fix for `currentTest`/`currentPhase` |
| `resolve(failResult)` bug found after Kafka adapter built on same pattern | HIGH | Fix both EventHub and Kafka simultaneously; add the `adapterResult.success` check to StepExecutor; write regression tests for both |
| `vi.mock` not intercepting dynamic imports in adapter unit tests | LOW | Switch to `vi.doMock` + `vi.resetModules()`; or inject mock dependencies directly into constructor rather than mocking the module |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Assertion dual-implementation drift | Assertion engine fix phase | Each adapter's inline assertions removed; `runAssertion` is the sole assertion path; all existing E2E YAML tests still pass |
| `failResult` resolves instead of rejects | EventHub bug fix phase (BEFORE Kafka phase) | Unit test: publish to EventHub with connection error → step reports `failed`, not `passed` |
| Parallel shared mutable state | Parallel execution fix phase | Integration test: run 5 tests with `parallel: 5`; verify each `test:start` event carries correct test name |
| `continueOnError` status incoherence | `continueOnError` fix phase | JUnit output shows `warned` tests as a distinct category; HTML report shows amber steps |
| Kafka consumer startup race | Kafka adapter phase | E2E test: publish immediately after adapter connect; verify message is consumed |
| Kafka test isolation (consumer group reuse) | Kafka adapter phase | Run full suite twice in sequence without restarting Kafka; both runs must pass |
| `require` blocking `vi.mock` | Code-fix phase (before unit test phase) | `vi.mock('minimatch')` intercepted correctly in test-discovery unit tests |
| Vitest dynamic import mock isolation | Unit test phase | Adapter unit tests run in isolation; one test's mock doesn't leak to another |
| Hook path CWD sensitivity | Bug fix phase | Hook integration test invoked from parent directory; module loads correctly |

---

## Sources

- KafkaJS GitHub issue #1629 — consumer startup race condition: https://github.com/tulios/kafkajs/issues/1629
- Kafka auto.offset.reset use cases and pitfalls (Quix.io): https://quix.io/blog/kafka-auto-offset-reset-use-cases-and-pitfalls
- KafkaJS consuming messages documentation: https://kafka.js.org/docs/consuming
- Kafka testing isolation strategies (signadot.com): https://www.signadot.com/blog/testing-microservices-message-isolation-for-kafka-sqs-more
- Vitest common errors guide: https://vitest.dev/guide/common-errors
- Vitest module mocking guide: https://vitest.dev/guide/mocking/modules
- Vitest — cannot mock modules imported via `require()` (GitHub discussion #3134): https://github.com/vitest-dev/vitest/discussions/3134
- Node.js race conditions (nodejsdesignpatterns.com): https://nodejsdesignpatterns.com/blog/node-js-race-conditions/
- Parallel test execution issues (oneuptime.com): https://oneuptime.com/blog/post/2026-01-24-parallel-test-execution-issues/view
- e2e-runner CONCERNS.md (project codebase): .planning/codebase/CONCERNS.md
- e2e-runner StepExecutor source: src/core/step-executor.ts
- e2e-runner TestOrchestrator source: src/core/test-orchestrator.ts
- e2e-runner EventHubAdapter source: src/adapters/eventhub.adapter.ts

---
*Pitfalls research for: E2E testing framework — bug-fixing milestone*
*Researched: 2026-03-02*
