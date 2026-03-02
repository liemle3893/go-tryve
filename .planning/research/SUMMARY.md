# Project Research Summary

**Project:** E2E Runner — Battle-Tested Milestone (v1.3.0)
**Domain:** API and database E2E testing framework (Node.js/TypeScript)
**Researched:** 2026-03-02
**Confidence:** HIGH

## Executive Summary

This is a brownfield milestone for an existing E2E testing framework at v1.2.1. The framework has solid architecture — layered CLI/Core/Adapters/Reporters, a YAML-first DSL, and support for HTTP, PostgreSQL, MongoDB, Redis, and Azure EventHub — but it shipped with several critical silent failures that make it untrustworthy for real-world use. The primary work is not building new surface area: it is making every existing feature correct, adding Kafka message queue support as a stated project requirement, and building a Vitest-based unit test suite to prevent regressions.

The recommended approach is to fix bugs in strict dependency order before adding new features. Three critical silent failures must be addressed first: the assertion engine is a no-op stub (assertions never run), `continueOnError` misreports failed steps as `passed`, and EventHub errors resolve instead of reject (hiding all infrastructure failures). These fixes unlock a correct baseline. The Kafka adapter must be built after the EventHub promise rejection fix — not before — because copying the EventHub pattern without fixing it would replicate the same critical bug into the new adapter. The unit test suite should be written last, after the fixes are in, to achieve real coverage rather than coverage of broken stubs.

The key risk is assertion dual-implementation drift: the codebase already has two assertion paths (shared `assertion-runner.ts` and per-adapter inline logic), and wiring the third path in `StepExecutor` without removing the per-adapter duplicates creates three divergent assertion paths that silently disagree on edge cases. The second major risk is Kafka consumer startup timing: KafkaJS consumers must be subscribed with `fromBeginning: true` and subscribed before messages are published, or tests will miss messages in CI environments. Both risks are well-understood and preventable with explicit conventions enforced at implementation time.

---

## Key Findings

### Recommended Stack

Two new additions are needed; the rest of the stack is correct and unchanged. For Kafka, use `kafkajs@2.2.4` — the only pure-JavaScript Kafka client that fits this project's peer dependency model. The Confluent client (`@confluentinc/kafka-javascript`) compiles native C++ via `node-pre-gyp` on every install, which breaks Alpine Docker and CI environments; it is disqualified for an npm-distributed testing framework. KafkaJS's lack of stable releases since 2023 is acceptable risk for a test infrastructure use case (not production message processing), and 1.9M weekly downloads confirm continued broad adoption. For unit tests, use `vitest@^3.2.4` — Vitest 4 requires Node >=20 and would break the project's `node >=18` engine guarantee. Vitest 3 includes TypeScript-native transforms, built-in Chai assertions, and V8 coverage — no additional assertion or coverage libraries needed.

**Core technologies (new additions):**
- `kafkajs@2.2.4`: Kafka adapter peer dependency — only pure-JS Kafka client, no native compilation, 1.9M weekly downloads, bundled TypeScript types
- `vitest@^3.2.4`: Unit test runner (devDependency only) — Node 18 compatible, TypeScript-native, built-in Chai assertions, V8 coverage included
- `@vitest/coverage-v8@^3.2.4`: Test coverage (devDependency) — must match Vitest version exactly; zero config, no Istanbul transform

**Critical version constraints:**
- Do NOT use `vitest@4.x` — requires Node >=20, breaks `node >=18` engine constraint
- Do NOT use `@confluentinc/kafka-javascript` as peer dep — native C++ compilation incompatible with peer dep model
- `kafkajs` must be added as `peerDependencies` with `optional: true` in `peerDependenciesMeta`, matching the existing `pg`/`mongodb`/`ioredis` pattern

### Expected Features

This is not a greenfield MVP. The question is: what is the minimum set of fixes and additions that makes the framework trustworthy for real-world use?

**Must have — P0 (table stakes currently broken):**
- Assertion engine wired — `StepExecutor.validateAssertions` calls `assertion-runner.ts`; without this, the entire `assert` surface is silently inert
- `continueOnError` reports `'warned'` not `'passed'` — CI tooling cannot trust test results while failures masquerade as passes
- `retryCount` aggregated at test level — retry-count visibility is expected in any framework claiming retry semantics
- Test dependency ordering enforced — `sortTestsByDependencies` exists and is correct but is never called in `run.command.ts`
- EventHub promise rejection fix — `processError` must call `reject(error)`, not `resolve(this.failResult(...))`
- Kafka adapter — stated project requirement; produce/consume/waitFor actions on KafkaJS
- Unit test suite (Vitest) — the mechanism preventing regressions across all other fixes

