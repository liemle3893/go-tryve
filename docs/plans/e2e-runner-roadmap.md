# E2E Runner — Master Implementation Roadmap

**Document Owner:** Architect  
**Created:** 2026-03-02  
**Status:** Proposed  
**Target Version:** v1.3.0

---

## Executive Summary

This roadmap synthesizes the Business PO and Technical PO analyses into a unified implementation plan. Both POs agree on a **foundation-first strategy**: fix critical bugs causing silent test failures and build unit test coverage before adding new features.

**Critical Convergence Points:**
1. ✅ Fix silent test failures (P0) — Biz PO Priority #1, Tech PO implicit requirement
2. ✅ Add unit test suite (P0) — Biz PO Priority #2, Tech PO Priority #5
3. ✅ Foundation fixes before feature expansion — Both POs explicitly agree

**Overall Strategy:** Quality over velocity. A reliable tool with fewer features beats a feature-rich tool with silent failures.

---

## Guiding Principles

1. **User Trust First** — Silent failures undermine the entire value proposition. Fix before any promotion.
2. **Test Coverage Enables Refactoring** — Zero coverage makes all future work risky.
3. **Incremental Delivery** — Each phase delivers working, tested code to main.
4. **Parallelize Where Safe** — Independent tasks can run concurrently; dependent tasks must be sequential.
5. **No Breaking Changes** — Existing YAML test files must continue to work.

---

## Phase Overview

| Phase | Name | Priority | Duration | Blocked By | Status |
|-------|------|----------|----------|------------|--------|
| 0 | Foundation Fixes | P0 (Critical) | 2-3 weeks | None | Ready to start |
| 1 | Technical Health | P0-P1 | 2-3 weeks | Phase 0 | Planning |
| 2 | Feature Expansion | P1 | 1-2 weeks | Phase 0 | Planning |
| 3 | Developer Experience | P1-P2 | 1-2 weeks | None (parallel) | Planning |

**Critical Path:** Phase 0 → Phase 1 → Phase 2

**Parallel Work:** Phase 3 can run concurrently with Phase 1 and Phase 2.

---

## Phase 0: Foundation Fixes

**Goal:** Every assertion actually runs, every test result accurately reflects what happened.

**Duration:** 2-3 weeks (40-60 hours)

**Dependencies:** None (can start immediately)

**Business Value (from Biz PO):**
- **User Trust:** HIGH — Silent failures cause production bugs
- **Market Reputation:** HIGH — Early adopters will abandon the tool if tests can't be trusted

**Technical Value (from Tech PO):**
- Eliminates critical bugs from ROADMAP.md Phase 1
- Enables reliable unit testing in Phase 1

### Tasks

#### Task 0.1: Wire Assertion Engine [independent]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 8-12 hours  
**Dependencies:** None

**Problem:** The assertion engine in `StepExecutor.validateAssertions` is a stub that never runs assertions. Tests with failing assertions silently pass.

**Files:**
- Modify: `src/core/step-executor.ts` — Replace stub with actual assertion-runner calls
- Test: `tests/unit/core/step-executor.test.ts`

**Acceptance Criteria:**
- [ ] A test with an `assert` block that should fail actually fails (not silent pass)
- [ ] All existing E2E tests continue to pass
- [ ] New unit test for assertion engine exists and passes

---

#### Task 0.2: Fix continueOnError Status [independent]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 4-6 hours  
**Dependencies:** None

**Problem:** Steps that fail with `continueOnError: true` are marked as `passed` instead of `warned`.

**Files:**
- Modify: `src/core/step-executor.ts` — Introduce `warned` status
- Modify: `src/reporters/console.reporter.ts` — Display `warned` status
- Modify: `src/reporters/html.reporter.ts` — Display `warned` status
- Modify: `src/types.ts` — Add `warned` to `StepStatus`
- Test: `tests/unit/core/step-executor.test.ts`

**Acceptance Criteria:**
- [ ] A step that fails with `continueOnError: true` shows status `warned` in all reporters
- [ ] A step that fails without `continueOnError` shows status `failed`
- [ ] New unit test exists and passes

