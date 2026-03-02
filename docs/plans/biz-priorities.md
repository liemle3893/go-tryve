# E2E Runner — Business Priorities & Roadmap Recommendations

**Document Owner:** Business Product Owner  
**Date:** 2026-03-02  
**Version:** 1.0

---

## Executive Summary

This document outlines the **top 5 business priorities** for E2E Runner based on market analysis, user persona needs, and competitive positioning. Each priority is evaluated for business impact, estimated effort, and strategic value.

**Core Strategy:** *Fix foundation before building features.* The framework has critical bugs causing silent test failures (see `.planning/PROJECT.md`). These must be resolved before adding new features or promoting the tool externally.

---

## Priority Framework

| Priority Level | Definition | Criteria |
|----------------|------------|----------|
| **P0 - Critical** | Blocks user trust; must fix immediately | Silent failures, data corruption, security issues |
| **P1 - High** | Significant competitive advantage or user pain point | Features that differentiate us from competitors |
| **P2 - Medium** | Improves developer experience or expands use cases | Nice-to-have that increases adoption |
| **P3 - Low** | Long-term strategic value | Future enhancements, edge cases |

---

## Top 5 Business Priorities

### 1. [P0] Fix Silent Test Failures — Foundation Trust

**Rationale:**  
Users cannot trust test results when the assertion engine is a stub (CORE-01), `continueOnError` marks failed steps as `passed` (CORE-02), and EventHub errors resolve instead of reject (CORE-04). This undermines the entire value proposition of E2E Runner.

**Business Impact:**
- **User Trust:** HIGH — Silent failures cause production bugs
- **Market Reputation:** HIGH — Early adopters will abandon the tool if tests can't be trusted
- **Support Burden:** MEDIUM — Users will report "tests pass but bugs occur"

**Scope Estimate:**
- **Files Affected:** `src/core/step-executor.ts`, `src/adapters/eventhub.adapter.ts`, `src/assertions/`
- **Effort:** 8-12 hours (already scoped in ROADMAP.md Phase 1)
- **Dependencies:** None (foundational fix)

**Success Metrics:**
- [ ] Test with failing assertion actually fails (not silent pass)
- [ ] Step with `continueOnError: true` shows `warned` status in reporters
- [ ] EventHub infrastructure error fails the step (not silent resolve)
- [ ] Unit tests for assertion engine pass

**Affected Areas:** Backend (core engine), QA (verification), Documentation (update reliability claims)

**Recommendation:** Execute ROADMAP.md Phase 1 immediately. No new features until this is complete.

---

### 2. [P0] Add Unit Test Suite — Refactoring Confidence

**Rationale:**  
E2E Runner has **0% unit test coverage** (`npm test` prints "No tests yet"). This makes refactoring risky and slows development velocity. Users cannot contribute safely without test coverage.

**Business Impact:**
- **Developer Velocity:** HIGH — Refactoring takes 3-5x longer without tests
- **Code Quality:** HIGH — Bugs slip through to production
- **Open Source Contributions:** MEDIUM — Contributors are hesitant without tests