**Should have — P1 (competitive differentiators):**
- Parallel execution state safety — fix shared `currentTest`/`currentPhase` mutable state in `TestOrchestrator`; triggers when `parallel > 1`
- Unified capture logic — consolidate dead `captureValues` in `StepExecutor` with `BaseAdapter` implementation; unblocks cross-adapter capture chains
- Parallel config validation — warn when `parallel` exceeds PostgreSQL pool size; prevents silent connection-exhaustion timeouts
- Metadata cache — cache after first parse in `run.command.ts`; performance for large suites

**Defer — P2/P3 (not blocking real-world use):**
- `notEquals`/`notContains` assertion operators — fill gap in `BaseAssertion` interface; defer until user reports the gap
- `ts-node` peer dep discoverability improvement
- GraphQL, gRPC, WebSocket adapters, GUI dashboard — explicitly out of scope; documented anti-features

### Architecture Approach

The existing layered architecture (CLI → Core → Adapters → Reporters) is the right shape; no structural changes are needed. All fixes are localized disconnects within the existing skeleton. The build order is strictly determined by data dependencies: `types.ts` changes first (add `'kafka'` to `AdapterType`, add `'warned'` to `StepStatus`), then `step-executor.ts` and `test-orchestrator.ts` fixes (independent of each other), then `kafka.adapter.ts` (requires EventHub fix and types.ts changes), then `adapter-registry.ts` wiring, then `run.command.ts` dependency sort wiring. Unit tests are written last, after fixes are in.

**Major components and their fix targets:**
1. `StepExecutor` — wire `validateAssertions` stub to `assertion-runner.ts`; fix `continueOnError` to return `'warned'` status
2. `TestOrchestrator` — eliminate shared mutable `currentTest`/`currentPhase` instance fields; pass context explicitly through call chain
3. `test-discovery.ts` — add `sortUnifiedTestsByDependencies()` variant that operates on `UnifiedTestDefinition[]` (post-load, where `depends` field is available)
4. `eventhub.adapter.ts` — change `processError` from `resolve(failResult)` to `reject(error)`
5. `kafka.adapter.ts` (new) — `BaseAdapter` subclass with `produce`/`consume`/`waitFor` actions; consumer wrapped in Promise+timeout pattern; `fromBeginning: true` as default for test subscriptions
6. `run.command.ts` — wire `sortUnifiedTestsByDependencies` after `loadTestDefinitions()`; add `adapterResult.success` check after adapter execute
7. All four reporters — render `'warned'` status distinctly (amber/yellow in console/HTML; `<skipped>` with message in JUnit XML)

### Critical Pitfalls

1. **Assertion dual-implementation drift** — Wiring `StepExecutor.validateAssertions` without removing per-adapter inline assertion logic creates three divergent paths. Prevention: after wiring `runAssertion` in StepExecutor, audit `eventhub.adapter.ts` lines 352-398 and `http.adapter.ts` for inline assertions and remove duplicates. HTTP adapter retains its HTTP-specific assertions (status codes, headers, duration) but delegates JSON body assertions to the shared runner.

2. **`failResult` resolves instead of rejects (EventHub, and potential Kafka)** — `processError` calling `resolve(this.failResult(...))` means all EventHub infrastructure errors appear as `status: 'passed'` to `StepExecutor`. Fix EventHub before scaffolding Kafka — never copy the broken pattern. Also add an `adapterResult.success` check in `StepExecutor.executeStepOnce` so that adapters returning `success: false` without throwing are also caught.

3. **Kafka consumer startup race condition** — `consumer.run()` returns before partition assignment completes. Messages published during the rebalancing window are missed (especially in CI where startup is faster). Prevention: always use `fromBeginning: true` in the Kafka adapter's `waitFor` and `consume` actions; subscribe before publishing in test YAML; generate unique `groupId` per test step using `${uuid()}`.

4. **Parallel shared mutable state in TestOrchestrator** — `this.currentTest` and `this.currentPhase` are overwritten by concurrent `runTest()` calls, causing events to carry wrong test names. Prevention: thread `testName`, `testIndex`, and `phaseName` as explicit parameters; never store them on `this`. Verify with a unit test running 5 concurrent tests and asserting each `test:start` event carries the correct name.