---

#### Task 0.3: Fix EventHub Error Handling [independent]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 2-4 hours  
**Dependencies:** None

**Problem:** EventHub `processError` handler resolves promise instead of rejecting it.

**Files:**
- Modify: `src/adapters/eventhub.adapter.ts` — Reject promise on error
- Test: `tests/unit/adapters/eventhub.test.ts`

**Acceptance Criteria:**
- [ ] An EventHub infrastructure error causes the step to fail
- [ ] Existing E2E EventHub tests pass
- [ ] New unit test for error handling exists and passes

---

#### Task 0.4: Fix Test-Level Retry Count [independent]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 2-3 hours  
**Dependencies:** None

**Problem:** `retryCount` at test level is hardcoded to zero.

**Files:**
- Modify: `src/core/test-orchestrator.ts` — Derive from step results
- Test: `tests/unit/core/test-orchestrator.test.ts`

**Acceptance Criteria:**
- [ ] A test that retried steps reports actual total retry count
- [ ] Reporters display correct retry counts
- [ ] New unit test exists and passes

---

#### Task 0.5: Add TypeScript Adapter Type [independent]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 1-2 hours  
**Dependencies:** None

**Problem:** TypeScript function-backed steps use misleading adapter type `http`.

**Files:**
- Modify: `src/types.ts` — Add `typescript` to `AdapterType`
- Modify: `src/core/yaml-loader.ts` — Support `typescript` adapter
- Modify: `src/core/test-orchestrator.ts` — Handle `typescript` adapter
- Modify: `tests/e2e/adapters/*.test.yaml` — Update TypeScript tests

**Acceptance Criteria:**
- [ ] TypeScript test files declare `adapter: typescript`
- [ ] Existing TypeScript tests work
- [ ] Documentation updated

---

#### Task 0.6: Fix Redis KEYS Command [independent]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 3-4 hours  
**Dependencies:** None

**Problem:** `flushPattern` uses blocking `KEYS` command instead of `SCAN`.

**Files:**
- Modify: `src/adapters/redis.adapter.ts` — Replace `KEYS` with `SCAN` loop
- Test: `tests/unit/adapters/redis.test.ts`

**Acceptance Criteria:**
- [ ] `flushPattern` completes without issuing `KEYS` command
- [ ] Existing Redis E2E tests pass
- [ ] New unit test exists and passes

---

#### Task 0.7: Fix MongoDB ObjectId Import [independent]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 1-2 hours  
**Dependencies:** None

**Problem:** `ObjectId` imported on every operation instead of once at connect time.

**Files:**
- Modify: `src/adapters/mongodb.adapter.ts` — Import `ObjectId` once

**Acceptance Criteria:**
- [ ] Single `ObjectId` import at module level
- [ ] Existing MongoDB E2E tests pass

---

### Phase 0 Summary

**Total Effort:** 40-60 hours  
**All tasks are independent** — can be parallelized across multiple developers  
**Critical Path:** None

**Success Criteria:**
1. All P0 tasks completed and tested
2. No silent test failures
3. Accurate status reporting in all reporters
4. All existing E2E tests pass

**Rollback:**
```bash
git revert <phase-0-commit-range>
```
# Phase 1: Technical Health

**Goal:** Improve code quality, enable test coverage visibility, implement critical missing features.

**Duration:** 2-3 weeks (40-60 hours)

**Dependencies:** Phase 0 (foundation must be solid before testing it)

**Business Value (from Biz PO):**
- **Developer Velocity:** HIGH — Refactoring takes 3-5x longer without tests
- **Code Quality:** HIGH — Bugs slip through to production

**Technical Value (from Tech PO):**
- Improves maintainability score from B+ (7.5/10) to A- (8.5/10)

### Tasks

#### Task 1.1: Enable TypeScript Strict Mode [depends on: Phase 0]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 16-24 hours

**Problem:** TypeScript strict mode disabled, allowing type safety issues.

**Files:**
- Modify: `tsconfig.json` — Enable strict flags
- Modify: All 45 TypeScript source files — Fix compilation errors

