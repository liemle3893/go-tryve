# Roadmap: E2E Runner — Feature Complete (v1.3.0)

## Overview

Five phases transform an architecturally sound but partially broken framework into a battle-tested tool. Phase 1 fixes silent failures (dead assertion engine, false pass status, zero retry counts, EventHub error swallowing) and cleans up code quality debt. Phase 2 eliminates parallel state corruption and enforces test dependency ordering. Phase 3 adds Kafka message queue support — the only new surface area in this milestone. Phase 4 writes the full Vitest unit test suite after the fixes are in place. Phase 5 polishes adapter internals for correctness under load.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Foundation Fixes** - Wire the assertion engine, correct status reporting, and remove dead code
- [ ] **Phase 2: Orchestrator Fixes** - Eliminate parallel state corruption and enforce test dependency ordering
- [ ] **Phase 3: Kafka Adapter** - Add produce/consume/waitFor support via KafkaJS
- [ ] **Phase 4: Unit Test Suite** - Build 85%+ coverage with Vitest after all fixes are in place
- [ ] **Phase 5: Adapter Polish** - SCAN-based Redis iteration, ObjectId import fix, adapter correctness guards

## Phase Details

### Phase 1: Foundation Fixes
**Goal**: Every assertion actually runs, every test result accurately reflects what happened, and dead code is removed
**Depends on**: Nothing (first phase)
**Requirements**: CORE-01, CORE-02, CORE-03, CORE-04, ADPT-01, QUAL-01, QUAL-02, QUAL-03, QUAL-04
**Success Criteria** (what must be TRUE):
  1. A test with an `assert` block that should fail actually fails — not silently passes
  2. A step that fails with `continueOnError: true` shows status `warned` in all reporter outputs, not `passed`
  3. A test that retried steps reports the actual total retry count at the test level, not zero
  4. An EventHub infrastructure error causes the step to fail, not silently resolve as passed
  5. TypeScript function-backed steps declare adapter type `typescript` in test definitions, not `http`
**Plans**: TBD

### Phase 2: Orchestrator Fixes
**Goal**: Parallel test runs produce correct event attribution and tests with `depends` execute in the declared order
**Depends on**: Phase 1
**Requirements**: EXEC-01, EXEC-02, EXEC-03
**Success Criteria** (what must be TRUE):
  1. Running 5 tests in parallel produces `test:start` and `test:end` events where each event carries the correct test name
  2. A test declaring `depends: [other-test]` always executes after `other-test` completes, regardless of file discovery order
  3. A hook file referenced with a relative path in `e2e.config.yaml` resolves correctly when the CLI is invoked from a different working directory
**Plans**: TBD

### Phase 3: Kafka Adapter
**Goal**: Test suites can produce messages to Kafka topics and assert on consumed message content via the standard step DSL
**Depends on**: Phase 1
**Requirements**: ADPT-04, ADPT-05, ADPT-06
**Success Criteria** (what must be TRUE):
  1. A test step with `adapter: kafka` and `action: produce` successfully publishes a message to a topic
  2. A test step with `adapter: kafka` and `action: consume` receives a message and exposes its content for JSONPath assertion
  3. A test step with `adapter: kafka` and `action: waitFor` resolves when a matching message arrives within the configured timeout, and fails the step when timeout is exceeded
  4. Kafka adapter errors (connection failure, timeout) fail the step rather than silently resolving as passed
**Plans**: TBD

### Phase 4: Unit Test Suite
**Goal**: Core modules, adapters, and the CLI have 85%+ unit test coverage verified by Vitest, enabling safe refactoring
**Depends on**: Phase 2, Phase 3
**Requirements**: QUAL-05, QUAL-06, QUAL-07, QUAL-08, QUAL-09
**Success Criteria** (what must be TRUE):
  1. `npm test` runs the full suite and exits 0 when all tests pass
  2. `npm run coverage` produces a coverage report showing 85%+ on `src/assertions/`, `src/core/`, and each adapter file
  3. The parallel state safety fix is verified by a test that runs 5 concurrent test executions and asserts each `test:start` event carries the correct test name
  4. All assertion matchers (`toBe`, `toEqual`, `toContain`, `toMatch`, `greaterThan`, etc.) have passing unit tests covering happy path and failure cases
**Plans**: TBD

### Phase 5: Adapter Polish
**Goal**: Redis and MongoDB adapters operate correctly under real-world workloads without blocking commands or per-operation import overhead
**Depends on**: Phase 1
**Requirements**: ADPT-02, ADPT-03
**Success Criteria** (what must be TRUE):
  1. Redis `flushPattern` with a wildcard pattern completes without issuing a `KEYS` command (verified by examining the command log or adapter code path)
  2. MongoDB adapter connects and executes operations without importing `ObjectId` on each operation — import happens once at connect time
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5
Note: Phase 2 and Phase 3 depend on Phase 1 but are independent of each other. Phase 5 depends on Phase 1 and is independent of Phases 2-4.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation Fixes | 0/TBD | Not started | - |
| 2. Orchestrator Fixes | 0/TBD | Not started | - |
| 3. Kafka Adapter | 0/TBD | Not started | - |
| 4. Unit Test Suite | 0/TBD | Not started | - |
| 5. Adapter Polish | 0/TBD | Not started | - |