5. **`vi.mock` not intercepting dynamic `require()` calls** — Vitest's `vi.mock` hoisting only intercepts static `import` statements, not `require()`. `test-discovery.ts` uses `require('minimatch')`. Prevention: convert `require('minimatch')` to `await import('minimatch')` in the code-fix phase, before the unit test phase. For adapter peer deps (`pg`, `mongodb`), use `vi.doMock()` + `vi.resetModules()` for dynamic import mocking.

---

## Implications for Roadmap

Based on combined research, the build order is strictly determined by fix dependencies. Five phases are recommended.

### Phase 1: Foundation Fixes (Assertion Engine + Status Correctness)

**Rationale:** The assertion engine stub and `continueOnError` status bug are the highest-leverage fixes — without them, the entire test framework is unreliable (assertions never run, failures masquerade as passes). These fixes have no external dependencies and unblock everything else. `types.ts` changes needed for `'warned'` status and `'kafka'` adapter type also land here.

**Delivers:** A framework where assertions actually execute, test results accurately reflect reality, and `retryCount` is visible at the test level.

**Addresses (from FEATURES.md):** Assertion engine wired, `continueOnError` `'warned'` status, `retryCount` aggregation, TypeScript adapter type fix

**Avoids (from PITFALLS.md):** Assertion dual-implementation drift (audit and remove per-adapter inline logic during this phase); `continueOnError` status incoherence cascading to reporters

**Build order within phase:**
1. `types.ts` — add `'warned'` to `StepStatus`, add `'kafka'` to `AdapterType`, add `KafkaAdapterConfig`
2. `assertion-runner.ts` — no changes needed; already complete
3. `step-executor.ts` — wire `validateAssertions`; fix `continueOnError` to `'warned'`; fix `retryCount` aggregation
4. All reporters (console, JUnit, HTML, JSON) — render `'warned'` distinctly
5. `eventhub.adapter.ts` — fix `processError` → `reject(error)`; fix `resolve(failResult)` pattern

**Research flag:** Standard patterns. No additional phase research needed — all fix targets are precisely identified in ARCHITECTURE.md with line numbers.

---

### Phase 2: Orchestrator Fixes (Parallel Safety + Dependency Ordering)

**Rationale:** Parallel state corruption in `TestOrchestrator` is independent of Phase 1 and can be developed in parallel with it, but must land before the unit test phase because the fix must be verifiable by unit tests. Dependency ordering fix (`sortUnifiedTestsByDependencies` in `run.command.ts`) is a one-call wire-up with no risk.

**Delivers:** Correct event attribution in parallel runs; test dependency ordering enforced; hook path resolution working from any `cwd`.

**Addresses (from FEATURES.md):** Parallel execution state safety, test dependency ordering enforced, hook path resolution fix

**Avoids (from PITFALLS.md):** Parallel shared mutable state corruption (Pitfall 3); hook CWD sensitivity (Pitfall 9)

**Build order within phase:**
1. `test-orchestrator.ts` — eliminate `this.currentTest`/`this.currentPhase`; pass context explicitly
2. `test-discovery.ts` — add `sortUnifiedTestsByDependencies()` for `UnifiedTestDefinition[]`
3. `run.command.ts` — wire `sortUnifiedTestsByDependencies` after `loadTestDefinitions()`; add `adapterResult.success` check in `StepExecutor`

**Research flag:** Standard patterns. Topological sort algorithm is confirmed correct and already implemented in `test-discovery.ts`. Explicit context threading is a well-documented parallel safety pattern.

---

### Phase 3: Kafka Adapter

**Rationale:** Kafka adapter requires Phase 1 to be complete first — specifically the EventHub `resolve(failResult)` fix must land before Kafka scaffolding to prevent inheriting the same bug. Kafka also requires the `types.ts` changes from Phase 1 (`AdapterType` and `KafkaAdapterConfig`). This is the primary new feature of the milestone.

**Delivers:** Message queue testing support — produce messages to Kafka topics, consume with timeout, assert on message content via JSONPath.

**Addresses (from FEATURES.md):** Kafka adapter (produce/consume/waitFor), distinct `'warned'` status in Kafka error paths

**Avoids (from PITFALLS.md):** Kafka consumer startup race (Pitfall 4) — `fromBeginning: true` as default, subscribe-then-publish pattern; Kafka test isolation (Pitfall 5) — unique `groupId` per test via `${uuid()}`, `deleteGroup` in teardown