**Acceptance Criteria:**
- [ ] All strict flags enabled
- [ ] Zero TypeScript compilation errors
- [ ] All existing tests pass

---

#### Task 1.2: Implement Test Coverage Reporting [depends on: Phase 0]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 8-16 hours

**Problem:** No coverage tooling configured.

**Files:**
- Modify: `vitest.config.ts` — Configure coverage
- Modify: `package.json` — Add coverage scripts
- Modify: `README.md` — Add coverage badge

**Acceptance Criteria:**
- [ ] Coverage report generated successfully
- [ ] Baseline coverage percentage documented
- [ ] Coverage badge in README

---

#### Task 1.3: Implement Lifecycle Hooks [depends on: Phase 0]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 4-6 hours

**Problem:** Lifecycle hooks not implemented.

**Files:**
- Modify: `src/core/config-loader.ts` — Load hooks
- Modify: `src/core/test-orchestrator.ts` — Execute hooks
- Create: `src/core/hook-loader.ts`
- Test: `tests/unit/core/hook-loader.test.ts`

**Acceptance Criteria:**
- [ ] All hooks (beforeAll, afterAll, beforeEach, afterEach) work
- [ ] Hook failures properly reported
- [ ] Unit tests pass

---

#### Task 1.4: Implement Test Dependency Ordering [depends on: Phase 0]

**Priority:** P0 (Critical)  
**Assigned to:** Backend Developer  
**Effort:** 3-4 hours

**Problem:** Test dependencies parsed but never enforced.

**Files:**
- Modify: `src/core/test-orchestrator.ts` — Call sort function
- Modify: `src/core/test-discovery.ts` — Topological sort
- Test: `tests/unit/core/test-discovery.test.ts`

**Acceptance Criteria:**
- [ ] Tests with `depends` execute in correct order
- [ ] Circular dependencies detected

---

#### Task 1.5: Add Unit Tests for Core Modules [depends on: 1.1, 1.2]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 40-80 hours

**Problem:** Only 2 unit test files for 45 source files.

**Files to create:**
- `tests/unit/core/test-orchestrator.test.ts`
- `tests/unit/core/test-discovery.test.ts`
- `tests/unit/core/variable-interpolator.test.ts`
- `tests/unit/core/yaml-loader.test.ts`
- `tests/unit/core/step-executor.test.ts`
- `tests/unit/core/config-loader.test.ts`

**Acceptance Criteria:**
- [ ] Core module coverage ≥ 70%
- [ ] All tests pass
- [ ] Edge cases covered

---

#### Task 1.6: Refactor html.reporter.ts [depends on: Phase 0]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 16-24 hours

**Problem:** Single file with 1044 lines.

**Files:**
- Modify: `src/reporters/html.reporter.ts`
- Create: `src/reporters/html/` directory with modules

**Acceptance Criteria:**
- [ ] No file exceeds 300 lines
- [ ] HTML output unchanged

---

### Phase 1 Summary

**Total Effort:** 88-154 hours (highly parallelizable)  
**Critical Path:** 1.1/1.2 → 1.5  
**Blocked By:** Phase 0

**Success Criteria:**
1. TypeScript strict mode enabled
2. Test coverage reporting configured
3. Lifecycle hooks implemented
4. Test dependency ordering enforced
5. Core module coverage ≥ 70%

**Rollback:**
```bash
git revert <phase-1-commit-range>
```

---

# Phase 2: Feature Expansion

**Goal:** Add Kafka adapter to expand market reach.

**Duration:** 1-2 weeks (16-24 hours)

**Dependencies:** Phase 0

**Business Value (from Biz PO):**
- **Market Reach:** HIGH — Kafka used by 80%+ of Fortune 100
- **Competitive Advantage:** HIGH — Unique positioning vs. API-only tools

### Tasks

#### Task 2.1: Implement Kafka Adapter [depends on: Phase 0]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 16-24 hours

