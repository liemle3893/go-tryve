# Feature Research

**Domain:** API and database E2E testing framework (Node.js, YAML/TypeScript DSL)
**Researched:** 2026-03-02
**Confidence:** HIGH (existing codebase directly inspected; patterns verified against mature frameworks)

---

## Context

This is a brownfield milestone. The framework exists at v1.2.1 with a solid architecture but multiple
incomplete features, silent failures, and zero unit test coverage. The goal is not to add new surface
area — it is to make every existing feature correct, add Kafka support, and build a test safety net.

Research answers: what do mature E2E testing frameworks for APIs/databases consider table stakes, what
distinguishes a battle-tested framework from a prototype, and what should deliberately not be built.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features that users assume exist. Missing or broken = product feels unreliable.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Assertions that actually run | Every assertion in a test YAML must be validated; silent no-ops are a silent trust-destroyer | LOW | Currently broken: `validateAssertions` in StepExecutor is a stub that logs "Phase 5" and does nothing. The assertion engine (assertion-runner.ts) exists and is correct — it just needs to be called |
| Correct pass/fail status on continueOnError | A step that failed and was forgiven is NOT a "passed" step. Reporters and CI must distinguish forgiven failures from genuine passes | LOW | Currently broken: `continueOnError` marks failed steps `'passed'` with an error attached. Industry standard (TestNG SoftAssert, AssertJ SoftAssertions) uses a distinct status — either `'warned'` or keeping `'failed'` but marking it `nonBlocking: true` |
| Accurate retry counts in reports | CI dashboards and flakiness detection depend on seeing how many retries each test consumed | LOW | Currently broken: `retryCount` at the test level is hardcoded to `0`. Step-level counts exist but are never aggregated upward |
| Test dependency ordering enforced | Tests that declare `depends: [OtherTest]` must run after their dependency | MEDIUM | Currently broken: `sortTestsByDependencies` exists in test-discovery.ts but is never called in run.command.ts. Topological sort (Kahn's algorithm) is the standard approach — already implemented, just not wired in |
| Full assertion operator set | Users expect: `equals`, `notEquals`, `contains`, `notContains`, `matches`, `greaterThan`, `greaterThanOrEqual`, `lessThan`, `lessThanOrEqualTo`, `exists`, `notExists`, `isNull`, `isNotNull`, `isEmpty`, `notEmpty`, `type`, `length` | LOW | Operators exist in matchers.ts and assertion-runner.ts. Missing: `notEquals`, `notContains`, `greaterThanOrEqual` (exists in matchers.ts but not in BaseAssertion interface in assertion-runner.ts). Gap: assertion-runner.ts BaseAssertion interface is incomplete relative to the full matchers.ts set |
| Accurate adapter type for TypeScript steps | Steps backed by TypeScript functions must not masquerade as `'http'` adapter steps in reports and filters | LOW | Currently broken: ts-loader uses `adapter: 'http'` as a placeholder. Fix: introduce `'typescript'` as a valid `AdapterType` value |
| Event/message queue errors reject, not resolve | An EventHub/Kafka error path that resolves instead of rejecting makes every error invisible to the orchestrator | LOW | Currently broken: EventHub `processError` calls `resolve(this.failResult(...))`. Fix: call `reject(error)` so the promise rejects and the orchestrator catches it |
| Reliable glob-based test discovery | Tests that match `**/*.test.yaml` must be found even when using character classes or multi-segment patterns | LOW | Currently fragile: `minimatch` is optional, and the fallback `simpleMatch` does not support full glob syntax. Fix: make `minimatch` a required dependency or use Node 22+ built-in `fs.glob` |
| Config and schema validation that warns when skipped | Users must know if their config was not validated due to missing `ajv` | LOW | Currently: skips silently. Fix: emit a `WARN` log when ajv is absent |
| Correct module resolution for relative hook paths | `beforeAll`/`afterAll` hook paths must resolve relative to the config file, not the process cwd | LOW | Currently broken: raw `import(hookPath)` where cwd may differ from the config directory. Fix: `path.resolve(path.dirname(configFilePath), hookPath)` before importing |

### Differentiators (Competitive Advantage)

Features that set this framework apart. Not expected, but genuinely valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Kafka adapter (produce + consume + assert) | Message queue testing is a gap in most YAML-driven E2E frameworks. Produce a message to a topic, consume with timeout/poll, assert on message key/value/headers using JSONPath | HIGH | KafkaJS is the right client (most popular Node.js Kafka client, strong TypeScript support). Pattern: produce → poll with `waitFor` + timeout → assert on received message content. Mirror EventHub adapter structure but fix the promise rejection bug |
| Distinct `warned` / non-blocking step status | Surfaces soft failures in CI without failing the build. Test output says "3 passed, 1 warned" instead of either silently hiding failures or blocking CI unnecessarily | LOW | Implement as a new `StepStatus` value `'warned'` and propagate up to `PhaseStatus` and `TestStatus`. Standard pattern from AssertJ SoftAssertions and TestNG |
| Deep parallel execution safety | Parallel-safe orchestration (no shared mutable `currentTest`/`currentPhase` instance state) means users can confidently set `parallel: 8` without getting misattributed test names in event emissions | MEDIUM | Pass test/phase context explicitly to helper methods instead of storing on `this`. No external library needed — pure refactor |
| Parallel config validation with connection pool guard | Warn users when `parallel` exceeds the PostgreSQL pool size, preventing silent connection-exhaustion timeouts | LOW | Simple validation: `parallel <= poolMax`. Emit a `WARN` log. Document recommended per-adapter values |
| Unified capture logic | Capture values from any adapter result, not just HTTP. Consolidating capture into the step executor layer means MongoDB, PostgreSQL, and Redis results can all feed into `${captured.x}` variable chains | MEDIUM | Remove dead `captureValues` in StepExecutor; consolidate with BaseAdapter's `captureValues`. Requires careful audit of which adapters currently handle capture inline |
| Unit test suite with Vitest | A framework that tests itself is trustworthy. 85%+ coverage on core modules means contributors can refactor with confidence | HIGH | Vitest is the right choice: TypeScript-native, fast, excellent ESM/CJS interop. Priority test targets: variable-interpolator, assertion-runner, step-executor, test-orchestrator, each adapter. This is the single highest-leverage investment |
| Metadata cache to avoid double-parsing | Fast `list`/`run` with tag filters, without paying full-parse cost twice per test file | LOW | Cache metadata after first parse in run.command.ts. Simple Map keyed by file path |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| GraphQL adapter | Teams using GraphQL want native support | GraphQL is HTTP with a structured body. Adding a dedicated adapter adds maintenance surface for no functional gain | Use the existing HTTP adapter with `POST` + JSON body. Document a standard pattern for GraphQL queries and mutations in the YAML DSL |
| gRPC adapter | Teams using gRPC microservices want native support | gRPC requires protobuf compilation, binary transport, and reflection — fundamentally different from the HTTP/DB model. Complexity far exceeds scope | Explicitly out of scope in PROJECT.md. Document as such |
| GUI dashboard | Visual test management seems appealing | This is a CLI-first framework for CI pipelines. A dashboard requires a server, storage, authentication, and a frontend build pipeline — a completely separate product | The HTML reporter already serves local visualization needs. Link to external CI integrations (GitHub Actions summary, JUnit XML) |
| Watch mode with hot reload | Developers want tests to re-run on file save | TS module cache clearing doesn't work in ESM (`require.cache` is CJS-only). Hot reload of test files in the same process reuses stale module instances | Run the CLI as a one-shot command. Use `nodemon ./bin/e2e.js run` as a documented pattern for watch-like behavior without rebuilding module infrastructure |
| Cloud-hosted execution | Teams want results stored and visible in a central dashboard | Out of scope for a local/CI tool. Adds auth, storage, and networking complexity | JUnit XML reporter + CI system (GitHub Actions, Jenkins) provides storage and history |
| Real-time WebSocket test adapter | Event-driven UIs often use WebSockets | WebSocket testing requires persistent connection lifecycle management not compatible with the step-based execution model | Test the downstream effects of WebSocket events via HTTP API state checks instead |
| Automatic test data seeding | Managed test data factories reduce boilerplate | Test data generation is domain-specific. A generic factory would either be too simple (useless) or too complex (its own product) | The `setup` phase with database adapter steps is the right primitive. Document seed patterns |

---

## Feature Dependencies

```
[Assertion engine wired to StepExecutor]
    └──requires──> [assertion-runner.ts runAssertion] (already exists, just not called)

[Kafka adapter]
    └──requires──> [BaseAdapter pattern] (already established)
    └──requires──> [waitFor + timeout pattern] (already in EventHub adapter — replicate and fix)
    └──requires──> [EventHub promise rejection fix] (same underlying bug — fix concurrently)

[Unified capture logic]
    └──requires──> [assertion engine wired] (both live in step executor layer — coordinate changes)

[Distinct 'warned' status]
    └──requires──> [types.ts StepStatus update]
    └──requires──> [reporters updated to display 'warned' distinctly]

[Parallel execution state safety]
    └──requires──> [test-orchestrator.ts refactor] (no new dependencies)

[Unit test suite]
    └──requires──> [assertion engine wired] (tests verify correct assertion behavior)
    └──requires──> [Kafka adapter] (adapter tests are part of the suite)
    └──enhances──> [all other features] (regression safety for every fix)

[Test dependency ordering enforced]
    └──requires──> [sortTestsByDependencies] (already implemented — just needs to be called in run.command.ts)

[Metadata cache]
    └──requires──> [run.command.ts filter logic] (simple Map addition)
```

### Dependency Notes

- **Assertion engine wired requires no new code** — `runAssertion` in assertion-runner.ts is complete and correct. The only work is replacing the stub in StepExecutor with a real call and deciding what `step.assert` shape maps to which runAssertion call.
- **Kafka adapter requires EventHub fix** — both share the `waitFor + processError → resolve` bug pattern. Fix them together or the same mistake will exist in two adapters.
- **Distinct 'warned' status feeds reporters** — all four reporters (console, JUnit, HTML, JSON) need to handle the new status value. JUnit XML maps `warned` to `<skipped>` with a message, which is the closest standard analog.
- **Unit test suite is a forcing function** — writing tests for the assertion engine will expose any remaining operator gaps. Writing tests for the orchestrator will expose the parallel state bug. Build suite after core fixes are in to get real coverage, not coverage of broken stubs.

---

## MVP Definition

This is a brownfield project — there is no "new MVP." The equivalent question is: what is the minimum set of fixes and additions that makes the framework trustworthy for real-world use?

### Ship With (this milestone's P0 work)

- [x] Assertion engine wired — replace StepExecutor.validateAssertions stub with actual assertion-runner calls. Without this, the entire assert surface is silently inert.
- [x] `continueOnError` reports `'warned'` not `'passed'` — CI tooling consuming JUnit XML or JSON reports cannot trust test results while failures masquerade as passes.
- [x] `retryCount` aggregated at test level — retry-count visibility is expected in any test reporting that claims to support retry semantics.
- [x] Test dependency ordering enforced — calling `sortTestsByDependencies` before execution is a one-line change with high correctness impact.
- [x] EventHub promise rejection fix — all promise-based adapter errors must reject, not resolve with failure data.
- [x] Kafka adapter — message queue testing support is a stated project requirement and the primary new feature of this milestone.
- [x] Unit test suite (Vitest) — this is the mechanism that prevents regressions in all the above fixes.

### Add After Core is Stable (P1 work)

- [ ] Parallel execution state safety — fix shared `currentTest`/`currentPhase` mutable state in TestOrchestrator. Triggers when `parallel > 1` and events report wrong test names.
- [ ] Unified capture logic — consolidate dead `captureValues` in StepExecutor with BaseAdapter implementation. Unblocks capture from non-HTTP adapters.
- [ ] Metadata cache — performance improvement. Trigger: filter-heavy runs feel slow on large test suites.
- [ ] Parallel config validation — warn when `parallel` exceeds PostgreSQL pool size. Prevent silent connection timeouts.

### Future Consideration (P2, if demand emerges)

- [ ] `notEquals` / `notContains` assertion operators — fill gaps in BaseAssertion interface vs matchers.ts. Low complexity, defer until a user reports the gap.
- [ ] `ts-node` / `tsx` in peerDependenciesMeta — improve discoverability for TypeScript test authors. Defer until TypeScript test adoption creates user reports.
- [ ] `ajv` missing warning — emit a log when schema validation is skipped. Trivially safe to ship but not blocking any real-world use case.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Assertion engine wired | HIGH — silent failures are the worst class of testing bug | LOW — existing code, just call it | P1 |
| continueOnError `'warned'` status | HIGH — correctness of test results | LOW — add status value, update reporters | P1 |
| retryCount aggregation | MEDIUM — reporting fidelity | LOW — sum step results in buildTestResult | P1 |
| Test dependency ordering | HIGH — flaky tests without it | LOW — one-line call in run.command.ts | P1 |
| EventHub promise rejection fix | HIGH — currently hides all EventHub errors | LOW — change resolve to reject in processError | P1 |
| Kafka adapter | HIGH — stated project requirement | HIGH — new adapter, produce/consume/assert/timeout | P1 |
| Unit test suite (Vitest) | HIGH — enables safe refactoring of all other work | HIGH — time investment, covers all modules | P1 |
| Parallel state safety | MEDIUM — affects parallel > 1 users | MEDIUM — refactor orchestrator context passing | P2 |
| Unified capture logic | MEDIUM — enables cross-adapter value capture chains | MEDIUM — audit and consolidate two implementations | P2 |
| Metadata cache | LOW — performance, not correctness | LOW — Map in run.command.ts | P2 |
| Parallel config validation | LOW — defensive guard | LOW — comparison + warn log | P2 |
| `notEquals`/`notContains` operators | LOW — niche use case | LOW — add to BaseAssertion + assertion-runner | P3 |
| `ts-node` peer dep documentation | LOW — discoverability | LOW — package.json change | P3 |

**Priority key:**
- P1: Must have for "battle-tested" milestone to be complete
- P2: Should have, improves reliability under load or edge cases
- P3: Nice to have, defer until user demand

---

## Competitor Feature Analysis

Compared against: Playwright (web E2E), REST-assured (Java API testing), Postman/Newman (YAML-like API runner), Hurl (HTTP-only YAML runner).

| Feature | REST-assured | Postman/Newman | Hurl | This Framework (target state) |
|---------|--------------|----------------|------|-------------------------------|
| JSONPath assertions | Yes, native | Yes, via scripts | Yes, native | Yes — JSONPath extraction exists; assertion engine must be wired |
| Soft assertions / continue-on-error with distinct status | Yes (SoftAssert) | No (fail-fast) | No | Yes — `'warned'` status with non-blocking flag |
| Test dependency ordering | No | Collection order | No | Yes — topological sort, just needs wiring |
| Message queue testing | No (HTTP only) | No | No | Yes — EventHub (fix needed), Kafka (new) |
| Database adapter testing | Via JDBC only | No | No | Yes — PostgreSQL, MongoDB, Redis natively |
| Retry count in reports | Per-request in logs | No | No | Yes — once aggregation is fixed |
| Parallel execution | Via JUnit parallel | Via Newman `--parallel` | No | Yes — p-limit based, needs state safety fix |
| YAML-first DSL | No (Java code) | Yes (collection JSON) | Yes | Yes — primary interface |
| TypeScript programmatic API | No | Via scripts | No | Yes — ts-loader with function steps |
| Unit-tested core | Yes | N/A | Yes | No — must build (Vitest) |

**Key insight:** No competitor combines YAML-declarative tests + database adapters + message queue testing in a single Node.js tool. That combination is the differentiating position. The gap between "has the feature" and "feature actually works" is what this milestone closes.

---

## Sources

- Direct codebase inspection: `src/core/step-executor.ts`, `src/assertions/assertion-runner.ts`, `src/assertions/matchers.ts`, `src/adapters/eventhub.adapter.ts`, `src/types.ts`, `src/core/test-orchestrator.ts`
- Project context: `.planning/PROJECT.md`, `.planning/codebase/CONCERNS.md`
- [KafkaJS Documentation — Consuming Messages](https://kafka.js.org/docs/consuming) — consume patterns, eachMessage/eachBatch, session timeout
- [KafkaJS Documentation — Testing](https://kafka.js.org/docs/testing) — test helpers, integration testing approach
- [Talend Cloud API Tester — Assertion Operators](https://help.qlik.com/talend/en-US/api-tester-user-guide/Cloud/assertion-operators) — complete assertion operator set for mature API testing tools
- [Soft Assertions with AssertJ | Baeldung](https://www.baeldung.com/java-assertj-soft-assertions) — industry pattern for soft/non-blocking assertions
- [How to Use Soft Asserts in TestNG? | GeeksforGeeks](https://www.geeksforgeeks.org/how-to-use-soft-asserts-in-testng/) — SoftAssert status handling pattern
- [Parallel Testing in Software Testing | ACCELQ](https://www.accelq.com/blog/parallel-testing/) — parallel execution state isolation requirements
- [JUnit 5 Parallel Execution: Thread-Safe Test Design Patterns](https://programgeeks.net/junit-5-parallel-execution-thread-safe-test-design-patterns/) — context passing vs shared mutable state
- [Testing Microservices: Message Isolation for Kafka](https://www.signadot.com/blog/testing-microservices-message-isolation-for-kafka-sqs-more) — message queue test isolation patterns
- [E2E Testing Best Practices, Reloaded | Kubernetes Contributors](https://www.kubernetes.dev/blog/2023/04/12/e2e-testing-best-practices-reloaded/) — structured failure messages, step recording
- [Testing Kafka and Spring Boot | Baeldung](https://www.baeldung.com/spring-boot-kafka-testing) — Awaitility/CountDownLatch pattern for async message assertions
- [Topological Sorting | Wikipedia](https://en.wikipedia.org/wiki/Topological_sorting) — Kahn's algorithm for dependency ordering

---

*Feature research for: E2E testing framework (battle-tested milestone)*
*Researched: 2026-03-02*