**Build order within phase:**
1. `kafka.adapter.ts` (new) — `BaseAdapter` subclass; `produce`/`consume`/`waitFor` actions; Promise+timeout consumer pattern; `fromBeginning: true` default
2. `adapter-registry.ts` — add `'kafka'` instantiation path
3. YAML documentation/examples — document subscribe-then-publish ordering, `groupId: "e2e-${uuid()}"` pattern

**Research flag:** Needs focused implementation care. The KafkaJS consumer lifecycle (subscribe → run → resolve/disconnect) is MEDIUM confidence from official docs. The Promise+timeout cancellation pattern for test-mode consumers needs careful validation. Consider writing a Kafka adapter integration test first before the full unit test suite.

---

### Phase 4: Unit Test Suite (Vitest)

**Rationale:** The unit test suite must be written after all fixes are in place — writing it earlier would produce coverage of broken stubs, not real behavior. With Phases 1-3 complete, the suite can verify actual correct behavior and serve as a regression safety net for all future changes. This is also the highest time-investment item (HIGH complexity) but the highest-leverage long-term investment.

**Delivers:** 85%+ coverage on core modules; regression safety for all Phase 1-3 fixes; contributor confidence for refactoring.

**Addresses (from FEATURES.md):** Unit test suite with Vitest, `85%+` coverage on `variable-interpolator`, `assertion-runner`, `step-executor`, `test-orchestrator`, each adapter

**Avoids (from PITFALLS.md):** `vi.mock` vs `require()` issue (Pitfall 7) — convert `require('minimatch')` to dynamic `import` in Phase 1/2; Vitest CJS interop for dynamic peer dep mocking (Pitfall 8) — use `vi.doMock()` + `vi.resetModules()` for adapters

**Build order within phase:**
1. `vitest.config.ts` — configure `environment: 'node'`, `coverage.provider: 'v8'`, exclude `dist/`, `tests/`, `bin/`
2. Core module tests: `variable-interpolator`, `assertion-runner`, `matchers`, `jsonpath`
3. Execution tests: `step-executor`, `test-orchestrator` (parallel state verification with concurrent test runs)
4. Adapter tests: `http`, `postgresql`, `mongodb`, `redis`, `eventhub`, `kafka`
5. CLI/utility tests: `test-discovery`, `run.command`, `context-factory`

**Research flag:** Needs adapter mock strategy documented before writing adapter tests. The `vi.doMock()` approach for dynamic peer dep imports needs a scaffolded example for the team to follow.

---

### Phase 5: P1 Polish (Unified Capture + Performance Guards)

**Rationale:** These are correctness-under-load and performance improvements. They do not block the "battle-tested" baseline but meaningfully improve reliability for teams running large or complex suites. Lower complexity than earlier phases.

**Delivers:** Cross-adapter value capture chains; metadata caching for fast filter operations; connection pool overflow warnings.

**Addresses (from FEATURES.md):** Unified capture logic, metadata cache, parallel config validation

**Avoids (from PITFALLS.md):** Sequential metadata loading performance trap; uncapped `parallel` config causing silent PostgreSQL connection exhaustion

**Research flag:** Standard patterns. Skip phase research. Consolidating two `captureValues` implementations requires a careful audit but no external research.

---

### Phase Ordering Rationale

- **Phase 1 must come first** because the assertion engine stub and `continueOnError` status bugs make the framework's output untrustworthy. Every subsequent phase builds on working assertions.
- **Phase 2 can overlap with Phase 1** in development time (they are independent), but both must complete before Phase 4 (unit tests verify the fixes).
- **Phase 3 (Kafka) must follow Phase 1** strictly because `kafka.adapter.ts` must not be scaffolded from `eventhub.adapter.ts` before the `resolve(failResult)` bug is fixed.
- **Phase 4 (unit tests) must follow Phases 1-3** — tests written against broken stubs produce false confidence.
- **Phase 5 is independent** of Phase 4 and can be developed concurrently once Phases 1-3 are complete.

### Research Flags