**Files:**
- Create: `src/adapters/kafka.adapter.ts`
- Create: `tests/e2e/adapters/TC-KAFKA-001.test.yaml`
- Create: `tests/unit/adapters/kafka.test.ts`
- Modify: `package.json` — Add `kafkajs` peer dependency

**Acceptance Criteria:**
- [ ] Produce messages to Kafka topics
- [ ] Consume messages with content assertions
- [ ] `waitFor` pattern with timeout
- [ ] Connection failures fail the step
- [ ] E2E tests pass

---

### Phase 2 Summary

**Total Effort:** 16-24 hours  
**Blocked By:** Phase 0

**Rollback:**
```bash
git revert <phase-2-commit-range>
npm uninstall kafkajs
```

---

# Phase 3: Developer Experience (Parallel Track)

**Goal:** Improve developer workflow and adoption.

**Duration:** 1-2 weeks (10-12 hours)

**Dependencies:** None (can run parallel with Phase 1 and 2)

**Business Value (from Biz PO):**
- **Developer Velocity:** MEDIUM — 20-30% faster test iteration
- **Competitive Parity:** MEDIUM — Matches expectations from other frameworks

### Tasks

#### Task 3.1: Add Watch Mode [independent]

**Priority:** P1 (High)  
**Assigned to:** Backend Developer  
**Effort:** 3-4 hours

**Files:**
- Modify: `src/cli/run.ts`
- Create: `src/core/watcher.ts`

**Acceptance Criteria:**
- [ ] `e2e run --watch` re-runs tests on file changes
- [ ] Debounced file watching
- [ ] Clear console output between runs

---

#### Task 3.2: Add TypeScript Test DSL [independent]

**Priority:** P2 (Medium)  
**Assigned to:** Backend Developer  
**Effort:** 6-8 hours

**Files:**
- Create: `src/dsl/` directory
- Create: Fluent API builder

**Acceptance Criteria:**
- [ ] Fluent API: `test('name').description('...').execute(...)`
- [ ] Full TypeScript type inference
- [ ] Example tests converted
- [ ] Documentation updated

---

### Phase 3 Summary

**Total Effort:** 10-12 hours  
**All tasks independent** — can run parallel with Phase 1 and 2
# Dependency Graph

```
Phase 0 (Foundation)
    ├── Task 0.1 (independent)
    ├── Task 0.2 (independent)
    ├── Task 0.3 (independent)
    ├── Task 0.4 (independent)
    ├── Task 0.5 (independent)
    ├── Task 0.6 (independent)
    └── Task 0.7 (independent)

Phase 0 → Phase 1 (Technical Health)
              ├── Task 1.1 [depends on: Phase 0]
              ├── Task 1.2 [depends on: Phase 0]
              ├── Task 1.3 [depends on: Phase 0]
              ├── Task 1.4 [depends on: Phase 0]
              ├── Task 1.5 [depends on: 1.1, 1.2]
              └── Task 1.6 [depends on: Phase 0]

Phase 0 → Phase 2 (Feature Expansion)
              └── Task 2.1 [depends on: Phase 0]

Phase 3 (Developer Experience) — parallel track
    ├── Task 3.1 (independent)
    └── Task 3.2 (independent)
```

**Parallelization Opportunities:**
- All Phase 0 tasks can run concurrently (7 parallel streams)
- Phase 1.1, 1.2, 1.3, 1.4, 1.6 can run concurrently after Phase 0
- Phase 3 can run concurrently with Phase 1 and Phase 2
- Phase 2 can run concurrently with Phase 1 after Phase 0

---

# Success Metrics (6-Month Outlook)

| Metric | Current | 3-Month Target | 6-Month Target |
|--------|---------|----------------|----------------|
| **Test coverage** | ~15% | 65% | 80% |
| **Critical bugs** | ~15 | < 5 | < 3 |
| **Silent failures** | Yes | No | No |
| **TypeScript strict** | ❌ | ✅ | ✅ |
| **Lifecycle hooks** | ❌ | ✅ | ✅ |
| **Kafka adapter** | ❌ | ✅ | ✅ |
| **Watch mode** | ❌ | ✅ | ✅ |
| **npm weekly downloads** | Unknown | 100+ | 500+ |