**Scope Estimate:**
- **Files Affected:** New test files in `tests/unit/`, `vitest.config.ts`
- **Effort:** 20-30 hours (ROADMAP.md Phase 4)
- **Dependencies:** Phase 1-3 fixes complete (don't test broken code)

**Success Metrics:**
- [ ] `npm test` runs full suite and exits 0
- [ ] Coverage report shows 85%+ on `src/core/`, `src/assertions/`, `src/adapters/`
- [ ] Parallel execution state safety verified by concurrent tests
- [ ] All assertion matchers have happy path + failure case tests

**Affected Areas:** Backend (all modules), QA (automated regression), DevOps (CI integration)

**Recommendation:** Prioritize after Phase 1 fixes. Tests for core modules first, adapters second.

---

### 3. [P1] Add Kafka Adapter — Market Expansion

**Rationale:**  
Message queue testing is a key differentiator vs. Postman/Playwright. E2E Runner currently supports Azure EventHub but not Kafka (the dominant message broker). Adding Kafka expands the addressable market significantly.

**Business Impact:**
- **Market Reach:** HIGH — Kafka is used by 80%+ of Fortune 100 companies
- **Competitive Advantage:** HIGH — Unique positioning vs. API-only tools
- **User Adoption:** MEDIUM — Attracts teams building event-driven architectures

**Scope Estimate:**
- **Files Affected:** New `src/adapters/kafka.adapter.ts`, docs update
- **Effort:** 16-24 hours (ROADMAP.md Phase 3)
- **Dependencies:** Phase 1 complete (don't build on broken foundation)

**Success Metrics:**
- [ ] Produce messages to Kafka topics
- [ ] Consume messages with content assertions
- [ ] `waitFor` pattern with timeout and pattern matching
- [ ] Connection failures fail the step (not silent resolve)
- [ ] E2E tests for Kafka adapter in `tests/e2e/adapters/`

**Affected Areas:** Backend (new adapter), Documentation (usage guide), QA (integration tests)

**Recommendation:** High strategic value. Prioritize after foundation fixes. Consider sponsoring or partnering with Kafka-using companies for case studies.

---

### 4. [P1] Add Watch Mode — Developer Experience

**Rationale:**  
Modern test frameworks (Jest, Vitest, Playwright) have watch mode for rapid iteration. E2E Runner lacks this, forcing developers to manually re-run tests after changes. This slows development velocity and reduces adoption.

**Business Impact:**
- **Developer Velocity:** MEDIUM — 20-30% faster test iteration
- **Competitive Parity:** MEDIUM — Matches expectations from other frameworks
- **User Satisfaction:** MEDIUM — Reduces friction in daily workflow

**Scope Estimate:**
- **Files Affected:** `src/cli/run.ts`, new `src/core/watcher.ts`
- **Effort:** 3-4 hours (TODO.md item #3)
- **Dependencies:** None (independent feature)
- **Risk:** LOW — Optional feature, doesn't break existing workflows

**Success Metrics:**
- [ ] `e2e run --watch` re-runs tests on file changes
- [ ] Debounced file watching (500ms stability threshold)
- [ ] Clear console output between runs
- [ ] "Watching for changes..." status message
- [ ] Smart test selection (only run affected tests)

**Affected Areas:** CLI (new flag), Backend (watcher module), Documentation (usage guide)

**Recommendation:** Quick win for developer experience. Can be parallelized with Phase 1 fixes.

---

### 5. [P2] Add TypeScript Test DSL — Developer Experience

**Rationale:**  
Current TypeScript support is basic (function exports). A fluent DSL would attract developers who prefer code over YAML and enable better IDE support (autocompletion, type checking).

**Business Impact:**
- **Developer Experience:** MEDIUM — Better IDE support, type safety
- **Market Reach:** MEDIUM — Attracts TypeScript-first teams
- **Learning Curve:** LOW — Developers already know TypeScript

**Scope Estimate:**
- **Files Affected:** New `src/dsl/` directory, TypeScript loader updates
- **Effort:** 6-8 hours (TODO.md item #6)
- **Dependencies:** None (independent feature)
- **Risk:** LOW — Optional, doesn't affect YAML users

**Success Metrics:**
- [ ] Fluent API: `test('name').description('...').execute(async (ctx) => ...)`
- [ ] Full TypeScript type inference for test context
- [ ] Autocompletion in VS Code / WebStorm
- [ ] Example tests converted to TypeScript DSL
- [ ] Documentation updated with TypeScript examples

**Affected Areas:** Backend (new DSL module), Documentation (TypeScript guide), Examples (conversions)

**Recommendation:** Medium priority. Nice-to-have for TypeScript teams, but not blocking for YAML users.

---

## Deferred Priorities (Not Top 5)

| Priority | Feature | Reason for Deferral |
|----------|---------|---------------------|
| P2 | HTTP Traffic Capture | Nice for debugging, but reporters already provide value |
| P2 | Step-by-Step Interactive Mode | Low demand, advanced debugging use case |
| P2 | Report History & Trends | Requires database setup, adds complexity |
| P3 | GraphQL Adapter | Niche use case, not blocking for target personas |
| P3 | gRPC Adapter | Niche use case, not blocking for target personas |
| P3 | Plugin System | Premature optimization, wait for community demand |

---

## Implementation Roadmap Recommendation

### Q1 2026 (Immediate Focus)
1. **Week 1-2:** Complete ROADMAP.md Phase 1 (Foundation Fixes)
   - Wire assertion engine
   - Fix `continueOnError` status
   - Fix EventHub error handling
   - Remove dead code
   
2. **Week 3-4:** Begin Phase 4 (Unit Test Suite)
   - Set up Vitest
   - Write tests for `src/core/` modules
   - Write tests for `src/assertions/` modules

### Q2 2026 (Feature Expansion)
3. **Week 5-6:** Phase 3 (Kafka Adapter)
   - Implement produce/consume/waitFor actions
   - Add E2E tests
   - Update documentation

4. **Week 7:** Watch Mode (parallel work possible)
   - Implement file watcher with chokidar
   - Integrate with CLI

### Q3 2026 (Polish & Growth)
5. **Week 8-10:** Continue Phase 4 (Unit Test Suite)
   - Achieve 85%+ coverage
   - Add adapter tests
   
6. **Week 11-12:** TypeScript DSL (if time permits)
   - Design fluent API
   - Implement builder pattern
   - Add examples

---

## Success Metrics (6-Month Outlook)

| Metric | Current | 6-Month Target | Measurement |
|--------|---------|----------------|-------------|
| **npm weekly downloads** | Unknown | 200+ | `npm stats @liemle3893/e2e-runner` |
| **GitHub stars** | Unknown | 300+ | GitHub Insights |
| **Test coverage** | 0% | 85%+ | `npm run coverage` |
| **Critical bugs open** | ~15 | < 3 | GitHub Issues |
| **Documentation completeness** | Good | Excellent | User feedback, time-to-first-test |
| **Contributor count** | 1 | 5+ | GitHub Contributors |

---

## Risk Assessment

| Risk | Mitigation Strategy |
|------|---------------------|
| **Phase 1 fixes take longer than estimated** | Cut scope to CORE-01/CORE-02 only; ship incremental fixes |
| **Unit test suite reveals more bugs** | Good! Fix them; document in CHANGELOG |
| **Kafka adapter complexity underestimated** | Start with produce/consume only; add waitFor in v2 |
| **Watch mode causes performance issues** | Add debounce, limit file watching to test directories |
| **Community doesn't adopt the tool** | Focus on blog posts, example repos, and conference talks |

---

## Open Questions for Discussion

1. **Should we bundle commonly-used adapters (PostgreSQL, Redis) to reduce install friction?**
   - Pros: Simpler onboarding
   - Cons: Larger install size, forces dependencies users don't need
   
2. **Should we prioritize marketing activities (blog posts, conference talks) before or after Phase 1 fixes?**
   - Risk: Promoting a broken tool damages reputation
   - Opportunity: Early feedback from early adopters

3. **Should we seek sponsorship or grants for Kafka adapter development?**
   - Potential partners: Confluent, companies using Kafka in production

4. **What's the minimum viable test coverage before promoting the tool externally?**
   - Recommendation: 60%+ for core modules, Phase 1 fixes complete

---

## Conclusion

The immediate priority is **fixing the foundation** (Phase 1) and **building test coverage** (Phase 4). These are prerequisites for sustainable growth and user trust. After the foundation is solid, expand market reach with **Kafka adapter** and improve developer experience with **watch mode** and **TypeScript DSL**.

**Key Principle:** *Quality over velocity.* A reliable tool with fewer features beats a feature-rich tool with silent failures.

---

## Appendix: Alignment with Existing Roadmap

| Business Priority | ROADMAP.md Phase | Alignment |
|-------------------|------------------|-----------|
| Fix Silent Failures | Phase 1 | ✓ Direct alignment |
| Unit Test Suite | Phase 4 | ✓ Direct alignment |
| Kafka Adapter | Phase 3 | ✓ Direct alignment |
| Watch Mode | N/A (TODO.md) | New recommendation |
| TypeScript DSL | N/A (TODO.md) | New recommendation |

**Note:** This business priorities document complements (not replaces) the technical roadmap in `.planning/ROADMAP.md`. Execution should follow the phased approach defined by the Architect.

---

*This document was created by the Business Product Owner. For technical implementation details, consult `.planning/ROADMAP.md` and `.planning/REQUIREMENTS.md`.*