**Needs focused implementation attention:**
- **Phase 3 (Kafka adapter):** KafkaJS consumer lifecycle (subscribe → run → timeout cancel) is MEDIUM confidence. The Promise+disconnect cancellation pattern for test-mode consumers needs validation against real KafkaJS behavior. Kafka startup race with `consumer.run()` before partition assignment is a documented known issue (KafkaJS GitHub #1629).
- **Phase 4 (unit tests):** Vitest dynamic import mocking strategy for adapter peer deps requires a proven pattern established before adapter tests are written. `vi.doMock()` + `vi.resetModules()` is the approach but needs a working example first.

**Standard patterns (skip additional research):**
- **Phase 1:** All fix targets are precisely identified with file paths and line numbers. No ambiguity.
- **Phase 2:** Topological sort and explicit context threading are well-documented. Existing `sortTestsByDependencies` implementation is correct; only needs a new variant for `UnifiedTestDefinition[]`.
- **Phase 5:** Consolidation and guard logic. Established patterns throughout.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Verified via `npm info` for exact engine constraints; Confluent native-compile requirement confirmed from `package.json scripts.install`; version incompatibilities confirmed with exact npm data |
| Features | HIGH | Based on direct codebase inspection of all source files; bugs verified at line-number precision; feature prioritization grounded in real code state, not speculation |
| Architecture | HIGH | All fix targets identified with exact file paths, function names, and line numbers from direct source analysis; Kafka adapter pattern verified against official KafkaJS docs |
| Pitfalls | HIGH (code pitfalls) / MEDIUM (Kafka timing) | Code pitfalls are grounded in direct codebase evidence; Kafka consumer startup race is documented in KafkaJS GitHub but exact timing behavior in test environments is MEDIUM confidence |

**Overall confidence:** HIGH

### Gaps to Address

- **KafkaJS consumer cancellation pattern:** The Promise+`consumer.disconnect()` cancellation approach for `waitFor` and `consume` actions needs validation. KafkaJS's behavior when `disconnect()` is called while `run()` is active is documented but requires an integration test to confirm in practice. Address in Phase 3 by writing a Kafka integration test before full implementation.

- **Kafka 4.x compatibility:** KafkaJS 2.2.4 has not had a stable release since 2023. Compatibility with Kafka 4.x is unconfirmed. Acceptable risk for current milestone (Kafka 3.x is the common production version), but should be tracked as a known future migration point if KafkaJS remains unmaintained.

- **Reporter `'warned'` status rendering:** JUnit XML has no native `warned` status — the recommended mapping is `<skipped>` with a message body. This is the closest standard analog, but CI systems that parse JUnit XML may interpret `skipped` differently. Validate with GitHub Actions JUnit reporter during Phase 1.

- **`adapterResult.success` check coverage:** StepExecutor must check `adapterResult.success === false` after adapter execute, not only catch thrown errors. This contract needs to be documented in `BaseAdapter` with explicit guidance on when to throw vs. return `failResult`. Address in Phase 1 alongside the EventHub fix.

---

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: all `src/**/*.ts` files — bug locations identified with line numbers, 2026-03-02
- `npm info kafkajs version`, `npm info vitest@3.2.4 engines`, `npm info vitest engines` — version compatibility confirmed
- `npm info @confluentinc/kafka-javascript scripts.install` — native compile confirmed
- KafkaJS official docs: [Consuming Messages](https://kafka.js.org/docs/consuming), [Getting Started](https://kafka.js.org/docs/getting-started)
- `.planning/PROJECT.md`, `.planning/codebase/CONCERNS.md` — project intent and known concerns

### Secondary (MEDIUM confidence)
- [KafkaJS GitHub #1629](https://github.com/tulios/kafkajs/issues/1629) — consumer startup race condition
- [Vitest blog: vitest-3](https://vitest.dev/blog/vitest-3) — Node 18 support, download stats
- [Vitest module mocking guide](https://vitest.dev/guide/mocking/modules) — `vi.mock` hoisting behavior
- [KafkaJS GitHub #1603](https://github.com/tulios/kafkajs/issues/1603) — maintenance status
- [Soft Assertions with AssertJ | Baeldung](https://www.baeldung.com/java-assertj-soft-assertions) — `'warned'` status industry pattern
- [JUnit 5 Parallel Execution: Thread-Safe Design Patterns](https://programgeeks.net/junit-5-parallel-execution-thread-safe-test-design-patterns/) — context passing vs shared mutable state
- [Kafka testing isolation strategies](https://www.signadot.com/blog/testing-microservices-message-isolation-for-kafka-sqs-more) — unique consumer group pattern

### Tertiary (LOW confidence)
- [Confluent kafka-javascript Alpine issue #48](https://github.com/confluentinc/confluent-kafka-javascript/issues/48) — musl/Alpine segfault; verifies native compile risk but issue-tracker evidence only

---
*Research completed: 2026-03-02*
*Ready for roadmap: yes*