---

# Risk Assessment

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **Phase 1 fixes take longer than estimated** | High | Medium | Cut scope to P0 tasks only; ship incremental fixes |
| **Unit test suite reveals more bugs** | Medium | High | Good! Fix them; document in CHANGELOG |
| **Kafka adapter complexity underestimated** | Medium | Low | Start with produce/consume only; add waitFor in v2 |
| **Watch mode causes performance issues** | Low | Low | Add debounce, limit file watching to test directories |
| **Strict mode reveals type errors** | Medium | High | Incremental enablement (one flag at a time) |

---

# Alignment with Existing Documentation

## Alignment with .planning/ROADMAP.md

| Roadmap Phase | This Document | Alignment |
|---------------|---------------|-----------|
| Phase 1: Foundation Fixes | Phase 0 | ✓ Direct alignment |
| Phase 2: Orchestrator Fixes | Phase 0 (partial), Phase 1 (partial) | ✓ Aligned |
| Phase 3: Kafka Adapter | Phase 2 | ✓ Direct alignment |
| Phase 4: Unit Test Suite | Phase 1 | ✓ Direct alignment |
| Phase 5: Adapter Polish | Phase 0 (Tasks 0.6, 0.7) | ✓ Aligned |

## Alignment with TODO.md

| TODO.md Item | This Document | Priority |
|--------------|---------------|----------|
| Lifecycle Hooks | Phase 1, Task 1.3 | P0 |
| Test Dependencies | Phase 1, Task 1.4 | P0 |
| Watch Mode | Phase 3, Task 3.1 | P1 |
| TypeScript DSL | Phase 3, Task 3.2 | P2 |

---

# Open Questions for Discussion

1. **Should we bundle commonly-used adapters (PostgreSQL, Redis) to reduce install friction?**
   - Pros: Simpler onboarding
   - Cons: Larger install size, forces dependencies users don't need
   - **Recommendation:** Keep peer dependency model for now

2. **Should we prioritize marketing activities (blog posts, conference talks) before or after Phase 0 fixes?**
   - Risk: Promoting a broken tool damages reputation
   - **Recommendation:** Wait until Phase 0 complete

3. **What's the minimum viable test coverage before promoting the tool externally?**
   - **Recommendation:** 60%+ for core modules, Phase 0 fixes complete

4. **Should we seek sponsorship or grants for Kafka adapter development?**
   - Potential partners: Confluent, companies using Kafka in production
   - **Recommendation:** Explore after Phase 0 complete

---

# Conclusion

This roadmap establishes a **foundation-first strategy** that addresses the critical convergence between Business and Technical priorities:

1. **Phase 0 (Foundation Fixes)** — Eliminate silent failures, ensure accurate test results
2. **Phase 1 (Technical Health)** — Enable strict mode, add test coverage, implement missing features
3. **Phase 2 (Feature Expansion)** — Add Kafka adapter for market expansion
4. **Phase 3 (Developer Experience)** — Add watch mode and TypeScript DSL (parallel track)

**Key Principle:** Quality over velocity. A reliable tool with fewer features beats a feature-rich tool with silent failures.

**Total Estimated Effort:** 154-250 hours (4-6 weeks with parallel execution)

**Next Steps:**
1. PM reviews and approves this roadmap
2. Architect creates detailed implementation plans for each phase
3. PM assigns tasks to Backend Developer
4. Track progress in project board

---

# References

- **Biz PO Analysis:** `docs/product-overview.md`, `docs/plans/biz-priorities.md`
- **Tech PO Analysis:** `docs/technical-assessment.md`, `docs/plans/tech-priorities.md`
- **Existing Roadmap:** `.planning/ROADMAP.md`
- **Requirements:** `.planning/REQUIREMENTS.md`
- **TODO List:** `TODO.md`
- **Architecture Docs:** `docs/architecture.md`

---

*This document was created by the Architect as a synthesis of Business PO and Technical PO analyses. For detailed implementation plans for individual tasks, see separate plan files in `docs/plans/`.*
